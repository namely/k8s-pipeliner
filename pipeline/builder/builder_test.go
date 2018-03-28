package builder_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/namely/k8s-pipeliner/pipeline/builder"
	"github.com/namely/k8s-pipeliner/pipeline/builder/types"
	"github.com/namely/k8s-pipeliner/pipeline/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilderAssignsNotifications(t *testing.T) {
	pipeline := &config.Pipeline{
		Notifications: []config.Notification{
			{
				Address: "#launchpad",
				Level:   "pipeline",
				Type:    "slack",
				Message: map[string]string{"pipeline.complete": "Pipeline Completed!"},
				When:    []string{"pipeline.complete"},
			},
		},
	}

	builder := builder.New(pipeline)
	spinnaker, err := builder.Pipeline()
	require.NoError(t, err, "error generating pipeline json")
	require.Len(t, spinnaker.Notifications, 1)

	notification := spinnaker.Notifications[0]
	assert.Equal(t, "#launchpad", notification.Address)
	assert.Equal(t, "pipeline", notification.Level)
	assert.Equal(t, "slack", notification.Type)
}

func TestBuilderAssignsPipelineConfiguration(t *testing.T) {
	pipeline := &config.Pipeline{
		DisableConcurrentExecutions: true,
		KeepQueuedPipelines:         true,
		Description:                 "fake description",
	}

	builder := builder.New(pipeline)
	spinnaker, err := builder.Pipeline()
	require.NoError(t, err, "error generating pipeline json")

	assert.True(t, spinnaker.KeepWaitingPipelines)
	assert.True(t, spinnaker.LimitConcurrent)
	assert.Equal(t, pipeline.Description, spinnaker.Description)
}

func TestBuilderAssignsRequisiteStageRefIds(t *testing.T) {
	wd, _ := os.Getwd()
	file := filepath.Join(wd, "testdata", "deployment.full.yml")

	pipeline := &config.Pipeline{
		DisableConcurrentExecutions: true,
		KeepQueuedPipelines:         true,
		Description:                 "fake description",
		Stages: 										 []config.Stage{
			{
				Name: "Test Stage",
				Deploy: &config.DeployStage{
					Groups: []config.Group{
						{
							ManifestFile: file,
						},
					},
				},
			},
		},
	}

	builder := builder.New(pipeline)
	spinnaker, err := builder.Pipeline()
	require.NoError(t, err, "error generating pipeline json")

	assert.Equal(t, []string{}, spinnaker.Stages[0].(*types.DeployStage).StageMetadata.RequisiteStageRefIds)
}
