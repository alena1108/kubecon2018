package controllers

import (
	"time"

	"github.com/rancher/kubecon2018/controllers/provisioner"
	clusterclient "github.com/rancher/kubecon2018/pkg/client/clientset/versioned"
	informers "github.com/rancher/kubecon2018/pkg/client/informers/externalversions"
	rest "k8s.io/client-go/rest"
)

func Register(config *rest.Config) error {
	clusterClient, err := clusterclient.NewForConfig(config)
	if err != nil {
		return err
	}
	clusterInformerFactory := informers.NewSharedInformerFactory(clusterClient, time.Second*30)

	provisioner.Register(clusterClient, clusterInformerFactory)
	return nil
}
