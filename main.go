package main

import (
	"io"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"time"

	"fmt"

	"github.com/pkg/errors"
	"github.com/rancher/kubecon2018/.trash-cache/src/github.com/Sirupsen/logrus"
	"github.com/rancher/kubecon2018/controllers"
	"github.com/urfave/cli"
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
		},
	}

	app.Action = func(c *cli.Context) error {
		return run(c.String("kubeconfig"))
	}

	app.Run(os.Args)
}

func run(kubeConfig string) error {
	//restConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	//if err != nil {
	//	return err
	//}

	logrus.Info("Running command")
	cmdName := "/Users/alena/go/src/github.com/rancher/rke/rke"
	cmdArgs := []string{"up", "--config", "/Users/alena/Desktop/conferences/KubeconEU2018/cluster_aws.yml"}

	cmd := exec.Command(cmdName, cmdArgs...)
	stdout, err := cmd.StderrPipe()
	if err != nil {
		return errors.Wrapf(err, "error getting stdout from cmd '%v'", cmd)
	}

	if err := cmd.Start(); err != nil {
		return errors.Wrapf(err, "error starting cmd '%v'", cmd)
	}
	defer func() {
		if err := cmd.Wait(); err != nil {
			logrus.Debugf("error waiting for cmd '%v' %v", cmd, err)
		}
	}()

	printLogs(stdout)
	controllers.Register()

	for {
		time.Sleep(5 * time.Second)
	}
}

func printLogs(r io.Reader) {
	buf := make([]byte, 80)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			fmt.Print(string(buf[0:n]))
		}
		if err != nil {
			os.Exit(0)
			break
		}
	}
}
