package builder

import (
	"encoding/json"

	"github.com/namely/k8s-pipeliner/pipeline/builder/types"
	"github.com/namely/k8s-pipeliner/pipeline/config"
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

	return json.Marshal(sp)
}
