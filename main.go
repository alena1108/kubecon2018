package main

import (
	"bytes"
	"io/ioutil"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"time"

	"fmt"

	"github.com/rancher/kubecon2018/.trash-cache/src/github.com/Sirupsen/logrus"
	"github.com/rancher/kubecon2018/controllers"
	"github.com/urfave/cli"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	VERSION = "dev"
)

func main() {
	app := cli.NewApp()
	app.Version = VERSION
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "kubeconfig",
			Usage:  "Kube config for accessing k8s cluster",
			EnvVar: "KUBECONFIG",
			Value:  "/Users/alena/.kube/config",
		},
	}

	app.Action = func(c *cli.Context) error {
		return run(c.String("kubeconfig"))
	}

	app.Run(os.Args)
}

func run(kubeConfig string) error {
	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		return err
	}

	// Create custom resource definitions
	if err := createCRDS(); err != nil {
		return err
	}

	// Register controllers
	if err := controllers.Register(restConfig); err != nil {
		return err
	}

	// Run controllers
	logrus.Info("Running controllers")

	for {
		time.Sleep(5 * time.Second)
	}
}

func createCRDS() error {
	logrus.Info("Creating CRDs...")
	cmdName := "kubectl"
	files, err := ioutil.ReadDir("./config/crd")
	if err != nil {
		return err
	}
	for _, file := range files {
		filePath := fmt.Sprintf("./config/crd/%s", file.Name())
		logrus.Infof("Creating crd for file %s", filePath)
		cmdArgs := []string{"apply", "-f", filePath}
		cmd := exec.Command(cmdName, cmdArgs...)
		var out bytes.Buffer
		cmd.Stderr = &out
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create CRD [%s] %v %v", file.Name(), err, out.String())
		}
	}

	logrus.Info("Created CRDs")
	return nil
}
