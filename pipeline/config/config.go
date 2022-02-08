// Package config implements YAML configuration for the k8s-pipeliner input files
package config

import (
	"io"
	"io/ioutil"

	yaml "gopkg.in/yaml.v2"
)

// NewPipeline unmarshals a reader into a pipeline object
func NewPipeline(r io.Reader) (*Pipeline, error) {
	content, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var p Pipeline
	if err := yaml.Unmarshal(content, &p); err != nil {
		return nil, err
	}

	return &p, nil
}

// Pipeline is the high level struct that contains all of the configuration
// of a pipeline
type Pipeline struct {
	Name              string             `yaml:"name"`
	Application       string             `yaml:"application"`
	Triggers          []Trigger          `yaml:"triggers"`
	Stages            []Stage            `yaml:"stages"`
	ImageDescriptions []ImageDescription `yaml:"imageDescriptions"`

	DisableConcurrentExecutions bool   `yaml:"disableConcurrentExecutions"`
	KeepQueuedPipelines         bool   `yaml:"keepQueuedPipelines"`
	Description                 string `yaml:"description"`

	Notifications []Notification `yaml:"notifications"`
	Parameters    []Parameter    `yaml:"parameters"`
}

// Parameter defines a single parameter in a pipeline config
type Parameter struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Default     string   `yaml:"default"`
	Required    bool     `yaml:"required"`
	Options     []Option `yaml:"options"`
}

// Option contains the option value of a single parameter in a pipeline config
type Option struct {
	Value string `yaml:"value"`
}

// ImageDescription contains the description of an image that can be referenced
// from stages to inject in an image.
type ImageDescription struct {
	Name         string `yaml:"name"`
	Account      string `yaml:"account"`
	ImageID      string `yaml:"image_id"`
	Registry     string `yaml:"registry"`
	Repository   string `yaml:"repository"`
	Tag          string `yaml:"tag"`
	Organization string `yaml:"organization"`
}

// Trigger contains the fields that are relevant for
// spinnaker triggers such as jenkins or docker registry
type Trigger struct {
	Jenkins *JenkinsTrigger `yaml:"jenkins"`
	Webhook *WebhookTrigger `yaml:"webhook"`
}

// JenkinsTrigger has the fields for triggering a Jenkins job
type JenkinsTrigger struct {
	Job          string `yaml:"job"`
	Master       string `yaml:"master"`
	PropertyFile string `yaml:"propertyFile,omitempty"`
	Enabled      *bool  `yaml:"enabled"`
}

// JenkinsStage has fields for triggering a Jenkins job
type JenkinsStage struct {
	Type string `yaml:"type,omitempty"`

	Job string `yaml:"job"`
	// string:string map of parameters to pass into the build
	Parameters []PassthroughParameter `yaml:"parameters,omitempty"`

	Master string `yaml:"master"`

	CompleteOtherBranchesThenFail *bool `yaml:"completeOtherBranchesThenFail,omitempty"`
	ContinuePipeline              *bool `yaml:"continuePipeline,omitempty"`
	FailPipeline                  *bool `yaml:"failPipeline,omitempty"`
	MarkUnstableAsSuccessful      *bool `yaml:"markUnstableAsSuccessful,omitempty"`
	WaitForCompletion             *bool `yaml:"waitForCompletion,omitempty"`
}

// PassthroughParameter represents a key value pair passed to a child process
type PassthroughParameter struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

// RunSpinnakerPipelineStage represents a stage where another pipeline is executed
type RunSpinnakerPipelineStage struct {
	Type string `yaml:"type,omitempty"`

	Job string `yaml:"job"`

	Application string `yaml:"application"`
	Pipeline    string `yaml:"pipeline"`

	// string:string map of parameters to pass into the build
	PipelineParameters []PassthroughParameter `yaml:"parameters,omitempty"`

	CompleteOtherBranchesThenFail *bool `yaml:"completeOtherBranchesThenFail,omitempty"`
	ContinuePipeline              *bool `yaml:"continuePipeline,omitempty"`
	FailPipeline                  *bool `yaml:"failPipeline,omitempty"`
	MarkUnstableAsSuccessful      *bool `yaml:"markUnstableAsSuccessful,omitempty"`
	WaitForCompletion             *bool `yaml:"waitForCompletion,omitempty"`
	StageTimeoutMS                int64 `yaml:"stageTimeoutMs,omitempty"`
}

// WebhookTrigger defines how a webhook can trigger a pipeline execution
type WebhookTrigger struct {
	Enabled bool   `yaml:"enabled"`
	Source  string `yaml:"source"`
}

// WebHookStage is a stage that triggers a webhook
type WebHookStage struct {
	Name          string              `yaml:"name"`
	Description   string              `yaml:"description"`
	Method        string              `yaml:"method"`
	URL           string              `yaml:"url"`
	CustomHeaders map[string][]string `yaml:"customHeaders"`
	Payload       string              `yaml:"payload"`
}

