package provisioner

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/Sirupsen/logrus"
	types "github.com/rancher/kubecon2018/pkg/apis/clusterprovisioner/v1alpha1"
	clusterclient "github.com/rancher/kubecon2018/pkg/client/clientset/versioned"
	informers "github.com/rancher/kubecon2018/pkg/client/informers/externalversions"
	listers "github.com/rancher/kubecon2018/pkg/client/listers/clusterprovisioner/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
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
		AddFunc:    controller.sync,
		DeleteFunc: controller.sync,
	})
	stop := make(chan struct{})
	go controller.clusterInformer.Run(stop)
	logrus.Info("Registered provisioner controller")
}

func (c *Controller) getName() string {
	return "provisioner"
}

func (c *Controller) sync(obj interface{}) {
	cluster := obj.(*types.Cluster)
	if cluster.DeletionTimestamp != nil {
		c.handleClusterRemove(cluster)
	} else {
		c.handleClusterAdd(cluster)
	}
}

func (c *Controller) handleClusterRemove(cluster *types.Cluster) {
	logrus.Infof("Removing cluser %v", cluster.Name)
	if err := c.finalize(cluster, c.getName()); err != nil {
		logrus.Errorf("Error removing cluster %s %v", cluster.Name, err)
	} else {
		logrus.Infof("Successfully removed cluster %v", cluster.Name)
	}
}

func (c *Controller) handleClusterAdd(cluster *types.Cluster) {
	logrus.Infof("Cluster [%s] is added; provisioning...", cluster.Name)
	if err := c.initialize(cluster, c.getName()); err != nil {
		logrus.Errorf("Error initializing cluster %s %v", cluster.Name, err)
	}

	if err := provisionCluster(cluster); err != nil {
		logrus.Errorf("Error provisioning cluster %s %v", cluster.Name, err)
	} else {
		logrus.Infof("Successfully provisioned cluster %v", cluster.Name)
	}
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

func (c *Controller) finalize(cluster *types.Cluster, finalizerKey string) error {
	metadata, err := meta.Accessor(cluster)
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

	// run deletion hook
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
		_, err = c.clusterClient.ClusterprovisionerV1alpha1().Clusters().Update(cluster)
		if err == nil {
			return err
		}
	}

	return err
}
