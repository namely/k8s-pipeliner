package builder

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/namely/k8s-pipeliner/pipeline/builder/types"
	"github.com/namely/k8s-pipeliner/pipeline/config"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

var (
	// ErrNoContainers is returned when a manifest has defined containers in it
	ErrNoContainers = errors.New("builder: no containers were found in given manifest file")
	// ErrNoDeployGroups is returned when a stage in the pipeline.yml does not have any deploy groups on it.
	ErrNoDeployGroups = errors.New("builder: no deploy groups were defined in given pipeline.yml")
	// ErrOverrideContention is returned when a manifest defines multiple containers and overrides were provided
	ErrOverrideContention = errors.New("builder: overrides were provided to a group that has multiple containers defined")
	// ErrDeploymentJob is returned when a manifest uses a deployment for a one shot job
	ErrDeploymentJob = errors.New("builder: a deployment manifest was provided for a run job pod")
	// ErrKubernetesAPI defines whether the manifest we've provided falls within the scope
	ErrKubernetesAPI = errors.New("builder: could not marshal this type of kubernetes manifest")
	// ErrNoManifestFiles is returned when a manifest stage does not
	ErrNoManifestFiles = errors.New("builder: no manifest files defined")
	// ErrNoNamespace is returned when a manifest does not have a namespace
	ErrNoNamespace = errors.New("builder: manifest does not have a namespace defined")
	// ErrNoKubernetesMetadata is returned when a manifest does not have kubernetes metadata
	ErrNoKubernetesMetadata = errors.New("builder: manifest does not have kubernetes metadata attached")
)

const (
	// JenkinsTrigger is the name of the type in the spinnaker json for pipeline config for jenkins job triggers
	JenkinsTrigger = "jenkins"
	// WebhookTrigger is the name of the type in the spinnaker json for pipeline config for webhooks
	WebhookTrigger = "webhook"
	// LoadBalancerFormat creates the label selectors to attach pipeline.yml labels to deployment selectors
	LoadBalancerFormat = "load-balancer-%s"
)