// Stage is an individual stage within a spinnaker pipeline
// It defines what type of stage and the reference to a manifest file (if applicable)
type Stage struct {
	Account       string         `yaml:"account"`
	Name          string         `yaml:"name"`
	RefID         string         `yaml:"refId,omitempty"`
	ReliesOn      []string       `yaml:"reliesOn,omitempty"`
	Notifications []Notification `yaml:"notifications,omitempty"`
	Condition     string         `yaml:"condition,omitempty"`

	// All of the different supported stages, only one may be set
	RunJob                  *RunJobStage               `yaml:"runJob,omitempty"`
	Deploy                  *DeployStage               `yaml:"deploy,omitempty"`
	ManualJudgement         *ManualJudgementStage      `yaml:"manualJudgement,omitempty"`
	DeployEmbeddedManifests *DeployEmbeddedManifests   `yaml:"deployEmbeddedManifests,omitempty"`
	DeleteEmbeddedManifest  *DeleteEmbeddedManifest    `yaml:"deleteEmbeddedManifest,omitempty"`
	ScaleManifest           *ScaleManifest             `yaml:"scaleManifest,omitempty"`
	WebHook                 *WebHookStage              `yaml:"webHook,omitempty"`
	Jenkins                 *JenkinsStage              `yaml:"jenkins,omitempty"`
	RunSpinnakerPipeline    *RunSpinnakerPipelineStage `yaml:"spinnaker,omitempty"`
	EvaluateVariables       *EvaluateVariablesStage    `yaml:"evaluatevariables,omitempty"`
}

// Notification config from pipeline configuration on a stage or pipeline
type Notification struct {
	Address string            `yaml:"address"`
	Level   string            `yaml:"level"`
	Type    string            `yaml:"type"`
	When    []string          `yaml:"when"`
	Message map[string]string `yaml:"message"`
}

// Container is used to provide overrides to the container defined in a k8s
// manifest file
type Container struct {
	Command []string `yaml:"command"`
	Args    []string `yaml:"args"`
}

// RunJobStage is the configuration for a one off job in a spinnaker pipeline
type RunJobStage struct {
	ManifestFile      string                `yaml:"manifestFile"`
	ImageDescriptions []ImageDescriptionRef `yaml:"imageDescriptions"`

	Container          *Container    `yaml:"container"`
	PodOverrides       *PodOverrides `yaml:"podOverrides,omitempty"`
	ServiceAccountName string        `yaml:"serviceAccountName"`
	DeleteJob          bool          `yaml:"deleteJob"`
}

// DeployStage is the configuration for deploying a cluster of servers (pods)
type DeployStage struct {
	Groups []Group `yaml:"groups"`
}

// ImageDescriptionRef represents a reference to a defined ImageDescription on
// a given pipeline
type ImageDescriptionRef struct {
	Name          string `yaml:"name"`
	ContainerName string `yaml:"containerName"`
}

// Group represents a group to be deployed (Think: Kubernetes Pods). Most of the configuration
// of a group is filled out by the defined manifest file. This means things like commands, env vars,
// etc, are all pulled into the group spec for you.
type Group struct {
	ManifestFile      string                `yaml:"manifestFile"`
	ImageDescriptions []ImageDescriptionRef `yaml:"imageDescriptions"`

	MaxRemainingASGS int      `yaml:"maxRemainingASGS"`
	ScaleDown        bool     `yaml:"scaleDown"`
	Stack            string   `yaml:"stack"`
	Details          string   `yaml:"details"`
	Strategy         string   `yaml:"strategy"`
	TargetSize       int      `yaml:"targetSize"`
	LoadBalancers    []string `yaml:"loadBalancers"`

	// If overrides are provided, the group will run a check to make sure
	// the given manifest only defines one container. If it does, the given
	// overrides will be written into the spinnaker json output.
	// This is useful for using the same container image, env, etc to run in a
	// different mode like a queue consumer process that needs the same config,
	// image, but different command.
	ContainerOverrides *ContainerOverrides `yaml:"containerOverrides"`

	// PodOverrides allows you to add things like annotations to the pod
	// spec that is generated from this configuration
	PodOverrides *PodOverrides `yaml:"podOverrides,omitempty"`
}

// ManualJudgementStage is the configuration for pausing a pipeline awaiting
// manual intervention to continue it
type ManualJudgementStage struct {
	FailPipeline bool     `yaml:"failPipeline"`
	Instructions string   `yaml:"instructions"`
	Inputs       []string `yaml:"inputs"`
	Timeout      int      `yaml:"timeoutHours,omitempty"`
}

// ManifestFile represents a single manifest file
type ManifestFile struct {
	Environment string `yaml:"env,omitempty"`
	File        string `yaml:"file"`
}

