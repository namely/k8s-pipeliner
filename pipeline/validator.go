package pipeline

import (
	"fmt"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/namely/k8s-pipeliner/pipeline/builder"
	"github.com/namely/k8s-pipeliner/pipeline/builder/types"
	"github.com/namely/k8s-pipeliner/pipeline/config"
)

// Validator validates that a pipeline is valid
type Validator struct {
	pipeline *config.Pipeline
}

// NewValidator initializes a validator object
func NewValidator(p *config.Pipeline) *Validator {
	return &Validator{pipeline: p}
}

// Validate performs some validations on the pipeline configuration
// to see if it passes some simple standards such as "do deploys have resources allocated"
func (v *Validator) Validate() error {
	b := builder.New(v.pipeline)
	sp, err := b.Pipeline()
	if err != nil {
		return err
	}

	var errs *multierror.Error
	for _, stage := range sp.Stages {
		switch s := stage.(type) {
		case *types.DeployStage:
			errs = multierror.Append(validateDeployStage(s))
		}
	}

	return errs
}

func validateDeployStage(s *types.DeployStage) error {
	var errs *multierror.Error

	for _, cluster := range s.Clusters {
		for _, container := range cluster.Containers {
			if container.Requests.CPU == "0" {
				err := fmt.Errorf("Stage: %s, Container: %s - Missing CPU on Resource Requests", s.Name, container.Name)
				errs = multierror.Append(err)
			}

			if container.Requests.Memory == "0" {
				err := fmt.Errorf("Stage: %s, Container: %s - Missing Memory on Resource Requests", s.Name, container.Name)
				errs = multierror.Append(err)
			}

			if container.Limits.CPU == "0" {
				err := fmt.Errorf("Stage: %s, Container: %s - Missing CPU on Resource Limits", s.Name, container.Name)
				errs = multierror.Append(err)
			}

			if container.Limits.Memory == "0" {
				err := fmt.Errorf("Stage: %s, Container: %s - Missing Memory on Resource Limits", s.Name, container.Name)
				errs = multierror.Append(err)
			}
		}
	}

	return errs
}
