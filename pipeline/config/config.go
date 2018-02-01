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
	Name        string    `yaml:"name"`
	Application string    `yaml:"application"`
	Triggers    []Trigger `yaml:"triggers"`
	Stages      []Stage   `yaml:"stages"`
}

// Trigger contains the fields that are relevant for
// spinnaker triggers such as jenkins or docker registry
type Trigger struct {
	Jenkins *JenkinsTrigger `yaml:"jenkins"`
}

// JenkinsTrigger has all of the fields defining how a trigger
// for a CI build should occur
type JenkinsTrigger struct {
	Job          string `yaml:"job"`
	Master       string `yaml:"master"`
	PropertyFile string `yaml:"propertyFile"`
}

// Stage is an individual stage within a spinnaker pipeline
// It defines what type of stage and the reference to a manifest file (if applicable)
type Stage struct {
	Account       string         `yaml:"account"`
	Name          string         `yaml:"name"`
	RefID         string         `yaml:"refId"`
	ReliesOn      []string       `yaml:"reliesOn"`
	Notifications []Notification `yaml:"notifications"`

	// All of the different supported stages, only one may be set
	RunJob          *RunJobStage          `yaml:"runJob"`
	Deploy          *DeployStage          `yaml:"deploy"`
	ManualJudgement *ManualJudgementStage `yaml:"manualJudgement"`
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
	ManifestFile string     `yaml:"manifestFile"`
	Container    *Container `yaml:"container"`
}

// DeployStage is the configuration for deploying a cluster of servers (pods)
type DeployStage struct {
	Groups []Group `yaml:"groups"`
}

// Group represents a group to be deployed (Think: Kubernetes Pods). Most of the configuration
// of a group is filled out by the defined manifest file. This means things like commands, env vars,
// etc, are all pulled into the group spec for you.
type Group struct {
	ManifestFile     string `yaml:"manifestFile"`
	MaxRemainingASGS int    `yaml:"maxRemainingASGS"`
	ScaleDown        bool   `yaml:"scaleDown"`
	Stack            string `yaml:"stack"`
	Strategy         string `yaml:"strategy"`
	TargetSize       int    `yaml:"targetSize"`

	// If overrides are provided, the group will run a check to make sure
	// the given manifest only defines one container. If it does, the given
	// overrides will be written into the spinnaker json output.
	// This is useful for using the same container image, env, etc to run in a
	// different mode like a queue consumer process that needs the same config,
	// image, but different command.
	ContainerOverrides *ContainerOverrides `yaml:"containerOverrides"`
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
