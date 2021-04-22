package builder_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/namely/k8s-pipeliner/pipeline/builder"
	"github.com/namely/k8s-pipeliner/pipeline/config"
)

type scaffoldMock struct {
	manifest             string
	imageDescriptionRefs []config.ImageDescriptionRef
	TargetSize           int
}

func (sm scaffoldMock) Manifest() string {
	return sm.manifest
}

func (sm scaffoldMock) GetTargetSize() int {
	return sm.TargetSize
}

func (sm scaffoldMock) ImageDescriptionRef(containerName string) *config.ImageDescriptionRef {
	for _, ref := range sm.imageDescriptionRefs {
		if ref.ContainerName == containerName {
			return &ref
		}
	}
	return nil
}

func TestContainersFromManifests(t *testing.T) {
	wd, _ := os.Getwd()

	t.Run("Deployment manifests are returned correctly", func(t *testing.T) {
		file := filepath.Join(wd, "testdata", "deployment.full.yml")
		parser := builder.NewManfifestParser(&config.Pipeline{
			ImageDescriptions: []config.ImageDescription{
				{
					Name:    "test-ref",
					ImageID: "this-is-the-image-id",
				},
			},
		})
		group, err := parser.ContainersFromScaffold(scaffoldMock{
			manifest: file,
			imageDescriptionRefs: []config.ImageDescriptionRef{
				{
					Name:          "test-ref",
					ContainerName: "hcm",
				},
			},
		})

		require.NoError(t, err, "error on retrieving the deployment manifests")

		assert.Len(t, group.Containers, 2)
		assert.Len(t, group.InitContainers, 1)
		assert.Len(t, group.Annotations, 2)
		assert.Equal(t, "fake-namespace", group.Namespace)

		t.Run("Container VolumeMounts are copied in", func(t *testing.T) {
			c := group.Containers[0]

			require.Len(t, c.VolumeMounts, 1)
			assert.Equal(t, "configmap-volume", c.VolumeMounts[0].Name)
			assert.Equal(t, "/thisisthemount", c.VolumeMounts[0].MountPath)
			assert.Equal(t, true, c.VolumeMounts[0].ReadOnly)
		})

		t.Run("Container image descriptions are returned correctly", func(t *testing.T) {
			c := group.Containers[0]

			assert.Equal(t, "this-is-the-image-id", c.ImageDescription.ImageID)
		})

		t.Run("Pod spec annotations are copied into the group", func(t *testing.T) {
			assert.Equal(t, "annotations", group.PodAnnotations["test"])
		})
	})

	t.Run("Deployments schemes are converted to latest", func(t *testing.T) {
		file := filepath.Join(wd, "testdata", "deployment.v1beta1.yml")
		parser := builder.NewManfifestParser(&config.Pipeline{})
		group, err := parser.ContainersFromScaffold(scaffoldMock{
			manifest: file,
		})

		require.NoError(t, err, "error on retrieving the deployment manifests")

		assert.Len(t, group.Containers, 1)
		assert.Len(t, group.Annotations, 2)
		assert.Equal(t, "fake-namespace", group.Namespace)
	})

	t.Run("V2 Provider Deployment", func(t *testing.T) {
		file := filepath.Join(wd, "testdata", "deployment.full.yml")
		parser := builder.NewManfifestParser(&config.Pipeline{
			ImageDescriptions: []config.ImageDescription{
				{
					Name:    "test-ref",
					ImageID: "this-is-the-v2-image",
				},
				{
					Name:    "init-v2",
					ImageID: "this-is-the-v2-init-image",
				},
			},
		})
		group, err := parser.ManifestFromScaffold(scaffoldMock{
			manifest: file,
			imageDescriptionRefs: []config.ImageDescriptionRef{
				{
					Name:          "test-ref",
					ContainerName: "test-container",
				},
				{
					Name:          "init-v2",
					ContainerName: "init-container",
				},
			},
			TargetSize: 5,
		})

		require.NoError(t, err, "error on retrieving the deployment manifest")
		assert.Equal(t, "apps/v1, Kind=Deployment", group.GetObjectKind().GroupVersionKind().String())
	})
	t.Run("V2 Provider Pod Spec", func(t *testing.T) {
		file := filepath.Join(wd, "testdata", "podspec.yml")
		parser := builder.NewManfifestParser(&config.Pipeline{})
		group, err := parser.ManifestFromScaffold(scaffoldMock{
			manifest: file,
		})

		require.NoError(t, err, "error on retrieving the pod manifest")
		assert.Equal(t, "/v1, Kind=Pod", group.GetObjectKind().GroupVersionKind().String())
	})
	t.Run("V2 Provider Configmap Spec", func(t *testing.T) {
		file := filepath.Join(wd, "testdata", "cm.yml")
		parser := builder.NewManfifestParser(&config.Pipeline{})
		group, err := parser.ManifestFromScaffold(scaffoldMock{
			manifest: file,
		})

		require.NoError(t, err, "error on retrieving the configmap manifest")
		assert.Equal(t, "/v1, Kind=ConfigMap", group.GetObjectKind().GroupVersionKind().String())
	})

	t.Run("PodSpecs work as manifest references", func(t *testing.T) {
		file := filepath.Join(wd, "testdata", "podspec.yml")
		parser := builder.NewManfifestParser(&config.Pipeline{})
		group, err := parser.ContainersFromScaffold(scaffoldMock{
			manifest: file,
		})

		require.NoError(t, err, "error on retrieving the deployment manifests")

		assert.Len(t, group.Containers, 1)
		assert.Len(t, group.Annotations, 2)
		assert.Equal(t, "fake-namespace", group.Namespace)
	})

	t.Run("Volume sources are copied", func(t *testing.T) {
		file := filepath.Join(wd, "testdata", "deployment.full.yml")
		parser := builder.NewManfifestParser(&config.Pipeline{})
		group, err := parser.ContainersFromScaffold(scaffoldMock{
			manifest: file,
		})
		require.NoError(t, err)
		require.Len(t, group.VolumeSources, 4)

		t.Run("ConfigMaps are copied", func(t *testing.T) {
			cms := group.VolumeSources[0]
			require.NotNil(t, cms.ConfigMap)
			assert.Equal(t, cms.Type, "CONFIGMAP")
		})

		t.Run("Secrets are copied", func(t *testing.T) {
			sec := group.VolumeSources[1]
			require.NotNil(t, sec.Secret)
			assert.Equal(t, sec.Type, "SECRET")
		})

		t.Run("EmptyDirs are copied", func(t *testing.T) {
			ed := group.VolumeSources[2]
			require.NotNil(t, ed.EmptyDir)
			assert.Equal(t, ed.Type, "EMPTYDIR")
		})

		t.Run("PersistentVolumeClaims are copied", func(t *testing.T) {
			vs := group.VolumeSources[3]
			require.NotNil(t, vs.PersistentVolumeClaim)
			assert.Equal(t, "PERSISTENTVOLUMECLAIM", vs.Type)
			assert.Equal(t, "persistent-volume-claim", vs.Name)
			assert.Equal(t, "my-claim-name", vs.PersistentVolumeClaim.ClaimName)
		})
	})

	t.Run("EnvFrom sources are copied in", func(t *testing.T) {
		file := filepath.Join(wd, "testdata", "deployment.envfrom.yml")
		parser := builder.NewManfifestParser(&config.Pipeline{})
		group, err := parser.ContainersFromScaffold(scaffoldMock{
			manifest: file,
		})
		require.NoError(t, err)

		container := group.Containers[0]
		require.Len(t, container.EnvFrom, 2)
		require.NotNil(t, container.EnvFrom[0].ConfigMapSource)
		assert.Equal(t, "dummy-ref", container.EnvFrom[0].ConfigMapSource.Name)
		assert.Equal(t, "some-prefix", container.EnvFrom[0].Prefix)

		secretSource := container.EnvFrom[1]
		require.NotNil(t, secretSource.SecretSource)
		assert.Equal(t, "secret-ref", secretSource.SecretSource.Name)
		assert.Equal(t, "secret-prefix", secretSource.Prefix)
	})

	t.Run("Optional Configmaps/secrets are copied in", func(t *testing.T) {
		file := filepath.Join(wd, "testdata", "deployment.optional.yml")
		parser := builder.NewManfifestParser(&config.Pipeline{})
		group, err := parser.ContainersFromScaffold(scaffoldMock{
			manifest: file,
		})
		require.NoError(t, err)

		container := group.Containers[0]
		require.Len(t, container.EnvVars, 3)
		require.NotNil(t, container.EnvVars[0].EnvSource.SecretSource.Optional)
		require.NotNil(t, container.EnvVars[1].EnvSource.ConfigMapSource.Optional)
		require.NotNil(t, container.EnvVars[2].EnvSource.ConfigMapSource.Optional)
		assert.Equal(t, false, container.EnvVars[0].EnvSource.SecretSource.Optional)
		assert.Equal(t, true, container.EnvVars[1].EnvSource.ConfigMapSource.Optional)
		assert.Equal(t, false, container.EnvVars[2].EnvSource.ConfigMapSource.Optional)
	})

	t.Run("FieldRefs are copied in correctly", func(t *testing.T) {
		file := filepath.Join(wd, "testdata", "deployment.fieldref.yml")
		parser := builder.NewManfifestParser(&config.Pipeline{})
		group, err := parser.ContainersFromScaffold(scaffoldMock{
			manifest: file,
		})
		require.NoError(t, err)

		container := group.Containers[0]
		require.Len(t, container.EnvVars, 1)

		ev := container.EnvVars[0]
		assert.Equal(t, ev.Name, "MY_ENV")
		assert.NotNil(t, ev.EnvSource.FieldRefSource)
	})

	t.Run("SecurityContext is copied in the correct format", func(t *testing.T) {
		file := filepath.Join(wd, "testdata", "deployment.security.yml")
		parser := builder.NewManfifestParser(&config.Pipeline{})
		group, err := parser.ContainersFromScaffold(scaffoldMock{
			manifest: file,
		})
		require.NoError(t, err)

		container := group.Containers[0]
		require.NotNil(t, container.SecurityContext)

		assert.True(t, *container.SecurityContext.Privileged)
		assert.True(t, *container.SecurityContext.ReadOnlyRootFileSystem)
		assert.Equal(t, int64(1337), *container.SecurityContext.RunAsUser)

		require.NotNil(t, container.SecurityContext.Capabilities)
		assert.Equal(t, []string{"FOO"}, container.SecurityContext.Capabilities.Add)
		assert.Equal(t, []string{"BAR"}, container.SecurityContext.Capabilities.Drop)
	})

	t.Run("LivenessProbe is copied in the correct format", func(t *testing.T) {
		file := filepath.Join(wd, "testdata", "deployment.probes.yml")
		parser := builder.NewManfifestParser(&config.Pipeline{})
		group, err := parser.ContainersFromScaffold(scaffoldMock{
			manifest: file,
		})
		require.NoError(t, err)

		container := group.Containers[0]

		require.NotNil(t, container.LivenessProbe)
		require.NotNil(t, container.ReadinessProbe)
	})

	t.Run("InitContainers are copied in the correct format", func(t *testing.T) {
		file := filepath.Join(wd, "testdata", "deployment.initContainer.yml")
		parser := builder.NewManfifestParser(&config.Pipeline{
			ImageDescriptions: []config.ImageDescription{
				{
					Name:    "test-ref",
					ImageID: "this-is-the-init-image-id",
				},
			},
		})

		group, err := parser.ContainersFromScaffold(scaffoldMock{
			manifest: file,
			imageDescriptionRefs: []config.ImageDescriptionRef{
				{
					Name:          "test-ref",
					ContainerName: "init-container",
				},
			},
		})
		require.NoError(t, err)

		initContainer := group.InitContainers[0]

		assert.Equal(t, "init-container", initContainer.Name)

		require.NotNil(t, initContainer.LivenessProbe)
		require.NotNil(t, initContainer.ReadinessProbe)

		t.Run("InitContainer env are copied in", func(t *testing.T) {
			require.Len(t, initContainer.EnvVars, 1)
			require.Nil(t, initContainer.EnvVars[0].EnvSource)
			assert.Equal(t, "WHATS_THE_WORD", initContainer.EnvVars[0].Name)
			assert.Equal(t, "bird is the word", initContainer.EnvVars[0].Value)
		})

		t.Run("InitContainer VolumeMounts are copied in", func(t *testing.T) {

			require.Len(t, initContainer.VolumeMounts, 1)
			assert.Equal(t, "configmap-volume", initContainer.VolumeMounts[0].Name)
			assert.Equal(t, "/thisisthemount", initContainer.VolumeMounts[0].MountPath)
			assert.Equal(t, true, initContainer.VolumeMounts[0].ReadOnly)
		})

		t.Run("InitContainer image descriptions are returned correctly", func(t *testing.T) {
			assert.Equal(t, "this-is-the-init-image-id", initContainer.ImageDescription.ImageID)
		})
	})

}

func TestEmpyYMLObject(t *testing.T) {
	wd, _ := os.Getwd()

	// testing for empty YML objects (eg: \n --- \n --- etc.)
	t.Run("empty YML objects are ignored", func(t *testing.T) {
		file := filepath.Join(wd, "testdata", "empty.yml")
		parser := builder.NewManfifestParser(&config.Pipeline{})

		_, err := parser.ManifestsFromFile(file)
		require.NoError(t, err)

	})
}
