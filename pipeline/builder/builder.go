package builder

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/namely/k8s-pipeliner/pipeline/builder/types"
	"github.com/namely/k8s-pipeliner/pipeline/config"
)

var (
	// ErrNoContainers is returned when a manifest has defined containers in it
	ErrNoContainers = errors.New("builder: no containers were found in given manifest file")

	// ErrOverrideContention is returned when a manifest defines multiple containers and overrides were provided
	ErrOverrideContention = errors.New("builder: overrides were provided to a group that has multiple containers defined")
)

// Builder constructs a spinnaker pipeline JSON from a pipeliner config
type Builder struct {
	pipeline *config.Pipeline

	isLinear bool
	basePath string
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
	sp := &types.SpinnakerPipeline{
		LimitConcurrent:      b.pipeline.DisableConcurrentExecutions,
		KeepWaitingPipelines: b.pipeline.KeepQueuedPipelines,
		Description:          b.pipeline.Description,
	}

	sp.Notifications = buildNotifications(b.pipeline.Notifications)

	sp.Triggers = []types.Trigger{}

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

	parser := NewManfifestParser(b.pipeline, b.basePath)

	mg, err := parser.ContainersFromScaffold(s.RunJob)
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

	parser := NewManfifestParser(b.pipeline, b.basePath)

	for _, group := range s.Deploy.Groups {
		mg, err := parser.ContainersFromScaffold(group)
		if err != nil {
			return nil, err
		}
		if len(mg.Containers) == 0 {
			return nil, ErrNoContainers
		}

		// check for overrides defined on the group so we can replace the containers
		// values before rendering our spinnaker json.
		if overrides := group.ContainerOverrides; overrides != nil {
			if len(mg.Containers) > 1 {
				return nil, ErrOverrideContention
			}

			container := mg.Containers[0]
			if overrides.Args != nil {
				container.Args = overrides.Args
			}

			if overrides.Command != nil {
				container.Command = overrides.Command
			}
		}

		cluster := types.Cluster{
			Account:               s.Account,
			Application:           b.pipeline.Application,
			Containers:            mg.Containers,
			InitContainers:        mg.InitContainers,
			LoadBalancers:         group.LoadBalancers,
			Region:                mg.Namespace,
			Namespace:             mg.Namespace,
			MaxRemainingAsgs:      group.MaxRemainingASGS,
			ReplicaSetAnnotations: mg.Annotations,
			PodAnnotations:        mg.PodAnnotations,
			ScaleDown:             group.ScaleDown,
			Stack:                 group.Stack,
			Details:               group.Details,
			Strategy:              group.Strategy,
			TargetSize:            group.TargetSize,
			VolumeSources:         mg.VolumeSources,

			// TODO(bobbytables): allow these to be configurable
			Events: []interface{}{},
			InterestingHealthProviderNames: []string{"KubernetesContainer", "KubernetesPod"},
			Provider:                       "kubernetes",
			CloudProvider:                  "kubernetes",
			DNSPolicy:                      "ClusterFirst",
			TerminationGracePeriodSeconds:  30,
		}

		ds.Clusters = append(ds.Clusters, cluster)
	}

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
	refID := s.RefID
	if s.ReliesOn == nil {
		s.ReliesOn = []string{}
	}
	reliesOn := s.ReliesOn

	if linear {
		refID = fmt.Sprintf("%d", index)
		if index > 0 {
			reliesOn = []string{fmt.Sprintf("%d", index-1)}
		}
	}

	notifications := buildNotifications(s.Notifications)
	return types.StageMetadata{
		Name:                 s.Name,
		RefID:                refID,
		RequisiteStageRefIds: reliesOn,
		Type:                 t,
		Notifications:        notifications,
		SendNotifications:    (len(notifications) > 0),
	}
}

func buildNotifications(notifications []config.Notification) []types.Notification {
	var nots []types.Notification
	for _, n := range notifications {
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

	return nots
}
