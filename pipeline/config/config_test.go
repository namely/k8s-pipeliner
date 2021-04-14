package config_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/namely/k8s-pipeliner/pipeline/config"
)

func TestNewConfig(t *testing.T) {
	wd, _ := os.Getwd()
	file, err := os.Open(filepath.Join(wd, "testdata", "pipeline.full.yml"))
	require.Nil(t, err, "error opening testdata file")

	cfg, err := config.NewPipeline(file)
	require.Nil(t, err, "error generating new config from file reader")

	assert.Equal(t, "Nginx Deployment", cfg.Name)
	require.Len(t, cfg.Triggers, 1)
	require.Equal(t, "nginx/job/master", cfg.Triggers[0].Jenkins.Job)

	require.Len(t, cfg.Stages, 5)

	stage := cfg.Stages[0]
	require.NotNil(t, stage.DeployEmbeddedManifests)
	assert.Equal(t, stage.DeployEmbeddedManifests.Files[0].File, "manifests/nginx-deployment.yml")
	assert.Equal(t, stage.Name, "Deploy nginx")
	assert.Equal(t, stage.Account, "int-k8s")
	assert.Equal(t, stage.DeployEmbeddedManifests.Kubecost.Profile, "development")
	stage2 := cfg.Stages[1]
	require.NotNil(t, stage2.ManualJudgement)
	assert.Equal(t, stage2.ManualJudgement.Timeout, 100)
	stage3 := cfg.Stages[2]
	require.NotNil(t, stage3.ManualJudgement)
	assert.Equal(t, stage3.ManualJudgement.Timeout, 0)

	webHookStage := cfg.Stages[3]
	require.NotNil(t, webHookStage.WebHook)
	assert.Equal(t, "postBuildInfoToBugsnag", webHookStage.WebHook.Name)
	assert.Equal(t, "https://build.bugsnag.com/", webHookStage.WebHook.URL)
	assert.Equal(t, "POST", webHookStage.WebHook.Method)
	assert.Equal(t, `{
    "apiKey": "some_key",
    "appVersion": "some_version",
    "builderName": "some_builder",
    "releaseStage": "int",
    "sourceControl": {
    "repository": "https://github.com/namely/repository",
    "revision": "some_revision"
    }
}`, webHookStage.WebHook.Payload)
	assert.Equal(t, "Post build info to Bugsnag", webHookStage.WebHook.Description)

	deployStage := cfg.Stages[4]
	assert.Equal(t, deployStage.Deploy.Groups[0].Kubecost.Profile, "production")

	expectedHeaders := map[string][]string{"Content-Type": {"application/json"}}
	assert.True(t, reflect.DeepEqual(expectedHeaders, webHookStage.WebHook.CustomHeaders))
}
