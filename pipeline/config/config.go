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
	Paramters     []Parameter    `yaml:"parameters"`
}

// Parameter defines a single parameter in a pipeline config
type Parameter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Default     string `yaml:"default"`
	Required    bool   `yaml:"required"`
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

// JenkinsTrigger has all of the fields defining how a trigger
// for a CI build should occur
type JenkinsTrigger struct {
	Job          string `yaml:"job"`
	Master       string `yaml:"master"`
	PropertyFile string `yaml:"propertyFile"`
	Enabled      *bool  `yaml:"enabled"`
}

// WebhookTrigger defines how a webhook can trigger a pipeline execution
type WebhookTrigger struct {
	Enabled bool   `yaml:"enabled"`
	Source  string `yaml:"source"`
}

// Stage is an individual stage within a spinnaker pipeline
// It defines what type of stage and the reference to a manifest file (if applicable)
type Stage struct {
	Account       string         `yaml:"account"`
	Name          string         `yaml:"name"`
	RefID         string         `yaml:"refId,omitempty"`
	ReliesOn      []string       `yaml:"reliesOn,omitempty"`
	Notifications []Notification `yaml:"notifications,omitempty"`

	// All of the different supported stages, only one may be set
	RunJob          *RunJobStage          `yaml:"runJob,omitempty"`
	Deploy          *DeployStage          `yaml:"deploy,omitempty"`
	ManualJudgement *ManualJudgementStage `yaml:"manualJudgement,omitempty"`
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
	MaxRemainingASGS  int                   `yaml:"maxRemainingASGS"`
	ScaleDown         bool                  `yaml:"scaleDown"`
	Stack             string                `yaml:"stack"`
	Details           string                `yaml:"details"`
	Strategy          string                `yaml:"strategy"`
	TargetSize        int                   `yaml:"targetSize"`
	LoadBalancers     []string              `yaml:"loadBalancers"`
	Deployment        *Deployment           `yaml:"deployment,omitempty"`

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

// Deployment allows you to create a server group of type deployment instead of replica sets
type Deployment struct {
	DeploymentStrategy   DeploymentStrategy `yaml:"deploymentStrategy"`
	Enabled              bool               `yaml:"enabled"`
	MinReadySeconds      int                `yaml:"minReadySeconds"`
	RevisionHistoryLimit int                `yaml:"revisionHistoryLimit"`
}

// DeploymentStrategy are generally either of type rolling update or recreate. Rolling update is preferred.
type DeploymentStrategy struct {
	RollingUpdate RollingUpdate `yaml:"rollingUpdate"`
}

// RollingUpdate can have a maximum number of new pods and unavailable pods in type string.
type RollingUpdate struct {
	MaxSurge       int `yaml:"maxSurge"`
	MaxUnavailable int `yaml:"maxUnavailable"`
}

// ManualJudgementStage is the configuration for pausing a pipeline awaiting
// manual intervention to continue it
type ManualJudgementStage struct {
	FailPipeline bool     `yaml:"failPipeline"`
	Instructions string   `yaml:"instructions"`
	Inputs       []string `yaml:"inputs"`
}

// ContainerOverrides are used to override a containers values for simple
// values like the command and arguments
type ContainerOverrides struct {
	Args    []string `yaml:"args,omitempty"`
	Command []string `yaml:"command,omitempty"`
}

// PodOverrides are used to override certain attributes about a pod spec
// but defined from a pipeline.yml file
type PodOverrides struct {
	Annotations map[string]string `yaml:"annotations",omitempty`
}

// ContainerScaffold is used to make it easy to get a file and image ref
// so you can build multiple types of stages (run job or deploys)
type ContainerScaffold interface {
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

// Manifest implements ContainerScaffold
func (rj RunJobStage) Manifest() string { return rj.ManifestFile }

// ImageDescriptionRef implements ContainerScaffold
func (rj RunJobStage) ImageDescriptionRef(containerName string) *ImageDescriptionRef {
	return findImageDescription(containerName, rj.ImageDescriptions)
}

func findImageDescription(containerName string, refs []ImageDescriptionRef) *ImageDescriptionRef {
	for _, r := range refs {
		if r.ContainerName == containerName {
			return &r
		}
	}

	return nil
}
