package main

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/namely/k8s-pipeliner/pipeline/builder"
	"github.com/namely/k8s-pipeliner/pipeline/config"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

const (
	// Version defines the current version of k8s-pipeliner
	Version = "0.0.1"
)

func main() {
	app := cli.NewApp()
	app.Name = "k8s-pipeliner"
	app.Description = "create spinnaker pipelines from kubernetes clusters"
	app.Flags = []cli.Flag{}
	app.Version = Version

	app.Commands = []cli.Command{
		{
			Name:   "create",
			Usage:  "creates a spinnaker pipeline for a given application on multiple k8s clusters",
			Action: createAction,
		},
	}

	if err := app.Run(os.Args); err != nil {
		logrus.WithError(err).Error()
		os.Exit(1)
	}
}

func createAction(ctx *cli.Context) error {
	pipelineFile := ctx.Args().First()
	if pipelineFile == "" {
		return errors.New("missing parameter: file")
	}

	f, err := os.Open(pipelineFile)
	if err != nil {
		return err
	}

	pipeline, err := config.NewPipeline(f)
	if err != nil {
		return err
	}

	return json.NewEncoder(os.Stdout).Encode(builder.New(pipeline))
}
