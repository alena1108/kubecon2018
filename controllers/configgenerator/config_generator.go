package configgenerator

import (
	"strings"

	"fmt"

	types "github.com/rancher/kubecon2018/pkg/apis/clusterprovisioner/v1alpha1"
	kubeconfigclient "github.com/rancher/kubecon2018/pkg/client/clientset/versioned"
	informers "github.com/rancher/kubecon2018/pkg/client/informers/externalversions"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

type Controller struct {
	clusterInformer  cache.SharedIndexInformer
	kubeconfigClient kubeconfigclient.Interface
}

func Register(kubeconfigClient kubeconfigclient.Interface,
	sampleInformerFactory informers.SharedInformerFactory) {
	controller := &Controller{
		clusterInformer:  sampleInformerFactory.Clusterprovisioner().V1alpha1().Clusters().Informer(),
		kubeconfigClient: kubeconfigClient,
	}
	controller.clusterInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.addConfig,
		UpdateFunc: controller.updateConfig,
	})
	stop := make(chan struct{})
	go controller.clusterInformer.Run(stop)
	logrus.Infof("Registered %s controller", controller.getName())
}

func (c *Controller) getName() string {
	return "configgenerator"
}

func (c *Controller) sync(cluster *types.Cluster) {
	if cluster.DeletionTimestamp != nil {
		return
	}
	if !types.ClusterConditionProvisioned.IsTrue(cluster) {
		return
	}
	kubeconfig, err := c.kubeconfigClient.ClusterprovisionerV1alpha1().Kubeconfigs().Get(cluster.Name, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		logrus.Errorf("Failed to fetch kubeconfig by name %s %v", cluster.Name, err)
		return
	}
	path := getKubeConfigPath(cluster)

	if apierrors.IsNotFound(err) || kubeconfig == nil {
		//create
		createKubeconfig(cluster, path, c)
	} else if kubeconfig.Spec.ConfigPath != path {
		updateKubeconfig(kubeconfig, path, c, cluster)
	}
}

func createKubeconfig(cluster *types.Cluster, path string, c *Controller) {
	controller := true
	ownerRef := metav1.OwnerReference{
		Name:       cluster.Name,
		APIVersion: "v1alpha1",
		UID:        cluster.UID,
		Kind:       "Cluster",
		Controller: &controller,
	}
	kubeconfig := &types.Kubeconfig{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{ownerRef},
			Name:            cluster.Name,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Kubeconfig",
			APIVersion: "clusterprovisioner.rke.io/v1alpha1",
		},
		Spec: types.KubeconfigSpec{
			ConfigPath: path,
		},
	}
	_, err := c.kubeconfigClient.ClusterprovisionerV1alpha1().Kubeconfigs().Create(kubeconfig)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return
		}
		logrus.Errorf("Failed to create kubeconfig for cluster %s %v", cluster.Name, err)
	}
}
func updateKubeconfig(kubeconfig *types.Kubeconfig, path string, c *Controller, cluster *types.Cluster) {
	toUpdate := kubeconfig.DeepCopy()
	toUpdate.Spec.ConfigPath = path
	_, err := c.kubeconfigClient.ClusterprovisionerV1alpha1().Kubeconfigs().Update(toUpdate)
	if err != nil {
		logrus.Errorf("Failed to update kubeconfig for cluster %s %v", cluster.Name, err)
	}
}

func getKubeConfigPath(cluster *types.Cluster) string {
	clusterConfigPath := cluster.Spec.ConfigPath
	splitted := strings.Split(clusterConfigPath, "/")
	fileName := splitted[len(splitted)-1]
	path := strings.TrimSuffix(clusterConfigPath, fileName)
	kubeConfigFileName := fmt.Sprintf("%s/kube_config_%s", path, fileName)
	return kubeConfigFileName
}

func (c *Controller) addConfig(obj interface{}) {
	cluster := obj.(*types.Cluster)
	c.sync(cluster)
}

func (c *Controller) updateConfig(obj interface{}, updated interface{}) {
	cluster := obj.(*types.Cluster)
	c.sync(cluster)
}