// Builder constructs a spinnaker pipeline JSON from a pipeliner config
type Builder struct {
	pipeline *config.Pipeline

	isLinear   bool
	basePath   string
	v2Provider bool
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
		AppConfig:            map[string]interface{}{},
	}

	sp.Notifications = buildNotifications(b.pipeline.Notifications)
	sp.Triggers = make([]types.Trigger, 0)

	for _, trigger := range b.pipeline.Triggers {
		if jt := trigger.Jenkins; jt != nil {
			sp.Triggers = append(sp.Triggers, &types.JenkinsTrigger{
				TriggerObject: types.TriggerObject{
					Enabled: newDefaultTrue(jt.Enabled),
					Type:    JenkinsTrigger,
				},

				Job:          jt.Job,
				Master:       jt.Master,
				PropertyFile: jt.PropertyFile,
			})
		}

		if wh := trigger.Webhook; wh != nil {
			sp.Triggers = append(sp.Triggers, &types.WebhookTrigger{
				TriggerObject: types.TriggerObject{
					Enabled: wh.Enabled,
					Type:    WebhookTrigger,
				},
				Source: wh.Source,
			})
		}
	}

	sp.Parameters = make([]types.Parameter, len(b.pipeline.Paramters))
	for i, param := range b.pipeline.Paramters {
		sp.Parameters[i] = types.Parameter{
			Name:        param.Name,
			Description: param.Description,
			Default:     param.Default,
			Required:    param.Required,
		}
	}
	var stageIndex = 0
	for _, stage := range b.pipeline.Stages {
		var s types.Stage
		var err error

		if b.v2Provider {
			if stage.RunJob != nil {
				s, err = b.buildV2RunJobStage(stageIndex, stage)
				stageIndex = stageIndex + 1
				if stage.RunJob.DeleteJob == true {
					sp.Stages = append(sp.Stages, s)
					s, err = b.buildV2DeleteManifestStage(stageIndex, stage)
					stageIndex = stageIndex + 1
				}
			}

			if stage.Deploy != nil {
				s, err = b.buildV2ManifestStageFromDeploy(stageIndex, stage)
				stageIndex = stageIndex + 1
			}
		} else {
			if stage.RunJob != nil {
				s, err = b.buildRunJobStage(stageIndex, stage)
				stageIndex = stageIndex + 1
			}
			if stage.Deploy != nil {
				s, err = b.buildDeployStage(stageIndex, stage)
				stageIndex = stageIndex + 1
			}
		}

		if stage.ManualJudgement != nil {
			s, err = b.buildManualJudgementStage(stageIndex, stage)
			stageIndex = stageIndex + 1
		}

		if stage.DeployEmbeddedManifests != nil {
			s, err = b.buildDeployEmbeddedManifestStage(stageIndex, stage)
			stageIndex = stageIndex + 1
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

		Account:            s.Account,
		Application:        b.pipeline.Application,
		Annotations:        make(map[string]string),
		CloudProvider:      "kubernetes",
		CloudProviderType:  "kubernetes",
		DNSPolicy:          "ClusterFirst", // hack for now
		ServiceAccountName: s.RunJob.ServiceAccountName,
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
	rjs.VolumeSources = mg.VolumeSources
	rjs.Annotations = mg.PodAnnotations

	if po := s.RunJob.PodOverrides; po != nil {
		for k, v := range po.Annotations {
			rjs.Annotations[k] = v
		}
	}

	// overrides can be provided for jobs since things like
	// migrations typically need all of the same environment variables
	// and such from a deployment manifest
	if s.RunJob.Container != nil {
		rjs.Container.Args = s.RunJob.Container.Args
		rjs.Container.Command = s.RunJob.Container.Command
	}

	return rjs, nil
}

func (b *Builder) buildDeployEmbeddedManifestStage(index int, s config.Stage) (*types.ManifestStage, error) {
	ds := b.defaultManifestStage(index, s)
	maniStage := s.DeployEmbeddedManifests

	if len(maniStage.Files) < 1 {
		return nil, ErrNoManifestFiles
	}

	// update the moniker
	ds.Moniker = types.Moniker{
		App:     maniStage.Moniker.App,
		Detail:  maniStage.Moniker.Detail,
		Stack:   maniStage.Moniker.Stack,
		Cluster: maniStage.Moniker.Cluster,
	}

	parser := NewManfifestParser(b.pipeline, b.basePath)
	for _, path := range maniStage.Files {
		obj, err := parser.ManifestFromFile(path)
		if err != nil {
			return nil, errors.Wrapf(err, "could not parse manifest file: %s", path)
		}

		ds.Manifests = append(ds.Manifests, obj)
	}

	return ds, nil
}

func (b *Builder) buildV2ManifestStageFromDeploy(index int, s config.Stage) (*types.ManifestStage, error) {
	ds := b.defaultManifestStage(index, s)

	if len(s.Deploy.Groups) == 0 {
		return nil, ErrNoDeployGroups
	}

	cluster := []string{s.Account, b.pipeline.Application, s.Deploy.Groups[0].Details, s.Deploy.Groups[0].Stack}

	ds.Moniker.Cluster = strings.Join(cluster, "-")
	ds.Moniker.Detail = s.Deploy.Groups[0].Details
	ds.Moniker.Stack = s.Deploy.Groups[0].Stack

	parser := NewManfifestParser(b.pipeline, b.basePath)

	for _, group := range s.Deploy.Groups {
		manifest, err := parser.ManifestFromScaffold(group)

		if err != nil {
			return nil, err
		}

		resource := &appsv1.Deployment{}
		switch manifest.GetObjectKind().GroupVersionKind().Kind {
		case "Deployment":
			if err := scheme.Scheme.Convert(manifest, resource, nil); err != nil {
				return nil, err
			}
		default:
			return nil, ErrKubernetesAPI
		}

		_, err = buildDeployment(resource, group)
		if err != nil {
			return nil, err
		}

		ds.Manifests = append(ds.Manifests, manifest)
	}

	return ds, nil
}

func (b *Builder) defaultManifestStage(index int, s config.Stage) *types.ManifestStage {
	return &types.ManifestStage{
		StageMetadata:           buildStageMetadata(s, "deployManifest", index, b.isLinear),
		Account:                 s.Account,
		CloudProvider:           "kubernetes",
		Location:                "",
		ManifestArtifactAccount: "embedded-artifact",
		ManifestName:            "",
		Moniker: types.Moniker{
			App: b.pipeline.Application,
		},
		Relationships: types.Relationships{LoadBalancers: []interface{}{}, SecurityGroups: []interface{}{}},
		Source:        "text",
	}
}

func (b *Builder) buildV2RunJobStage(index int, s config.Stage) (*types.ManifestStage, error) {
	ds := &types.ManifestStage{
		StageMetadata:           buildStageMetadata(s, "deployManifest", index, b.isLinear),
		Account:                 s.Account,
		CloudProvider:           "kubernetes",
		Location:                "",
		ManifestArtifactAccount: "embedded-artifact",
		ManifestName:            "",
		Moniker: types.Moniker{
			App:     b.pipeline.Application,
			Cluster: fmt.Sprintf("%s-%s", b.pipeline.Application, s.Account),
			Detail:  "",
			Stack:   "",
		},
		Relationships: types.Relationships{LoadBalancers: []interface{}{}, SecurityGroups: []interface{}{}},
		Source:        "text",
	}

	parser := NewManfifestParser(b.pipeline, b.basePath)
	obj, err := parser.ManifestFromScaffold(s.RunJob)

	if err != nil {
		return nil, err
	}

	switch t := obj.(type) {
	case *corev1.Pod:
		if s.RunJob.Container != nil {
			t.Spec.Containers[0].Command = s.RunJob.Container.Command
			t.Spec.Containers[0].Args = s.RunJob.Container.Args
		}
	default:
		return nil, ErrDeploymentJob
	}

	ds.Manifests = append(ds.Manifests, obj)

	return ds, nil

}

func (b *Builder) buildV2DeleteManifestStage(index int, s config.Stage) (*types.DeleteManifestStage, error) {
	s.Name = "Delete " + s.Name
	dms := &types.DeleteManifestStage{
		StageMetadata: buildStageMetadata(s, "deleteManifest", index, b.isLinear),
		Account:       s.Account,
		CloudProvider: "kubernetes",
		Kinds:         []string{"Job"},
		LabelSelectors: types.LabelSelectors{
			Selectors: []types.Selector{},
		},
		Location: "",
		Options: types.Options{
			Cascading: true,
		},
	}

	parser := NewManfifestParser(b.pipeline, b.basePath)
	obj, err := parser.ManifestFromScaffold(s.RunJob)

	if err != nil {
		return nil, err
	}

	var labels map[string]string

	switch t := obj.(type) {
	case metav1.Object:
		labels = t.GetLabels()
		namespace := t.GetNamespace()
		if namespace == "" {
			return nil, ErrNoNamespace
		}
		dms.Location = namespace
	default:
		return nil, ErrNoKubernetesMetadata
	}

	for key, value := range labels {
		dms.LabelSelectors.Selectors = append(dms.LabelSelectors.Selectors, types.Selector{Key: key, Values: []string{value}, Kind: "EQUALS"})
	}

	return dms, nil

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

		if po := group.PodOverrides; po != nil {
			for k, v := range po.Annotations {
				mg.PodAnnotations[k] = v
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

func buildDeployment(deploy *appsv1.Deployment, group config.Group) (*appsv1.Deployment, error) {

	if len(deploy.Spec.Template.Spec.Containers) == 0 {
		return nil, ErrNoContainers
	}

	if overrides := group.ContainerOverrides; overrides != nil {
		if len(deploy.Spec.Template.Spec.Containers) > 1 {
			return nil, ErrOverrideContention
		}
		if overrides.Args != nil {
			deploy.Spec.Template.Spec.Containers[0].Args = overrides.Args
		}
		if overrides.Command != nil {
			deploy.Spec.Template.Spec.Containers[0].Command = overrides.Command
		}
	}

	if lb := group.LoadBalancers; lb != nil {
		labels := make(map[string]string)

		for _, l := range lb {
			labels[fmt.Sprintf(LoadBalancerFormat, l)] = "true"
		}

		if deploy.Spec.Selector != nil {
			for key, val := range labels {
				deploy.Spec.Selector.MatchLabels[key] = val
			}
		} else {
			deploy.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: labels,
			}
		}

		l := deploy.ObjectMeta.GetLabels()

		if l == nil {
			l = make(map[string]string)
		}

		for key, val := range labels {
			l[key] = val
		}

		deploy.ObjectMeta.SetLabels(l)

		l = deploy.Spec.Template.GetLabels()

		for key, val := range labels {
			l[key] = val
		}

		deploy.Spec.Template.SetLabels(l)
	}
	return deploy, nil
}

func newDefaultTrue(original *bool) bool {
	if original == nil {
		return true
	}

	return *original
}
