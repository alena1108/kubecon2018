package controllers

import (
	"time"

	"github.com/rancher/kubecon2018/controllers/annotator"
	"github.com/rancher/kubecon2018/controllers/configgenerator"
	"github.com/rancher/kubecon2018/controllers/healthchecker"
	"github.com/rancher/kubecon2018/controllers/provisioner"
	client "github.com/rancher/kubecon2018/pkg/client/clientset/versioned"
	informers "github.com/rancher/kubecon2018/pkg/client/informers/externalversions"
	rest "k8s.io/client-go/rest"
)

func Register(config *rest.Config) error {
	client, err := client.NewForConfig(config)
	if err != nil {
		return err
	}
	clusterInformerFactory := informers.NewSharedInformerFactory(client, time.Second*30)

	provisioner.Register(client, clusterInformerFactory)
	configgenerator.Register(client, clusterInformerFactory)
	healthchecker.Register(client, clusterInformerFactory)
	annotator.Register(client, clusterInformerFactory)

	return nil
}