// DeployEmbeddedManifests is a Kubernetes V2 provider stage configuration
// for deploying YAML manifest files
type DeployEmbeddedManifests struct {
	DefaultMoniker     *Moniker              `yaml:"defaultMoniker,omitempty"`
	ConfiguratorFiles  []ManifestFile        `yaml:"configuratorFiles,omitempty"`
	Files              []ManifestFile        `yaml:"files"`
	ContainerOverrides []*ContainerOverrides `yaml:"containerOverrides,omitempty"`

	CompleteOtherBranchesThenFail *bool `yaml:"completeOtherBranchesThenFail,omitempty"`
	ContinuePipeline              *bool `yaml:"continuePipeline,omitempty"`
	FailPipeline                  *bool `yaml:"failPipeline,omitempty"`
	MarkUnstableAsSuccessful      *bool `yaml:"markUnstableAsSuccessful,omitempty"`
	WaitForCompletion             *bool `yaml:"waitForCompletion,omitempty"`
	StageTimeoutMS                int64 `yaml:"stageTimeoutMs,omitempty"`
}

// DeleteEmbeddedManifest represents a single resource to be deleted
// that is identified automatically by the manifest file provided
// Internally, the builder uses a Delete Manifest stage that matches on
// name and type. The namespace is populated from the manifest metadata.
type DeleteEmbeddedManifest struct {
	File string `yaml:"file"`

	CompleteOtherBranchesThenFail *bool `yaml:"completeOtherBranchesThenFail,omitempty"`
	ContinuePipeline              *bool `yaml:"continuePipeline,omitempty"`
	FailPipeline                  *bool `yaml:"failPipeline,omitempty"`
	MarkUnstableAsSuccessful      *bool `yaml:"markUnstableAsSuccessful,omitempty"`
	WaitForCompletion             *bool `yaml:"waitForCompletion,omitempty"`
}

// Moniker describes a name set for a Spinnaker resource
type Moniker struct {
	App     string `yaml:"app"`
	Cluster string `yaml:"cluster"`
	Detail  string `yaml:"detail"`
	Stack   string `yaml:"stack"`
}

// ScaleManifest is a Kubernetes V2 provider stage configuration
// for scaling a Kubernetes object
type ScaleManifest struct {
	Kind      string `yaml:"kind"`
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
	Replicas  int    `yaml:"replicas"`

	CompleteOtherBranchesThenFail *bool `yaml:"completeOtherBranchesThenFail,omitempty"`
	ContinuePipeline              *bool `yaml:"continuePipeline,omitempty"`
	FailPipeline                  *bool `yaml:"failPipeline,omitempty"`
	MarkUnstableAsSuccessful      *bool `yaml:"markUnstableAsSuccessful,omitempty"`
	WaitForCompletion             *bool `yaml:"waitForCompletion,omitempty"`
}

// Resources represents a set of resources to use for each container
type Resources struct {
	Requests *Resource `yaml:"requests,omitempty"`
	Limits   *Resource `yaml:"limits,omitempty"`
}

// Resource represent the cpu and memory of a resource
type Resource struct {
	Memory string `yaml:"memory,omitempty"`
	CPU    string `yaml:"cpu,omitempty"`
}

// ContainerOverrides are used to override a containers values for simple
// values like the command and arguments
type ContainerOverrides struct {
	Name      string     `yaml:"name"`
	Args      []string   `yaml:"args,omitempty"`
	Command   []string   `yaml:"command,omitempty"`
	Resources *Resources `yaml:"resources,omitempty"`
}

// PodOverrides are used to override certain attributes about a pod spec
// but defined from a pipeline.yml file
type PodOverrides struct {
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

// ContainerScaffold is used to make it easy to get a file and image ref
// so you can build multiple types of stages (run job or deploys)
type ContainerScaffold interface {
	GetTargetSize() int
	Manifest() string
	ImageDescriptionRef(containerName string) *ImageDescriptionRef
}

var _ ContainerScaffold = Group{}
var _ ContainerScaffold = RunJobStage{}

// Manifest implements ContainerScaffold
func (g Group) Manifest() string { return g.ManifestFile }

// ImageDescriptionRef implements ContainerScaffold
func (g Group) ImageDescriptionRef(containerName string) *ImageDescriptionRef {
	return findImageDescription(containerName, g.ImageDescriptions)
}

// GetTargetSize returns target size of manifest group
func (g Group) GetTargetSize() int { return g.TargetSize }

// Manifest implements ContainerScaffold
func (rj RunJobStage) Manifest() string { return rj.ManifestFile }

// ImageDescriptionRef implements ContainerScaffold
func (rj RunJobStage) ImageDescriptionRef(containerName string) *ImageDescriptionRef {
	return findImageDescription(containerName, rj.ImageDescriptions)
}

// GetTargetSize returns target size of manifest group
func (rj RunJobStage) GetTargetSize() int { return 1 }

func findImageDescription(containerName string, refs []ImageDescriptionRef) *ImageDescriptionRef {
	for _, r := range refs {
		if r.ContainerName == containerName {
			return &r
		}
	}

	return nil
}

// EvaluateVariablesStage parses complex expressions for reuse throughout a pipeline
type EvaluateVariablesStage struct {
	Type      string                 `yaml:"type,omitempty"`
	Variables []PassthroughParameter `yaml:"variables,omitempty"`
}
