package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/namely/k8s-pipeliner/pipeline/config"
)

func TestNewConfig(t *testing.T) {
	wd, _ := os.Getwd()
	file, err := os.Open(filepath.Join(wd, "testdata", "pipeline.full.yaml"))
	require.Nil(t, err, "error opening testdata file")

	cfg, err := config.NewPipeline(file)
	require.Nil(t, err, "error generating new config from file reader")

	assert.Equal(t, "Nginx Deployment", cfg.Name)
	require.Len(t, cfg.Triggers, 1)
	require.Equal(t, "nginx/job/master", cfg.Triggers[0].Jenkins.Job)

	require.Len(t, cfg.Stages, 3)

	stage := cfg.Stages[0]
	require.NotNil(t, stage.DeployEmbeddedManifests)
	assert.Equal(t, stage.DeployEmbeddedManifests.Files[0].File, "manifests/nginx-deployment.yml")
	assert.Equal(t, stage.Name, "Deploy nginx")
	assert.Equal(t, stage.Account, "int-k8s")
	stage2 := cfg.Stages[1]
	require.NotNil(t, stage2.ManualJudgement)
	assert.Equal(t, stage2.ManualJudgement.Timeout, 100)
	stage3 := cfg.Stages[2]
	require.NotNil(t, stage3.ManualJudgement)
	assert.Equal(t, stage3.ManualJudgement.Timeout, 0)
}
