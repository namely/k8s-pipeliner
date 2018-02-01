package builder

import (
	"encoding/json"
	"errors"
	"fmt"
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

	isLinear bool
}

// New initializes a new builder for a pipeline config
func New(p *config.Pipeline, opts ...OptFunc) *Builder {
	b := &Builder{pipeline: p}
	for _, opt := range opts {
		opt(b)
	}

	return b
}

// Pipeline returns a filled out spinnaker pipeline from the given
// config
func (b *Builder) Pipeline() (*types.SpinnakerPipeline, error) {
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
		var s types.Stage
		var err error

		if stage.RunJob != nil {
			s, err = b.buildRunJobStage(stageIndex, stage)
		}
		if stage.Deploy != nil {
			s, err = b.buildDeployStage(stageIndex, stage)
		}
		if stage.ManualJudgement != nil {
			s, err = b.buildManualJudgementStage(stageIndex, stage)
		}

		if err != nil {
			return nil, err
		}
		sp.Stages = append(sp.Stages, s)
	}

	return sp, nil
}

// MarshalJSON implements json.Marshaller
func (b *Builder) MarshalJSON() ([]byte, error) {
	sp, err := b.Pipeline()
	if err != nil {
		return nil, err
	}

	return json.Marshal(sp)
}

func (b *Builder) buildRunJobStage(index int, s config.Stage) (*types.RunJobStage, error) {
	rjs := &types.RunJobStage{
		StageMetadata: buildStageMetadata(s, "runJob", index, b.isLinear),

		Account:           s.Account,
		Application:       b.pipeline.Application,
		Annotations:       make(map[string]string),
		CloudProvider:     "kubernetes",
		CloudProviderType: "kubernetes",
		VolumeSources:     []interface{}{},
		DNSPolicy:         "ClusterFirst", // hack for now
	}

	mg, err := ContainersFromManifest(s.RunJob.ManifestFile)
	if err != nil {
		return nil, err
	}

	if len(mg.Containers) == 0 {
		return nil, ErrNoContainers
	}
	rjs.Container = mg.Containers[0]
	rjs.Namespace = mg.Namespace

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
		StageMetadata: buildStageMetadata(s, "deploy", index, b.isLinear),
	}

	mg, err := ContainersFromManifest(s.Deploy.ManifestFile)
	if err != nil {
		return nil, err
	}
	if len(mg.Containers) == 0 {
		return nil, ErrNoContainers
	}

	// grab the load balancers for the deployment
	var lbs []string
	if l, ok := mg.Annotations[SpinnakerLoadBalancersAnnotations]; ok {
		lbs = strings.Split(l, ",")
	}

	cluster := types.Cluster{
		Account:               s.Account,
		Application:           b.pipeline.Application,
		Containers:            mg.Containers,
		LoadBalancers:         lbs,
		Region:                mg.Namespace,
		Namespace:             mg.Namespace,
		MaxRemainingAsgs:      s.Deploy.MaxRemainingASGS,
		ReplicaSetAnnotations: mg.Annotations,
		ScaleDown:             s.Deploy.ScaleDown,
		Stack:                 s.Deploy.Stack,
		Strategy:              s.Deploy.Strategy,
		TargetSize:            s.Deploy.TargetSize,
		VolumeSources:         mg.VolumeSources,

		// TODO(bobbytables): allow these to be configurable
		Events: []interface{}{},
		InterestingHealthProviderNames: []string{"KubernetesContainer", "KubernetesPod"},
		Provider:                       "kubernetes",
		CloudProvider:                  "kubernetes",
		DNSPolicy:                      "ClusterFirst",
		TerminationGracePeriodSeconds:  30,
	}

	ds.Clusters = []types.Cluster{cluster}

	return ds, nil
}

func (b *Builder) buildManualJudgementStage(index int, s config.Stage) (*types.ManualJudgementStage, error) {
	mjs := &types.ManualJudgementStage{
		StageMetadata: buildStageMetadata(s, "manualJudgment", index, b.isLinear),

		FailPipeline: s.ManualJudgement.FailPipeline,
		Instructions: s.ManualJudgement.Instructions,
		Inputs:       s.ManualJudgement.Inputs,
	}

	return mjs, nil
}

func buildStageMetadata(s config.Stage, t string, index int, linear bool) types.StageMetadata {
	var nots []types.Notification
	for _, n := range s.Notifications {
		message := make(map[string]types.NotificationMessage)
		for messageOn, text := range n.Message {
			message[messageOn] = types.NotificationMessage{Text: text}
		}

		nots = append(nots, types.Notification{
			Address: n.Address,
			Level:   n.Level,
			Type:    n.Type,
			When:    n.When,
			Message: message,
		})
	}

	refID := s.RefID
	reliesOn := s.ReliesOn
	if linear {
		refID = fmt.Sprintf("%d", index)
		if index > 0 {
			reliesOn = []string{fmt.Sprintf("%d", index-1)}
		}
	}

	return types.StageMetadata{
		Name:                 s.Name,
		RefID:                refID,
		RequisiteStageRefIds: reliesOn,
		Type:                 t,
		Notifications:        nots,
		SendNotifications:    (len(nots) > 0),
	}
}
