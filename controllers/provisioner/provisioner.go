package provisioner

import (
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"

	types "github.com/rancher/kubecon2018/pkg/apis/clusterprovisioner/v1alpha1"
	clusterclient "github.com/rancher/kubecon2018/pkg/client/clientset/versioned"
	informers "github.com/rancher/kubecon2018/pkg/client/informers/externalversions"
	listers "github.com/rancher/kubecon2018/pkg/client/listers/clusterprovisioner/v1alpha1"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

type Controller struct {
	clusterLister   listers.ClusterLister
	clusterInformer cache.SharedIndexInformer
	clusterClient   clusterclient.Interface
}

func Register(
	clusterClient clusterclient.Interface,
	sampleInformerFactory informers.SharedInformerFactory) {
	clusterInformer := sampleInformerFactory.Clusterprovisioner().V1alpha1().Clusters()

	controller := &Controller{
		clusterLister:   clusterInformer.Lister(),
		clusterInformer: clusterInformer.Informer(),
		clusterClient:   clusterClient,
	}
	controller.clusterInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.clusterAdd,
		UpdateFunc: controller.clusterUpdate,
		DeleteFunc: controller.clusterRemove,
	})
	stop := make(chan struct{})
	go controller.clusterInformer.Run(stop)
	logrus.Infof("Registered %s controller", controller.getName())
}

func (c *Controller) getName() string {
	return "provisioner"
}

func (c *Controller) clusterAdd(obj interface{}) {
	cluster := obj.(*types.Cluster)
	if cluster.DeletionTimestamp != nil {
		c.handleClusterRemove(cluster)
	} else {
		if err := c.handleClusterAdd(cluster); err != nil {
			logrus.Errorf("Failed to provision cluster %s %v", cluster.Name, err)
		}
	}
}

func (c *Controller) clusterUpdate(obj interface{}, newObj interface{}) {
	c.clusterAdd(newObj)
}

func (c *Controller) clusterRemove(obj interface{}) {
	c.handleClusterRemove(obj.(*types.Cluster))
}

func (c *Controller) handleClusterRemove(cluster *types.Cluster) {
	logrus.Infof("Removing cluser %v", cluster.Name)
	if err := c.finalize(cluster, c.getName()); err != nil {
		logrus.Errorf("Error removing cluster %s %v", cluster.Name, err)
	} else {
		logrus.Infof("Successfully removed cluster %v", cluster.Name)
	}
}

func (c *Controller) handleClusterAdd(cluster *types.Cluster) error {
	config, err := getConfigStr(cluster)
	if err != nil {
		return err
	}
	if config == cluster.Status.AppliedConfig {
		return nil
	}
	logrus.Infof("Cluster [%s] is updated; provisioning...", cluster.Name)
	if err := c.initialize(cluster, c.getName()); err != nil {
		return fmt.Errorf("error initializing cluster %s %v", cluster.Name, err)
	}

	_, err = types.ClusterConditionProvisioned.Do(cluster, func() (runtime.Object, error) {
		return cluster, provisionCluster(cluster)
	})

	if err != nil {
		return fmt.Errorf("error provisioning cluster %s %v", cluster.Name, err)
	}
	if err := c.updateAppliedConfig(cluster, config); err != nil {
		return fmt.Errorf("error updating cluster %s %v", cluster.Name, err)
	}
	logrus.Infof("Successfully provisioned cluster %v", cluster.Name)
	return nil
}

func removeCluster(cluster *types.Cluster) (err error) {
	cmdName := "/Users/alena/go/src/github.com/rancher/rke/rke"
	cmdArgs := []string{"remove", "--force", "--config", cluster.Spec.ConfigPath}
	return executeCommand(cmdName, cmdArgs)
}

func executeCommand(cmdName string, cmdArgs []string) (err error) {
	cmd := exec.Command(cmdName, cmdArgs...)
	var stdout io.ReadCloser
	stdout, err = cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("error getting stdout from cmd '%v' %v", cmd, err)
	}
	if err = cmd.Start(); err != nil {
		return fmt.Errorf("error starting cmd '%v' %v", cmd, err)
	}
	defer func() {
		err = cmd.Wait()
	}()
	printLogs(stdout)
	return err
}

func provisionCluster(cluster *types.Cluster) (err error) {
	cmdName := "/Users/alena/go/src/github.com/rancher/rke/rke"
	cmdArgs := []string{"up", "--config", cluster.Spec.ConfigPath}
	return executeCommand(cmdName, cmdArgs)
}

func getConfigStr(cluster *types.Cluster) (string, error) {
	b, err := ioutil.ReadFile(cluster.Spec.ConfigPath)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func printLogs(r io.Reader) {
	buf := make([]byte, 80)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			fmt.Print(string(buf[0:n]))
		}
		if err != nil {
			break
		}
	}
}

func containsString(slice []string, item string) bool {
	for _, j := range slice {
		if j == item {
			return true
		}
	}
	return false
}

func (c *Controller) initialize(cluster *types.Cluster, finalizerKey string) error {
	//set finalizers
	metadata, err := meta.Accessor(cluster)
	if err != nil {
		return err
	}
	if containsString(metadata.GetFinalizers(), finalizerKey) {
		return nil
	}
	finalizers := metadata.GetFinalizers()
	finalizers = append(finalizers, finalizerKey)
	metadata.SetFinalizers(finalizers)
	for i := 0; i < 3; i++ {
		_, err = c.clusterClient.ClusterprovisionerV1alpha1().Clusters().Update(cluster)
		if err == nil {
			return err
		}
	}
	return nil
}

func (c *Controller) updateAppliedConfig(cluster *types.Cluster, config string) error {
	cluster.Status.AppliedConfig = config
	for i := 0; i < 3; i++ {
		_, err := c.clusterClient.ClusterprovisionerV1alpha1().Clusters().Update(cluster)
		if err == nil {
			return nil
		}
	}
	return nil
}

func (c *Controller) finalize(cluster *types.Cluster, finalizerKey string) error {
	toUpdate, err := c.clusterClient.ClusterprovisionerV1alpha1().Clusters().Get(cluster.Name, v1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	metadata, err := meta.Accessor(toUpdate)
	if err != nil {
		return err
	}
	// Check finalizer
	if metadata.GetDeletionTimestamp() == nil {
		return nil
	}

	if !containsString(metadata.GetFinalizers(), finalizerKey) {
		return nil
	}

	//run deletion hook
	if err = removeCluster(cluster); err != nil {
		return err
	}

	// remove finalizer
	var finalizers []string
	for _, finalizer := range metadata.GetFinalizers() {
		if finalizer == finalizerKey {
			continue
		}
		finalizers = append(finalizers, finalizer)
	}
	metadata.SetFinalizers(finalizers)

	for i := 0; i < 3; i++ {
		_, err = c.clusterClient.ClusterprovisionerV1alpha1().Clusters().Update(toUpdate)
		if err == nil {
			break
		}
	}

	return err
}
