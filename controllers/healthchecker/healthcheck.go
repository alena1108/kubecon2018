package healthchecker

import (
	"github.com/sirupsen/logrus"

	types "github.com/rancher/kubecon2018/pkg/apis/clusterprovisioner/v1alpha1"
	client "github.com/rancher/kubecon2018/pkg/client/clientset/versioned"
	clusterclient "github.com/rancher/kubecon2018/pkg/client/clientset/versioned"
	informers "github.com/rancher/kubecon2018/pkg/client/informers/externalversions"
	listers "github.com/rancher/kubecon2018/pkg/client/listers/clusterprovisioner/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
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
		UpdateFunc: controller.handleClusterUpdate,
	})
	stop := make(chan struct{})
	go controller.clusterInformer.Run(stop)
	logrus.Infof("Registered %s controller", controller.getName())
}

func (c *Controller) getName() string {
	return "healthchecker"
}

func (c *Controller) handleClusterUpdate(obj interface{}, updated interface{}) {
	c.sync(updated)
}

func (c *Controller) sync(obj interface{}) {
	cluster := obj.(*types.Cluster)
	// skip non provisioned clusters
	if !types.ClusterConditionProvisioned.IsTrue(cluster) {
		return
	}

	toUpdate, err := types.ClusterConditionReady.Do(cluster, func() (runtime.Object, error) {
		return cluster, c.validateHealthcheck(cluster)
	})
	if err != nil {
		logrus.Errorf("Failed to validate healthcheck on cluster %s %v", cluster.Name, err)
	}

	for i := 0; i < 3; i++ {
		_, err = c.clusterClient.ClusterprovisionerV1alpha1().Clusters().Update(toUpdate.(*types.Cluster))
		if err == nil {
			break
		}
	}
	if err != nil {
		logrus.Debugf("Failed to update cluster %s %v", cluster.Name, err)
	}
}

func (c *Controller) validateHealthcheck(cluster *types.Cluster) error {
	kubeConfig, err := c.clusterClient.ClusterprovisionerV1alpha1().Kubeconfigs().Get(cluster.Name, v1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	configPath := kubeConfig.Spec.ConfigPath
	restConfig, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		return err
	}
	client, err := client.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	//healthcheck passes if we can contact underlying cluster
	_, err = client.Discovery().ServerVersion()
	return err
}
