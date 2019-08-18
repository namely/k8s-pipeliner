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

	t.Run("Triggers", func(t *testing.T) {
		t.Run("Defaults to an empty slice", func(t *testing.T) {
			pipeline := &config.Pipeline{}

			builder := builder.New(pipeline)
			spinnaker, err := builder.Pipeline()
			require.NoError(t, err, "error generating pipeline json")

			assert.Equal(t, []types.Trigger{}, spinnaker.Triggers)
		})

		t.Run("JenkinsTrigger is parsed correctly and enabled", func(t *testing.T) {
			pipeline := &config.Pipeline{
				Triggers: []config.Trigger{
					{
						Jenkins: &config.JenkinsTrigger{
							Job:          "My Job Name",
							Master:       "namely-jenkins",
							PropertyFile: ".test-ci-properties",
						},
					},
				},
			}

			builder := builder.New(pipeline)
			spinnaker, err := builder.Pipeline()
			require.NoError(t, err, "error generating pipeline json")

			assert.Len(t, spinnaker.Triggers, 1)

			assert.Equal(t, true, spinnaker.Triggers[0].(*types.JenkinsTrigger).Enabled)
			assert.Equal(t, "My Job Name", spinnaker.Triggers[0].(*types.JenkinsTrigger).Job)
			assert.Equal(t, "namely-jenkins", spinnaker.Triggers[0].(*types.JenkinsTrigger).Master)
			assert.Equal(t, ".test-ci-properties", spinnaker.Triggers[0].(*types.JenkinsTrigger).PropertyFile)
			assert.Equal(t, "jenkins", spinnaker.Triggers[0].(*types.JenkinsTrigger).Type)
		})
		t.Run("JenkinsTrigger is disabled", func(t *testing.T) {
			var enabled *bool
			enabled = newFalse()
			pipeline := &config.Pipeline{
				Triggers: []config.Trigger{
					{
						Jenkins: &config.JenkinsTrigger{
							Job:          "My Job Name",
							Master:       "namely-jenkins",
							PropertyFile: ".test-ci-properties",
							Enabled:      enabled,
						},
					},
				},
			}

			builder := builder.New(pipeline)
			spinnaker, err := builder.Pipeline()
			require.NoError(t, err, "error generating pipeline json")

			assert.Len(t, spinnaker.Triggers, 1)

			jt := spinnaker.Triggers[0].(*types.JenkinsTrigger)
			assert.Equal(t, false, jt.Enabled)
			assert.Equal(t, "My Job Name", jt.Job)
			assert.Equal(t, "namely-jenkins", jt.Master)
			assert.Equal(t, ".test-ci-properties", jt.PropertyFile)
			assert.Equal(t, "jenkins", jt.Type)
		})

		t.Run("WebhooksTrigger is configured correctly", func(t *testing.T) {
			pipeline := &config.Pipeline{
				Triggers: []config.Trigger{
					{
						Webhook: &config.WebhookTrigger{
							Source:  "this-is-a-test",
							Enabled: true,
						},
					},
				},
			}

			b := builder.New(pipeline)
			spinnaker, err := b.Pipeline()
			require.NoError(t, err, "error generating pipeline json")

			assert.Len(t, spinnaker.Triggers, 1)

			whTrigger := spinnaker.Triggers[0].(*types.WebhookTrigger)
			assert.Equal(t, true, whTrigger.Enabled)
			assert.Equal(t, builder.WebhookTrigger, whTrigger.Type)
			assert.Equal(t, "this-is-a-test", whTrigger.Source)
		})
	})

	t.Run("Parameter configuration is parsed correctly", func(t *testing.T) {
		pipeline := &config.Pipeline{
			Parameters: []config.Parameter{
				{
					Name:        "param1",
					Description: "parameter description",
					Default:     "default value",
					Required:    true,
				},
			},
		}

		b := builder.New(pipeline)
		spinnaker, err := b.Pipeline()
		require.NoError(t, err, "error generating pipeline json")

		require.Len(t, spinnaker.Parameters, 1)

		param := spinnaker.Parameters[0]
		assert.Equal(t, true, param.Required)
		assert.Equal(t, "parameter description", param.Description)
		assert.Equal(t, "param1", param.Name)
		assert.Equal(t, "default value", param.Default)
	})

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
									PodOverrides: &config.PodOverrides{
										Annotations: map[string]string{"hello": "world"},
									},
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

			expected := map[string]string{"hello": "world", "test": "annotations"}
			assert.Equal(t, expected, spinnaker.Stages[0].(*types.DeployStage).Clusters[0].PodAnnotations)
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

		t.Run("PodAnnotations are assigned when overrides", func(t *testing.T) {
			pipeline := &config.Pipeline{
				Stages: []config.Stage{
					{
						ReliesOn: []string{"2"},
						RunJob: &config.RunJobStage{
							ManifestFile: file,
							PodOverrides: &config.PodOverrides{
								Annotations: map[string]string{"hello": "world"},
							},
						},
					},
				},
			}

			builder := builder.New(pipeline)
			spinnaker, err := builder.Pipeline()
			require.NoError(t, err, "error generating pipeline json")

			expected := map[string]string{"hello": "world", "test": "annotations"}
			assert.Equal(t, expected, spinnaker.Stages[0].(*types.RunJobStage).Annotations)
		})

		t.Run("Image Descriptions are assigned", func(t *testing.T) {
			pipeline := &config.Pipeline{
				Stages: []config.Stage{
					{
						Name: "Test Deploy Stage",
						Deploy: &config.DeployStage{
							Groups: []config.Group{
								{
									ManifestFile: file,
									PodOverrides: &config.PodOverrides{
										Annotations: map[string]string{"hello": "world"},
									},
								},
							},
						},
					},
				},
				ImageDescriptions: []config.ImageDescription{
					{
						Name:         "hcm",
						Account:      "namely-registry",
						ImageID:      "${ parameters.docker_image }",
						Registry:     "registry.namely.land",
						Repository:   "namely/namely",
						Tag:          "${ parameters.docker_tag }",
						Organization: "namely",
					},
				},
			}
			builder := builder.New(pipeline)
			_, err := builder.Pipeline()
			require.NoError(t, err, "error generating pipeline json")
		})

		t.Run("ServiceAccountName is assigned", func(t *testing.T) {
			pipeline := &config.Pipeline{
				Stages: []config.Stage{
					{
						ReliesOn: []string{"2"},
						RunJob: &config.RunJobStage{
							ManifestFile:       file,
							ServiceAccountName: "test-svc-acc",
						},
					},
				},
			}

			builder := builder.New(pipeline)
			spinnaker, err := builder.Pipeline()
			require.NoError(t, err, "error generating pipeline json")

			assert.Equal(t, "test-svc-acc", spinnaker.Stages[0].(*types.RunJobStage).ServiceAccountName)
		})
	})

	t.Run("ScaleManifest stage is parsed correctly", func(t *testing.T) {
		t.Run("Properties are assigned", func(t *testing.T) {

			boolt := true
			boolf := false
			pipeline := &config.Pipeline{
				Stages: []config.Stage{
					{
						Name: "Test ScaleManifest Stage",
						ScaleManifest: &config.ScaleManifest{
							Kind:                          "deployment",
							Name:                          "mydeployname",
							Namespace:                     "mynamespace",
							Replicas:                      5,
							CompleteOtherBranchesThenFail: &boolt,
							ContinuePipeline:              &boolt,
							FailPipeline:                  &boolf,
							MarkUnstableAsSuccessful:      &boolt,
							WaitForCompletion:             &boolf,
						},
					},
				},
			}

			builder := builder.New(pipeline)
			spinnaker, err := builder.Pipeline()
			require.NoError(t, err, "error generating pipeline json")

			stg := spinnaker.Stages[0].(*types.ScaleManifestStage)
			assert.Equal(t, "Test ScaleManifest Stage", stg.Name)
			assert.Equal(t, "deployment", stg.Kind)
			assert.Equal(t, "deployment mydeployname", stg.ManifestName)
			assert.Equal(t, "mynamespace", stg.Location)
			assert.Equal(t, 5, stg.Replicas)
			assert.Equal(t, &boolt, stg.CompleteOtherBranchesThenFail)
			assert.Equal(t, &boolt, stg.ContinuePipeline)
			assert.Equal(t, &boolf, stg.FailPipeline)
			assert.Equal(t, &boolt, stg.MarkUnstableAsSuccessful)
			assert.Equal(t, &boolf, stg.WaitForCompletion)
		})

		t.Run("Default properties are assigned", func(t *testing.T) {

			boolt := true
			boolf := false
			pipeline := &config.Pipeline{
				Stages: []config.Stage{
					{
						Name: "Test ScaleManifest Stage",
						ScaleManifest: &config.ScaleManifest{
							Kind:      "deployment",
							Name:      "mydeployname",
							Namespace: "mynamespace",
							Replicas:  5,
						},
					},
				},
			}

			builder := builder.New(pipeline)
			spinnaker, err := builder.Pipeline()
			require.NoError(t, err, "error generating pipeline json")

			stg := spinnaker.Stages[0].(*types.ScaleManifestStage)
			assert.Equal(t, "Test ScaleManifest Stage", stg.Name)
			assert.Equal(t, "deployment", stg.Kind)
			assert.Equal(t, "deployment mydeployname", stg.ManifestName)
			assert.Equal(t, "mynamespace", stg.Location)
			assert.Equal(t, 5, stg.Replicas)
			assert.Equal(t, &boolf, stg.CompleteOtherBranchesThenFail)
			assert.Equal(t, &boolf, stg.ContinuePipeline)
			assert.Equal(t, &boolt, stg.FailPipeline)
			assert.Equal(t, &boolf, stg.MarkUnstableAsSuccessful)
			assert.Equal(t, &boolt, stg.WaitForCompletion)
		})

		t.Run("RequisiteStageRefIds defaults to an empty slice", func(t *testing.T) {
			pipeline := &config.Pipeline{
				Stages: []config.Stage{
					{
						ScaleManifest: &config.ScaleManifest{},
					},
				},
			}

			builder := builder.New(pipeline)
			spinnaker, err := builder.Pipeline()
			require.NoError(t, err, "error generating pipeline json")

			assert.Equal(t, []string{}, spinnaker.Stages[0].(*types.ScaleManifestStage).StageMetadata.RequisiteStageRefIds)
		})

		t.Run("RequisiteStageRefIds is assigned when ReliesOn is provided", func(t *testing.T) {
			pipeline := &config.Pipeline{
				Stages: []config.Stage{
					{
						ReliesOn:      []string{"2"},
						ScaleManifest: &config.ScaleManifest{},
					},
				},
			}

			builder := builder.New(pipeline)
			spinnaker, err := builder.Pipeline()
			require.NoError(t, err, "error generating pipeline json")

			assert.Equal(t, []string{"2"}, spinnaker.Stages[0].(*types.ScaleManifestStage).StageMetadata.RequisiteStageRefIds)
		})
	})

	t.Run("Jenkins stage is parsed correctly", func(t *testing.T) {
		t.Run("Properties are assigned", func(t *testing.T) {

			boolt := true
			boolf := false
			pipeline := &config.Pipeline{
				Stages: []config.Stage{
					{
						Name: "Test Jenkins Stage",
						Jenkins: &config.JenkinsStage{
							Type: "jenkins",
							Job:  "QA/job/stage/job/UI/job/SLI",
							Parameters: []config.PassthroughParameter{
								config.PassthroughParameter{
									Key:   "BROWSER",
									Value: "chrome",
								},
								config.PassthroughParameter{
									Key:   "Environment",
									Value: "stage",
								},
								config.PassthroughParameter{
									Key:   "NPMSCRIPT",
									Value: "test:sli",
								},
								config.PassthroughParameter{
									Key:   "timeout",
									Value: "10",
								},
							},
							Master:                        "namely-jenkins",
							CompleteOtherBranchesThenFail: &boolf,
							ContinuePipeline:              &boolf,
							FailPipeline:                  &boolt,
							MarkUnstableAsSuccessful:      &boolt,
							WaitForCompletion:             &boolf,
						},
					},
				},
			}

			builder := builder.New(pipeline)
			spinnaker, err := builder.Pipeline()
			require.NoError(t, err, "error generating pipeline json")

			stg := spinnaker.Stages[0].(*types.JenkinsStage)
			assert.Equal(t, "Test Jenkins Stage", stg.Name)
			assert.Equal(t, "jenkins", stg.Type)
			assert.Equal(t, "QA/job/stage/job/UI/job/SLI", stg.Job)

			params := stg.Parameters
			assert.Equal(t, "chrome", params["BROWSER"])
			assert.Equal(t, "stage", params["Environment"])
			assert.Equal(t, "test:sli", params["NPMSCRIPT"])
			assert.Equal(t, "10", params["timeout"])
			assert.Equal(t, "namely-jenkins", stg.Master)
			assert.Equal(t, &boolf, stg.CompleteOtherBranchesThenFail)
			assert.Equal(t, &boolf, stg.ContinuePipeline)
			assert.Equal(t, &boolt, stg.FailPipeline)
			assert.Equal(t, &boolt, stg.MarkUnstableAsSuccessful)
			assert.Equal(t, &boolf, stg.WaitForCompletion)
		})

		t.Run("Default properties are assigned", func(t *testing.T) {

			boolt := true
			boolf := false
			pipeline := &config.Pipeline{
				Stages: []config.Stage{
					{
						Name: "Test Jenkins Stage",
						Jenkins: &config.JenkinsStage{
							Type: "jenkins",
							Job:  "QA/job/stage/job/UI/job/SLI",
							Parameters: []config.PassthroughParameter{
								config.PassthroughParameter{
									Key:   "BROWSER",
									Value: "chrome",
								},
								config.PassthroughParameter{
									Key:   "Environment",
									Value: "stage",
								},
								config.PassthroughParameter{
									Key:   "NPMSCRIPT",
									Value: "test:sli",
								},
								config.PassthroughParameter{
									Key:   "timeout",
									Value: "10",
								},
							},
							Master: "namely-jenkins",
						},
					},
				},
			}

			builder := builder.New(pipeline)
			spinnaker, err := builder.Pipeline()
			require.NoError(t, err, "error generating pipeline json")

			stg := spinnaker.Stages[0].(*types.JenkinsStage)
			assert.Equal(t, "Test Jenkins Stage", stg.Name)
			assert.Equal(t, "jenkins", stg.Type)
			assert.Equal(t, "QA/job/stage/job/UI/job/SLI", stg.Job)

			params := stg.Parameters
			assert.Equal(t, "chrome", params["BROWSER"])
			assert.Equal(t, "stage", params["Environment"])
			assert.Equal(t, "test:sli", params["NPMSCRIPT"])
			assert.Equal(t, "10", params["timeout"])
			assert.Equal(t, "namely-jenkins", stg.Master)
			assert.Equal(t, &boolt, stg.CompleteOtherBranchesThenFail)
			assert.Equal(t, &boolt, stg.ContinuePipeline)
			assert.Equal(t, &boolf, stg.FailPipeline)
			assert.Equal(t, &boolf, stg.MarkUnstableAsSuccessful)
			assert.Equal(t, &boolt, stg.WaitForCompletion)
		})

		t.Run("RequisiteStageRefIds defaults to an empty slice", func(t *testing.T) {
			pipeline := &config.Pipeline{
				Stages: []config.Stage{
					{
						Jenkins: &config.JenkinsStage{},
					},
				},
			}

			builder := builder.New(pipeline)
			spinnaker, err := builder.Pipeline()
			require.NoError(t, err, "error generating pipeline json")

			assert.Equal(t, []string{}, spinnaker.Stages[0].(*types.JenkinsStage).StageMetadata.RequisiteStageRefIds)
		})

		t.Run("RequisiteStageRefIds is assigned when ReliesOn is provided", func(t *testing.T) {
			pipeline := &config.Pipeline{
				Stages: []config.Stage{
					{
						ReliesOn: []string{"2"},
						Jenkins:  &config.JenkinsStage{},
					},
				},
			}

			builder := builder.New(pipeline)
			spinnaker, err := builder.Pipeline()
			require.NoError(t, err, "error generating pipeline json")

			assert.Equal(t, []string{"2"}, spinnaker.Stages[0].(*types.JenkinsStage).StageMetadata.RequisiteStageRefIds)
		})
	})

	t.Run("RunSpinnakerPipeline stage is parsed correctly", func(t *testing.T) {
		t.Run("Properties are assigned", func(t *testing.T) {

			boolt := true
			boolf := false
			pipeline := &config.Pipeline{
				Stages: []config.Stage{
					{
						Name: "Test RunSpinnakerPipeline Stage",
						RunSpinnakerPipeline: &config.RunSpinnakerPipelineStage{
							Type:        "pipeline",
							Application: "badges",
							Pipeline:    "2c4a14d9-2f25-4a2a-b2d6-c31e596bce19",
							PipelineParameters: []config.PassthroughParameter{
								config.PassthroughParameter{
									Key:   "badges_docker_image",
									Value: "img",
								},
								config.PassthroughParameter{
									Key:   "file_data",
									Value: "data=has,date=now",
								},
								config.PassthroughParameter{
									Key:   "file_path",
									Value: "rel/ative/path",
								},
								config.PassthroughParameter{
									Key:   "service_name",
									Value: "k8s-pipeliner",
								},
							},
							CompleteOtherBranchesThenFail: &boolt,
							ContinuePipeline:              &boolt,
							FailPipeline:                  &boolf,
							MarkUnstableAsSuccessful:      &boolt,
							WaitForCompletion:             &boolf,
						},
					},
				},
			}

			builder := builder.New(pipeline)
			spinnaker, err := builder.Pipeline()
			require.NoError(t, err, "error generating pipeline json")

			stg := spinnaker.Stages[0].(*types.RunSpinnakerPipelineStage)

			assert.Equal(t, "Test RunSpinnakerPipeline Stage", stg.Name)
			assert.Equal(t, "pipeline", stg.Type)

			params := stg.PipelineParameters
			assert.Equal(t, "img", params["badges_docker_image"])
			assert.Equal(t, "data=has,date=now", params["file_data"])
			assert.Equal(t, "rel/ative/path", params["file_path"])
			assert.Equal(t, "k8s-pipeliner", params["service_name"])
			assert.Equal(t, &boolt, stg.CompleteOtherBranchesThenFail)
			assert.Equal(t, &boolt, stg.ContinuePipeline)
			assert.Equal(t, &boolf, stg.FailPipeline)
			assert.Equal(t, &boolt, stg.MarkUnstableAsSuccessful)
			assert.Equal(t, &boolf, stg.WaitForCompletion)
		})

		t.Run("Default properties are assigned", func(t *testing.T) {

			boolt := true
			boolf := false
			pipeline := &config.Pipeline{
				Stages: []config.Stage{
					{
						Name: "Test RunSpinnakerPipeline Stage",
						RunSpinnakerPipeline: &config.RunSpinnakerPipelineStage{
							Type:        "pipeline",
							Application: "badges",
							Pipeline:    "2c4a14d9-2f25-4a2a-b2d6-c31e596bce19",
							PipelineParameters: []config.PassthroughParameter{
								config.PassthroughParameter{
									Key:   "badges_docker_image",
									Value: "img",
								},
								config.PassthroughParameter{
									Key:   "file_data",
									Value: "data=has,date=now",
								},
								config.PassthroughParameter{
									Key:   "file_path",
									Value: "rel/ative/path",
								},
								config.PassthroughParameter{
									Key:   "service_name",
									Value: "k8s-pipeliner",
								},
							},
						},
					},
				},
			}

			builder := builder.New(pipeline)
			spinnaker, err := builder.Pipeline()
			require.NoError(t, err, "error generating pipeline json")

			stg := spinnaker.Stages[0].(*types.RunSpinnakerPipelineStage)

			assert.Equal(t, "Test RunSpinnakerPipeline Stage", stg.Name)
			assert.Equal(t, "pipeline", stg.Type)

			params := stg.PipelineParameters
			assert.Equal(t, "img", params["badges_docker_image"])
			assert.Equal(t, "data=has,date=now", params["file_data"])
			assert.Equal(t, "rel/ative/path", params["file_path"])
			assert.Equal(t, "k8s-pipeliner", params["service_name"])
			assert.Equal(t, &boolf, stg.CompleteOtherBranchesThenFail)
			assert.Equal(t, &boolf, stg.ContinuePipeline)
			assert.Equal(t, &boolt, stg.FailPipeline)
			assert.Equal(t, &boolf, stg.MarkUnstableAsSuccessful)
			assert.Equal(t, &boolt, stg.WaitForCompletion)
		})

		t.Run("RequisiteStageRefIds defaults to an empty slice", func(t *testing.T) {
			pipeline := &config.Pipeline{
				Stages: []config.Stage{
					{
						RunSpinnakerPipeline: &config.RunSpinnakerPipelineStage{},
					},
				},
			}

			builder := builder.New(pipeline)
			spinnaker, err := builder.Pipeline()
			require.NoError(t, err, "error generating pipeline json")

			assert.Equal(t, []string{}, spinnaker.Stages[0].(*types.RunSpinnakerPipelineStage).StageMetadata.RequisiteStageRefIds)
		})

		t.Run("RequisiteStageRefIds is assigned when ReliesOn is provided", func(t *testing.T) {
			pipeline := &config.Pipeline{
				Stages: []config.Stage{
					{
						ReliesOn:             []string{"2"},
						RunSpinnakerPipeline: &config.RunSpinnakerPipelineStage{},
					},
				},
			}

			builder := builder.New(pipeline)
			spinnaker, err := builder.Pipeline()
			require.NoError(t, err, "error generating pipeline json")

			assert.Equal(t, []string{"2"}, spinnaker.Stages[0].(*types.RunSpinnakerPipelineStage).StageMetadata.RequisiteStageRefIds)
		})
	})
}

func newFalse() *bool {
	b := false
	return &b
}
