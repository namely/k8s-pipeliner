package builder

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	cnfgrtr "github.com/namely/k8s-configurator"
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

	// Stages helps to translate from spinnaker account to configurator stages
	Stages = map[string]string{
		"int":            "int",
		"int-k8s":        "int",
		"staging":        "stage",
		"staging-k8s":    "stage",
		"production":     "production",
		"production-k8s": "production",
		"ops":            "ops",
		"ops-k8s":        "ops",
	}
)

const (
	// JenkinsTrigger is the name of the type in the spinnaker json for pipeline config for jenkins job triggers
	JenkinsTrigger = "jenkins"
	// WebhookTrigger is the name of the type in the spinnaker json for pipeline config for webhooks
	WebhookTrigger = "webhook"
	// LoadBalancerFormat creates the label selectors to attach pipeline.yml labels to deployment selectors
	LoadBalancerFormat = "load-balancer-%s"
	// HourInMS provides 1 hour in milliseconds
	HourInMS int64 = 3600000
)

// Builder constructs a spinnaker pipeline JSON from a pipeliner config
type Builder struct {
	pipeline *config.Pipeline

	isLinear         bool
	basePath         string
	timeoutHours     int
	overrideAccounts map[string]string
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

	sp.Parameters = make([]types.Parameter, len(b.pipeline.Parameters))
	for i, param := range b.pipeline.Parameters {
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

		// if the account has an override, switch the account name
		if account, ok := b.overrideAccounts[stage.Account]; ok {
			stage.Account = account
		}

		if stage.RunJob != nil {
			s, err = b.buildRunJobStage(stageIndex, stage)
			if err != nil {
				return sp, fmt.Errorf("Failed to b.buildRunJobStage with error: %v", err)
			}
			stageIndex = stageIndex + 1
		}
		if stage.Deploy != nil {
			s, err = b.buildDeployStage(stageIndex, stage)
			if err != nil {
				return sp, fmt.Errorf("Failed to b.buildDeployStage with error: %v", err)
			}
			stageIndex = stageIndex + 1
		}

		if stage.ManualJudgement != nil {
			s, err = b.buildManualJudgementStage(stageIndex, stage)
			if err != nil {
				return sp, fmt.Errorf("Failed to buildManualJudgementStage with error: %v", err)
			}
			stageIndex = stageIndex + 1
		}

		if stage.DeployEmbeddedManifests != nil {
			s, err = b.buildDeployEmbeddedManifestStage(stageIndex, stage)
			if err != nil {
				return sp, fmt.Errorf("Failed to buildDeployEmbeddedManifestStage with error: %s", err)
			}
			stageIndex = stageIndex + 1
		}

		if stage.DeleteEmbeddedManifest != nil {
			s, err = b.buildDeleteEmbeddedManifestStage(stageIndex, stage)
			if err != nil {
				return sp, fmt.Errorf("Failed to buildDeleteEmbeddedManifestStage with error: %v", err)
			}
			stageIndex = stageIndex + 1
		}

		if stage.ScaleManifest != nil {
			s, err = b.buildScaleManifestStage(stageIndex, stage)
			if err != nil {
				return sp, fmt.Errorf("Failed to buildScaleManifestStage with error: %v", err)
			}
			stageIndex = stageIndex + 1
		}

		if stage.WebHook != nil {
			s, err = b.buildWebHookStage(stageIndex, stage)
			if err != nil {
				return sp, fmt.Errorf("Failed to webhook stage with error: %v", err)
			}
			stageIndex = stageIndex + 1
		}

		if stage.Jenkins != nil {
			s, err = b.buildJenkinsStage(stageIndex, stage)
			if err != nil {
				return sp, fmt.Errorf("Failed to build jenkins stage with error: %v", err)
			}
			stageIndex = stageIndex + 1
		}

		if stage.RunSpinnakerPipeline != nil {
			s, err = b.buildRunSpinnakerPipelineStage(stageIndex, stage)
			if err != nil {
				return sp, fmt.Errorf("Failed to build spinnaker pipeline stage with error: %v", err)
			}
			stageIndex = stageIndex + 1
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

	if len(maniStage.Files)+len(maniStage.ConfiguratorFiles) < 1 {
		return nil, ErrNoManifestFiles
	}

	// update the moniker
	if maniStage.DefaultMoniker != nil {
		ds.Moniker = types.Moniker{
			App:     maniStage.DefaultMoniker.App,
			Detail:  maniStage.DefaultMoniker.Detail,
			Stack:   maniStage.DefaultMoniker.Stack,
			Cluster: maniStage.DefaultMoniker.Cluster,
		}
	}

	parser := NewManfifestParser(b.pipeline, b.basePath)
	for _, file := range maniStage.Files {
		objs, err := parser.ManifestsFromFile(file.File)
		if err != nil {
			return nil, errors.Wrapf(err, "could not parse manifest file: %s", file.File)
		}

		ds.Manifests = append(ds.Manifests, objs...)
	}

	// Generate the configurator config map
	for _, configuratorFile := range maniStage.ConfiguratorFiles {

		file, err := ioutil.ReadFile(path.Join(b.basePath, configuratorFile.File))
		if err != nil {
			return nil, errors.Wrapf(err, "could not read from configurator manifest file: %s", configuratorFile.File)
		}

		env, ok := Stages[s.Account] // Set env based on account by default
		if len(configuratorFile.Environment) > 0 {
			env = configuratorFile.Environment // Allow an override to be set
		}

		if len(env) == 0 && !ok {
			env = "default" // If env was not set and can not be found in the Stages map, fall back to default
		}

		destFileName := configuratorFile.File + "." + env
		destFilePath := path.Join(b.basePath, destFileName)
		configuredConfigMap, err := os.Create(destFilePath)

		err = cnfgrtr.Generate(file, env, configuredConfigMap)
		if err != nil {
			os.Remove(destFilePath)
			return nil, errors.Wrapf(err, "k8s-configurator could not generate manifest file: %s for env: %s", configuratorFile.File, env)
		}

		objs, err := parser.ManifestsFromFile(destFileName)
		os.Remove(destFilePath)
		if err != nil {
			return nil, errors.Wrapf(err, "could not parse manifest file: %s", configuratorFile.File)
		}

		ds.Manifests = append(ds.Manifests, objs...)

	}

	return ds, nil
}

func (b *Builder) buildDeleteEmbeddedManifestStage(index int, s config.Stage) (*types.DeleteManifestStage, error) {
	parser := NewManfifestParser(b.pipeline, b.basePath)
	file := s.DeleteEmbeddedManifest.File

	objs, err := parser.ManifestsFromFile(file)
	if err != nil {
		return nil, errors.Wrapf(err, "could not parse manifest file: %s", file)
	}

	if len(objs) > 1 {
		return nil, fmt.Errorf("the manifest file %s declared more than one resource which cant be used in a delete embedded manifest stage", file)
	}

	if len(objs) == 0 {
		return nil, fmt.Errorf("the manifest file %s doesnt define a resource which cant be used in a delete embedded manifest stage", file)
	}

	obj := objs[0]
	mObj := obj.(metav1.Object)
	tObj := obj.(metav1.Type)

	ns := mObj.GetNamespace()
	if ns == "" {
		ns = "default"
	}

	// Set default values
	completeOtherBranchesThenFail := setDefaultIfNil(s.DeleteEmbeddedManifest.CompleteOtherBranchesThenFail, false)
	continuePipeline := setDefaultIfNil(s.DeleteEmbeddedManifest.ContinuePipeline, false)
	failPipeline := setDefaultIfNil(s.DeleteEmbeddedManifest.FailPipeline, true)
	markUnstableAsSuccessful := setDefaultIfNil(s.DeleteEmbeddedManifest.MarkUnstableAsSuccessful, false)
	waitForCompletion := setDefaultIfNil(s.DeleteEmbeddedManifest.WaitForCompletion, true)

	stage := &types.DeleteManifestStage{
		StageMetadata: buildStageMetadata(s, "deleteManifest", index, b.isLinear),
		CloudProvider: "kubernetes",
		Account:       s.Account,

		ManifestName:                  fmt.Sprintf("%s %s", tObj.GetKind(), mObj.GetName()),
		Location:                      ns,
		CompleteOtherBranchesThenFail: &completeOtherBranchesThenFail,
		ContinuePipeline:              &continuePipeline,
		FailPipeline:                  &failPipeline,
		MarkUnstableAsSuccessful:      &markUnstableAsSuccessful,
		WaitForCompletion:             &waitForCompletion,
	}

	return stage, nil
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
	// Set default values
	completeOtherBranchesThenFail := setDefaultIfNil(s.DeployEmbeddedManifests.CompleteOtherBranchesThenFail, false)
	continuePipeline := setDefaultIfNil(s.DeployEmbeddedManifests.ContinuePipeline, false)
	failPipeline := setDefaultIfNil(s.DeployEmbeddedManifests.FailPipeline, true)
	markUnstableAsSuccessful := setDefaultIfNil(s.DeployEmbeddedManifests.MarkUnstableAsSuccessful, false)
	waitForCompletion := setDefaultIfNil(s.DeployEmbeddedManifests.WaitForCompletion, true)

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
		Relationships:                 types.Relationships{LoadBalancers: []interface{}{}, SecurityGroups: []interface{}{}},
		Source:                        "text",
		CompleteOtherBranchesThenFail: &completeOtherBranchesThenFail,
		ContinuePipeline:              &continuePipeline,
		FailPipeline:                  &failPipeline,
		MarkUnstableAsSuccessful:      &markUnstableAsSuccessful,
		WaitForCompletion:             &waitForCompletion,
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
	// Set default values
	completeOtherBranchesThenFail := setDefaultIfNil(s.DeleteEmbeddedManifest.CompleteOtherBranchesThenFail, false)
	continuePipeline := setDefaultIfNil(s.DeleteEmbeddedManifest.ContinuePipeline, false)
	failPipeline := setDefaultIfNil(s.DeleteEmbeddedManifest.FailPipeline, true)
	markUnstableAsSuccessful := setDefaultIfNil(s.DeleteEmbeddedManifest.MarkUnstableAsSuccessful, false)
	waitForCompletion := setDefaultIfNil(s.DeleteEmbeddedManifest.WaitForCompletion, true)

	s.Name = "Delete " + s.Name
	dms := &types.DeleteManifestStage{
		StageMetadata: buildStageMetadata(s, "deleteManifest", index, b.isLinear),
		Account:       s.Account,
		CloudProvider: "kubernetes",
		Kinds:         []string{"Job"},
		Location:      "",
		Options: types.Options{
			Cascading: true,
		},
		CompleteOtherBranchesThenFail: &completeOtherBranchesThenFail,
		ContinuePipeline:              &continuePipeline,
		FailPipeline:                  &failPipeline,
		MarkUnstableAsSuccessful:      &markUnstableAsSuccessful,
		WaitForCompletion:             &waitForCompletion,
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

func (b *Builder) buildWebHookStage(index int, s config.Stage) (*types.Webhook, error) {
	stage := &types.Webhook{
		StageMetadata: buildStageMetadata(s, "webhook", index, b.isLinear),
		CustomHeaders: s.WebHook.CustomHeaders,
		Description:   s.WebHook.Description,
		Method:        s.WebHook.Method,
		Name:          s.WebHook.Name,
		Payload:       s.WebHook.Payload,
		URL:           s.WebHook.URL,
	}

	return stage, nil
}

func (b *Builder) buildJenkinsStage(index int, s config.Stage) (*types.JenkinsStage, error) {

	// Set default values
	master := s.Jenkins.Master
	if len(master) == 0 {
		master = "namely-jenkins"
	}

	completeOtherBranchesThenFail := setDefaultIfNil(s.Jenkins.CompleteOtherBranchesThenFail, true)
	continuePipeline := setDefaultIfNil(s.Jenkins.ContinuePipeline, true)
	failPipeline := setDefaultIfNil(s.Jenkins.FailPipeline, false)
	markUnstableAsSuccessful := setDefaultIfNil(s.Jenkins.MarkUnstableAsSuccessful, false)
	waitForCompletion := setDefaultIfNil(s.Jenkins.WaitForCompletion, true)

	stage := &types.JenkinsStage{
		StageMetadata:                 buildStageMetadata(s, "jenkins", index, b.isLinear),
		Type:                          JenkinsTrigger,
		Job:                           s.Jenkins.Job,
		Parameters:                    make(map[string]string),
		Master:                        master,
		CompleteOtherBranchesThenFail: &completeOtherBranchesThenFail,
		ContinuePipeline:              &continuePipeline,
		FailPipeline:                  &failPipeline,
		MarkUnstableAsSuccessful:      &markUnstableAsSuccessful,
		WaitForCompletion:             &waitForCompletion,
	}

	for _, p := range s.Jenkins.Parameters {
		stage.Parameters[p.Key] = p.Value
	}

	return stage, nil
}

func (b *Builder) buildRunSpinnakerPipelineStage(index int, s config.Stage) (*types.RunSpinnakerPipelineStage, error) {

	// Set default values
	completeOtherBranchesThenFail := setDefaultIfNil(s.RunSpinnakerPipeline.CompleteOtherBranchesThenFail, false)
	continuePipeline := setDefaultIfNil(s.RunSpinnakerPipeline.ContinuePipeline, false)
	failPipeline := setDefaultIfNil(s.RunSpinnakerPipeline.FailPipeline, true)
	markUnstableAsSuccessful := setDefaultIfNil(s.RunSpinnakerPipeline.MarkUnstableAsSuccessful, false)
	waitForCompletion := setDefaultIfNil(s.RunSpinnakerPipeline.WaitForCompletion, true)

	stage := &types.RunSpinnakerPipelineStage{
		StageMetadata:                 buildStageMetadata(s, "pipeline", index, b.isLinear),
		Type:                          JenkinsTrigger,
		Application:                   s.RunSpinnakerPipeline.Application,
		Pipeline:                      s.RunSpinnakerPipeline.Pipeline,
		PipelineParameters:            make(map[string]string),
		CompleteOtherBranchesThenFail: &completeOtherBranchesThenFail,
		ContinuePipeline:              &continuePipeline,
		FailPipeline:                  &failPipeline,
		MarkUnstableAsSuccessful:      &markUnstableAsSuccessful,
		WaitForCompletion:             &waitForCompletion,
	}

	for _, p := range s.RunSpinnakerPipeline.PipelineParameters {
		stage.PipelineParameters[p.Key] = p.Value
	}

	return stage, nil
}

// setDefaultIfNil is a helper function that returns defaultValue if givenValue is nil
func setDefaultIfNil(givenValue *bool, defaultValue bool) bool {
	retValue := defaultValue
	if givenValue != nil {
		retValue = *(givenValue)
	}

	return retValue
}

func (b *Builder) buildScaleManifestStage(index int, s config.Stage) (*types.ScaleManifestStage, error) {
	// Set default values
	completeOtherBranchesThenFail := setDefaultIfNil(s.ScaleManifest.CompleteOtherBranchesThenFail, false)
	continuePipeline := setDefaultIfNil(s.ScaleManifest.ContinuePipeline, false)
	failPipeline := setDefaultIfNil(s.ScaleManifest.FailPipeline, true)
	markUnstableAsSuccessful := setDefaultIfNil(s.ScaleManifest.MarkUnstableAsSuccessful, false)
	waitForCompletion := setDefaultIfNil(s.ScaleManifest.WaitForCompletion, true)

	stage := &types.ScaleManifestStage{
		StageMetadata:                 buildStageMetadata(s, "scaleManifest", index, b.isLinear),
		Account:                       s.Account,
		CloudProvider:                 "kubernetes",
		Kind:                          s.ScaleManifest.Kind,
		Location:                      s.ScaleManifest.Namespace,
		ManifestName:                  fmt.Sprintf("%s %s", s.ScaleManifest.Kind, s.ScaleManifest.Name),
		Replicas:                      s.ScaleManifest.Replicas,
		CompleteOtherBranchesThenFail: &completeOtherBranchesThenFail,
		ContinuePipeline:              &continuePipeline,
		FailPipeline:                  &failPipeline,
		MarkUnstableAsSuccessful:      &markUnstableAsSuccessful,
		WaitForCompletion:             &waitForCompletion,
	}

	return stage, nil
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
			Events:                         []interface{}{},
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
		FailPipeline:  s.ManualJudgement.FailPipeline,
		Instructions:  s.ManualJudgement.Instructions,
		Inputs:        s.ManualJudgement.Inputs,
	}
	// if global timeout override has been set
	if b.timeoutHours != 0 {
		mjs.OverrideTimeout = true
		mjs.StageTimeoutMS = int64(b.timeoutHours) * HourInMS
	}
	// if the timeout is actually set go
	if s.ManualJudgement.Timeout != 0 {
		mjs.OverrideTimeout = true
		mjs.StageTimeoutMS = int64(s.ManualJudgement.Timeout) * HourInMS
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

	metadata := types.StageMetadata{
		Name:                 s.Name,
		RefID:                refID,
		RequisiteStageRefIds: reliesOn,
		Type:                 t,
		Notifications:        notifications,
		SendNotifications:    (len(notifications) > 0),
	}

	if len(s.Condition) > 0 {
		metadata.StageEnabled = &types.OptionalStageSupport{
			Expression: s.Condition,
			Type:       "expression",
		}
	}

	return metadata
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
