package builder

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/namely/k8s-pipeliner/pipeline/builder/types"
	"github.com/namely/k8s-pipeliner/pipeline/config"
)

var (
	// ErrNoContainers is returned when a manifest has defined containers in it
	ErrNoContainers = errors.New("builder: no containers were found in given manifest file")
)

// Builder constructs a spinnaker pipeline JSON from a pipeliner config
type Builder struct {
	pipeline *config.Pipeline
}

// New initializes a new builder for a pipeline config
func New(p *config.Pipeline) *Builder {
	return &Builder{p}
}

// MarshalJSON implements json.Marshaller
func (b *Builder) MarshalJSON() ([]byte, error) {
	sp := &types.SpinnakerPipeline{}

	for _, trigger := range b.pipeline.Triggers {
		if trigger.Jenkins != nil {
			jt := trigger.Jenkins

			sp.Triggers = append(sp.Triggers, &types.JenkinsTrigger{
				Enabled:      true,
				Job:          jt.Job,
				Master:       jt.Master,
				PropertyFile: jt.PropertyFile,
				Type:         "jenkins",
			})

			continue
		}
	}

	for stageIndex, stage := range b.pipeline.Stages {
		if stage.RunJob != nil {
			stage, err := b.buildRunJobStage(stageIndex, stage)
			if err != nil {
				return nil, err
			}

			sp.Stages = append(sp.Stages, stage)
			continue
		}

		if stage.Deploy != nil {
			stage, err := b.buildDeployStage(stageIndex, stage)
			if err != nil {
				return nil, err
			}

			sp.Stages = append(sp.Stages, stage)
		}
	}

	return json.Marshal(sp)
}

func (b *Builder) buildRunJobStage(index int, s config.Stage) (*types.RunJobStage, error) {
	reliesOn := make([]string, 0)
	if s.ReliesOn != nil {
		reliesOn = s.ReliesOn
	}

	rjs := &types.RunJobStage{
		Account:              s.Account,
		Application:          b.pipeline.Application,
		Annotations:          make(map[string]string),
		CloudProvider:        "kubernetes",
		CloudProviderType:    "kubernetes",
		Name:                 s.Name,
		VolumeSources:        []interface{}{},
		RefID:                s.RefID,
		RequisiteStageRefIds: reliesOn,
		DNSPolicy:            "ClusterFirst", // hack for now
		Type:                 "runJob",
	}

	cs, _, err := ContainersFromManifest(s.RunJob.ManifestFile)
	if err != nil {
		return nil, err
	}

	if len(cs) == 0 {
		return nil, ErrNoContainers
	}
	rjs.Container = cs[0]

	// overrides can be provided for jobs since things like
	// migrations typically need all of the same environment variables
	// and such from a deployment manifest
	if s.RunJob.Container != nil {
		rjs.Container.Args = s.RunJob.Container.Args
		rjs.Container.Command = s.RunJob.Container.Command
	}

	return rjs, nil
}

// As of right now, this tool only supports deploying one server group at a time from a
// manifest file. So the clusters array will ALWAYS be 1 in length.
func (b *Builder) buildDeployStage(index int, s config.Stage) (*types.DeployStage, error) {
	ds := &types.DeployStage{
		Name:                 s.Name,
		RefID:                s.RefID,
		RequisiteStageRefIds: s.ReliesOn,
		Type:                 "deploy",
	}

	cs, annotations, err := ContainersFromManifest(s.Deploy.ManifestFile)
	if err != nil {
		return nil, err
	}
	if len(cs) == 0 {
		return nil, ErrNoContainers
	}

	// grab the load balancers for the deployment
	var lbs []string
	if l, ok := annotations[SpinnakerLoadBalancersAnnotations]; ok {
		lbs = strings.Split(l, ",")
	}

	cluster := types.Cluster{
		Account:                        s.Account,
		Application:                    b.pipeline.Application,
		VolumeSources:                  []interface{}{}, // TODO(bobbytables): allow this to be configurable
		Events:                         []interface{}{},
		Containers:                     cs,
		LoadBalancers:                  lbs,
		InterestingHealthProviderNames: []string{"KubernetesContainer", "KubernetesPod"}, // TODO(bobbytables): allow this to be configurable
		Region:                        s.Region,
		Namespace:                     s.Region,
		DNSPolicy:                     "ClusterFirst", // TODO(bobbytables): allow this to be configurable
		MaxRemainingAsgs:              s.Deploy.MaxRemainingASGS,
		ReplicaSetAnnotations:         annotations,
		ScaleDown:                     s.Deploy.ScaleDown,
		Stack:                         s.Deploy.Stack,
		Strategy:                      s.Deploy.Strategy,
		TargetSize:                    s.Deploy.TargetSize,
		TerminationGracePeriodSeconds: 30, // TODO(bobbytables): allow this to be configurable
	}

	ds.Clusters = []types.Cluster{cluster}

	return ds, nil
}
