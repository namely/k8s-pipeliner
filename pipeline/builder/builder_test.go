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

func TestBuilderPipelineStages(t *testing.T) {
	wd, _ := os.Getwd()
	file := filepath.Join(wd, "testdata", "deployment.full.yml")

	t.Run("Deploy stage is parsed correctly", func(t *testing.T) {
		t.Run("Clusters are assigned", func(t *testing.T) {
			pipeline := &config.Pipeline{
				Stages: []config.Stage{
					{
						Name: "Test Deploy Stage",
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

			assert.Equal(t, "Test Deploy Stage", spinnaker.Stages[0].(*types.DeployStage).Name)
			assert.Len(t, spinnaker.Stages[0].(*types.DeployStage).Clusters, 1)
		})

		t.Run("RequisiteStageRefIds defaults to an empty slice", func(t *testing.T) {
			pipeline := &config.Pipeline{
				Stages: []config.Stage{
					{
						Deploy: &config.DeployStage{},
					},
				},
			}

			builder := builder.New(pipeline)
			spinnaker, err := builder.Pipeline()
			require.NoError(t, err, "error generating pipeline json")

			assert.Equal(t, []string{}, spinnaker.Stages[0].(*types.DeployStage).StageMetadata.RequisiteStageRefIds)
		})

		t.Run("RequisiteStageRefIds is assigned when ReliesOn is provided", func(t *testing.T) {
			pipeline := &config.Pipeline{
				Stages: []config.Stage{
					{
						ReliesOn: []string{"2"},
						Deploy:   &config.DeployStage{},
					},
				},
			}

			builder := builder.New(pipeline)
			spinnaker, err := builder.Pipeline()
			require.NoError(t, err, "error generating pipeline json")

			assert.Equal(t, []string{"2"}, spinnaker.Stages[0].(*types.DeployStage).StageMetadata.RequisiteStageRefIds)
		})
	})

	t.Run("RunJob stage is parsed correctly", func(t *testing.T) {
		t.Run("Name is assigned", func(t *testing.T) {
			pipeline := &config.Pipeline{
				Stages: []config.Stage{
					{
						Name: "Test RunJob Stage",
						RunJob: &config.RunJobStage{
							ManifestFile: file,
						},
					},
				},
			}

			builder := builder.New(pipeline)
			spinnaker, err := builder.Pipeline()
			require.NoError(t, err, "error generating pipeline json")

			assert.Equal(t, "Test RunJob Stage", spinnaker.Stages[0].(*types.RunJobStage).Name)
		})

		t.Run("RequisiteStageRefIds defaults to an empty slice", func(t *testing.T) {
			pipeline := &config.Pipeline{
				Stages: []config.Stage{
					{
						RunJob: &config.RunJobStage{
							ManifestFile: file,
						},
					},
				},
			}

			builder := builder.New(pipeline)
			spinnaker, err := builder.Pipeline()
			require.NoError(t, err, "error generating pipeline json")

			assert.Equal(t, []string{}, spinnaker.Stages[0].(*types.RunJobStage).StageMetadata.RequisiteStageRefIds)
		})

		t.Run("RequisiteStageRefIds is assigned when ReliesOn is provided", func(t *testing.T) {
			pipeline := &config.Pipeline{
				Stages: []config.Stage{
					{
						ReliesOn: []string{"2"},
						RunJob: &config.RunJobStage{
							ManifestFile: file,
						},
					},
				},
			}

			builder := builder.New(pipeline)
			spinnaker, err := builder.Pipeline()
			require.NoError(t, err, "error generating pipeline json")

			assert.Equal(t, []string{"2"}, spinnaker.Stages[0].(*types.RunJobStage).StageMetadata.RequisiteStageRefIds)
		})
	})

	t.Run("ManualJudgement stage is parsed correctly", func(t *testing.T) {
		t.Run("Name is assigned", func(t *testing.T) {
			pipeline := &config.Pipeline{
				Stages: []config.Stage{
					{
						Name:            "Test ManualJudgementStage Stage",
						ManualJudgement: &config.ManualJudgementStage{},
					},
				},
			}

			builder := builder.New(pipeline)
			spinnaker, err := builder.Pipeline()
			require.NoError(t, err, "error generating pipeline json")

			assert.Equal(t, "Test ManualJudgementStage Stage", spinnaker.Stages[0].(*types.ManualJudgementStage).Name)
		})

		t.Run("RequisiteStageRefIds defaults to an empty slice", func(t *testing.T) {
			pipeline := &config.Pipeline{
				Stages: []config.Stage{
					{
						ManualJudgement: &config.ManualJudgementStage{},
					},
				},
			}

			builder := builder.New(pipeline)
			spinnaker, err := builder.Pipeline()
			require.NoError(t, err, "error generating pipeline json")

			assert.Equal(t, []string{}, spinnaker.Stages[0].(*types.ManualJudgementStage).StageMetadata.RequisiteStageRefIds)
		})

		t.Run("RequisiteStageRefIds is assigned when ReliesOn is provided", func(t *testing.T) {
			pipeline := &config.Pipeline{
				Stages: []config.Stage{
					{
						ReliesOn:        []string{"2"},
						ManualJudgement: &config.ManualJudgementStage{},
					},
				},
			}

			builder := builder.New(pipeline)
			spinnaker, err := builder.Pipeline()
			require.NoError(t, err, "error generating pipeline json")

			assert.Equal(t, []string{"2"}, spinnaker.Stages[0].(*types.ManualJudgementStage).StageMetadata.RequisiteStageRefIds)
		})
	})
}
