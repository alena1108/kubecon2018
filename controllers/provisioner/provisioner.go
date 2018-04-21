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
		AddFunc: handleClusterAdd,
	})
	stop := make(chan struct{})
	go controller.clusterInformer.Run(stop)
	logrus.Info("Registered provisioner controller")
}

func handleClusterAdd(obj interface{}) {
	cluster := obj.(*types.Cluster)
	logrus.Infof("Cluster [%s] is added; provisioning...", cluster.Name)
	//TODO provision the cluster

	logrus.Info("Running command")
	cmdName := "/Users/alena/go/src/github.com/rancher/rke/rke"
	cmdArgs := []string{"up", "--config", cluster.Spec.ConfigPath}

	cmd := exec.Command(cmdName, cmdArgs...)
	stdout, err := cmd.StderrPipe()
	if err != nil {
		logrus.Errorf("error getting stdout from cmd '%v' %v", cmd, err)
	}

	if err := cmd.Start(); err != nil {
		logrus.Errorf("error starting cmd '%v' %v", cmd, err)
	}
	defer func() {
		if err := cmd.Wait(); err != nil {
			logrus.Debugf("error waiting for cmd '%v' %v", cmd, err)
		}
	}()

	printLogs(stdout)

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
