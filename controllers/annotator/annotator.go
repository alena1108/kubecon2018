package annotator

import (
	"github.com/sirupsen/logrus"

	types "github.com/rancher/kubecon2018/pkg/apis/clusterprovisioner/v1alpha1"
	client "github.com/rancher/kubecon2018/pkg/client/clientset/versioned"
	clusterclient "github.com/rancher/kubecon2018/pkg/client/clientset/versioned"
	informers "github.com/rancher/kubecon2018/pkg/client/informers/externalversions"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	kubernetesVersionAnnotation = "clusterprovisioner.rke.io/kubernetes-version"
)

type Controller struct {
	clusterInformer cache.SharedIndexInformer
	clusterClient   clusterclient.Interface
}

func Register(kubeconfigClient clusterclient.Interface,
	sampleInformerFactory informers.SharedInformerFactory) {
	controller := &Controller{
		clusterInformer: sampleInformerFactory.Clusterprovisioner().V1alpha1().Clusters().Informer(),
		clusterClient:   kubeconfigClient,
	}
	controller.clusterInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.addAnnotation,
		UpdateFunc: controller.updateAnnotation,
	})
	stop := make(chan struct{})
	go controller.clusterInformer.Run(stop)
	logrus.Infof("Registered %s controller", controller.getName())
}

func (c *Controller) getName() string {
	return "annotator"
}

func (c *Controller) addAnnotation(obj interface{}) {
	cluster := obj.(*types.Cluster)
	c.sync(cluster)
}

func (c *Controller) updateAnnotation(obj interface{}, updated interface{}) {
	cluster := obj.(*types.Cluster)
	c.sync(cluster)
}

func (c *Controller) sync(cluster *types.Cluster) {
	if cluster.DeletionTimestamp != nil {
		return
	}
	if !types.ClusterConditionProvisioned.IsTrue(cluster) {
		return
	}
	if !types.ClusterConditionReady.IsTrue(cluster) {
		return
	}

	version, err := c.getVersion(cluster)
	if err != nil {
		logrus.Errorf("Failed to fetch cluster %s version %v", cluster.Name, err)
	}

	currentVersion := ""
	if cluster.Annotations != nil {
		currentVersion = cluster.Annotations[kubernetesVersionAnnotation]
	}
	if currentVersion == version {
		return
	}

	toUpdate := cluster.DeepCopy()
	if toUpdate.Annotations == nil {
		toUpdate.Annotations = map[string]string{}
	}
	toUpdate.Annotations[kubernetesVersionAnnotation] = version

	for i := 0; i < 3; i++ {
		_, err = c.clusterClient.ClusterprovisionerV1alpha1().Clusters().Update(toUpdate)
		if err == nil {
			break
		}
	}
	if err != nil {
		logrus.Debugf("Failed to update cluster %s %v", cluster.Name, err)
	}

}

func (c *Controller) getVersion(cluster *types.Cluster) (string, error) {
	kubeConfig, err := c.clusterClient.ClusterprovisionerV1alpha1().Kubeconfigs().Get(cluster.Name, v1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", nil
		}
		return "", err
	}
	configPath := kubeConfig.Spec.ConfigPath
	restConfig, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		return "", err
	}
	client, err := client.NewForConfig(restConfig)
	if err != nil {
		return "", err
	}
	//healthcheck passes if we can contact underlying cluster
	version, err := client.Discovery().ServerVersion()
	if err != nil {
		return "", err
	}
	return version.String(), nil
}
