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
	Account      string     `yaml:"account"`
	Name         string     `yaml:"name"`
	Type         string     `yaml:"type"`
	ManifestFile *string    `yaml:"manifestFile"`
	Container    *Container `yaml:"container"`
}

// Container is used to provide overrides to the container defined in a k8s
// manifest file
type Container struct {
	Command []string `yaml:"command"`
	Args    []string `yaml:"args"`
}
