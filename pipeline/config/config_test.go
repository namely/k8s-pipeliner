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

	assert.Equal(t, "Connect Deployment", cfg.Name)
	require.Len(t, cfg.Triggers, 1)
	require.Equal(t, "Connect/job/master", cfg.Triggers[0].Jenkins.Job)

	assert.Len(t, cfg.Stages, 2)
	assert.Equal(t, cfg.Stages[0].RunJob.ManifestFile, "manifests/deploy/connect.yml")
	assert.NotNil(t, cfg.Stages[0].RunJob.Container)

	require.Len(t, cfg.Stages[1].Deploy.Groups, 1, "no groups on deploy stage")
	assert.Equal(t, cfg.Stages[1].Deploy.Groups[0].ManifestFile, "manifests/deploy/connect.yml")

	require.Len(t, cfg.ImageDescriptions, 1, "image descriptions was empty")
}

func TestNewConfigForPodAnnotations(t *testing.T) {
	wd, _ := os.Getwd()
	file, err := os.Open(filepath.Join(wd, "testdata", "pipeline.podannotations.yaml"))
	require.Nil(t, err, "error opening testdata file")

	cfg, err := config.NewPipeline(file)
	require.Nil(t, err, "error generating new config from file reader")

	require.Len(t, cfg.Stages, 1)
	require.NotNil(t, cfg.Stages[0].RunJob)

	runJob := cfg.Stages[0].RunJob
	require.NotNil(t, runJob.PodOverrides)
	assert.Equal(t, runJob.PodOverrides.Annotations, map[string]string{"hello": "world"})
}
