package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/namely/kubernetes-pipeliner/spinnaker"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	app := cli.NewApp()
	app.Name = "k8s-pipeliner"
	app.Description = "create spinnaker pipelines from kubernetes clusters"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "spinnaker-api",
			Usage: "The URL for the spinnaker API (http://spinnaker-gate.example.com)",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:  "create",
			Usage: "creates a spinnaker pipeline for a given application on multiple k8s clusters",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "deployment",
					Usage: "name of the deployment in the kubernetes cluster",
				},
				cli.StringFlag{
					Name:  "namespace",
					Usage: "namespace in the k8s cluster that the deployment exists in",
				},
				cli.StringFlag{
					Name:  "config",
					Usage: "a kubeconfig file location",
				},
				cli.StringFlag{
					Name:  "account",
					Usage: "the name of the account in spinnaker",
				},
			},
			Action: createAction,
		},
	}

	if err := app.Run(os.Args); err != nil {
		logrus.WithError(err).Error()
		os.Exit(1)
	}
}

func createAction(ctx *cli.Context) error {
	deployment := ctx.String("deployment")
	namespace := ctx.String("namespace")

	// the name of the app that will exist inside of spinnaker
	applicationName := strings.Split(deployment, "-")[0]

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", ctx.String("config"))
	if err != nil {
		logrus.Fatal(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logrus.Fatal(err.Error())
	}

	dep, err := clientset.AppsV1beta1().Deployments(namespace).Get(deployment, metav1.GetOptions{})
	if err != nil {
		logrus.Fatal(err)
	}

	cfg := spinnaker.DeployStageConfig{
		Account:       ctx.String("account"),
		Application:   applicationName,
		DockerAccount: "namely-registry",
	}

	b, _ := json.MarshalIndent(spinnaker.DeployStageFromK8sDep(cfg, dep), "", "\t")
	fmt.Println(string(b))

	return nil
}
