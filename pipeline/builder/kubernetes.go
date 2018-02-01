package builder

import (
	"errors"
	"io/ioutil"
	"os"
	"strings"

	"github.com/namely/k8s-pipeliner/pipeline/builder/types"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

var (
	// ErrUnsupportedManifest is returned when a given kubernetes manifest file
	// is not supported
	ErrUnsupportedManifest = errors.New("builder: manifest type is not supported")
)

const (
	// SpinnakerImageDescriptionAccountAnnotation is used for injecting in the docker registry
	// that should be used when generating the imageDescription struct on a container
	// field. This should match a docker registry account you've added to spinnaker
	SpinnakerImageDescriptionAccountAnnotation = "namely.com/spinnaker-image-description-account"

	// SpinnakerImageDescriptionImageIDAnnotation represents the whole repository
	// Example: registry.namely.com/namely/namely:latest
	SpinnakerImageDescriptionImageIDAnnotation = "namely.com/spinnaker-image-description-imageid"

	// SpinnakerImageDescriptionRegistryAnnotation is the registry host
	// Example: registry.namely.com
	SpinnakerImageDescriptionRegistryAnnotation = "namely.com/spinnaker-image-description-registry"

	// SpinnakerImageDescriptionRepositoryAnnotation is the user / repository name
	// Example: "namely/namely"
	SpinnakerImageDescriptionRepositoryAnnotation = "namely.com/spinnaker-image-description-repository"

	// SpinnakerImageDescriptionTagAnnotation is the tag portion of the image ID
	// Example: "latest"
	SpinnakerImageDescriptionTagAnnotation = "namely.com/spinnaker-image-description-tag"

	// SpinnakerImageDescriptionOrganizationAnnotation is the registry org that owns the image.
	// Example: "namely" (where registry.namely.land/namely <- is the org)
	SpinnakerImageDescriptionOrganizationAnnotation = "namely.com/spinnaker-image-description-organization"
)

// ManifestGroup keeps a collection of containers from a deployment
// and metadata associated with them
type ManifestGroup struct {
	Namespace     string
	Annotations   map[string]string
	Containers    []*types.Container
	VolumeSources []*types.VolumeSource
}

// ContainersFromManifest loads a kubernetes manifest file and generates
// spinnaker pipeline containers config from it.
//
// NOTE: If your manifest file declares multiple types, only the first will be taken
// to generate the config.
func ContainersFromManifest(file string) (*ManifestGroup, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, g, err := decode(b, nil, nil)
	if err != nil {
		return nil, err
	}

	var mg ManifestGroup
	var resource runtime.Object

	if g.Kind == "Deployment" {
		resource = &appsv1.Deployment{}
		if err := scheme.Scheme.Convert(obj, resource, nil); err != nil {
			return nil, err
		}
	}

	switch t := resource.(type) {
	case *appsv1.Deployment:
		mg.Containers = deploymentContainers(t)
		mg.Annotations = t.Annotations
		mg.Namespace = t.Namespace
		mg.VolumeSources = volumeSources(t.Spec.Template.Spec.Volumes)
	default:
		return nil, ErrUnsupportedManifest
	}

	return &mg, nil
}

// converts kubernetes volume sources into builder types
func volumeSources(vols []corev1.Volume) []*types.VolumeSource {
	var vs []*types.VolumeSource

	for _, vol := range vols {
		spinVol := &types.VolumeSource{
			Name: vol.Name,
		}

		if cm := vol.ConfigMap; cm != nil {
			spinVol.ConfigMap = &types.ConfigMapVolumeSource{
				ConfigMapName: cm.Name,
				Items:         cm.Items,
				DefaultMode:   cm.DefaultMode,
			}
			spinVol.Type = "CONFIGMAP"
		}

		if sec := vol.Secret; sec != nil {
			spinVol.Secret = &types.SecretVolumeSource{
				SecretName: sec.SecretName,
				Items:      sec.Items,
			}
			spinVol.Type = "SECRET"
		}

		if ed := vol.EmptyDir; ed != nil {
			spinVol.EmptyDir = &types.EmptyDirVolumeSource{
				// Spinnaker requires this to be uppercased for some reason
				Medium: strings.ToUpper(string(ed.Medium)),
			}
			spinVol.Type = "EMPTYDIR"
		}

		vs = append(vs, spinVol)
	}

	return vs
}

func deploymentContainers(dep *appsv1.Deployment) []*types.Container {
	var c []*types.Container

	for _, container := range dep.Spec.Template.Spec.Containers {
		spinContainer := &types.Container{}

		// add the image description first off using the annotations on the container
		spinContainer.ImageDescription = types.ImageDescription{
			Account:      dep.Annotations[SpinnakerImageDescriptionAccountAnnotation],
			ImageID:      dep.Annotations[SpinnakerImageDescriptionImageIDAnnotation],
			Tag:          dep.Annotations[SpinnakerImageDescriptionTagAnnotation],
			Repository:   dep.Annotations[SpinnakerImageDescriptionRepositoryAnnotation],
			Registry:     dep.Annotations[SpinnakerImageDescriptionRegistryAnnotation],
			Organization: dep.Annotations[SpinnakerImageDescriptionOrganizationAnnotation],
		}

		args := []string{}
		if container.Args != nil {
			args = container.Args
		}

		spinContainer.Name = container.Name
		spinContainer.Args = args
		spinContainer.Command = container.Command
		spinContainer.ImagePullPolicy = strings.ToUpper(string(container.ImagePullPolicy))
		spinContainer.Requests.CPU = container.Resources.Requests.Cpu().String()
		spinContainer.Requests.Memory = container.Resources.Requests.Memory().String()
		spinContainer.Limits.CPU = container.Resources.Limits.Cpu().String()
		spinContainer.Limits.Memory = container.Resources.Limits.Memory().String()

		// appends all of the ports on the deployment type into the spinnaker definition
		for _, port := range container.Ports {
			spinContainer.Ports = append(spinContainer.Ports, types.Port{
				ContainerPort: port.ContainerPort,
				Name:          port.Name,
				Protocol:      string(port.Protocol),
			})
		}

		// appends all of the environment variables on the deployment type into the spinnaker definition
		for _, env := range container.Env {
			var e types.EnvVar
			e.Name = env.Name
			e.Value = env.Value

			if vf := env.ValueFrom; vf != nil {
				if vf.ConfigMapKeyRef != nil {
					e.EnvSource = &types.EnvSource{
						ConfigMapSource: &types.ConfigMapSource{
							ConfigMapName: vf.ConfigMapKeyRef.Name,
							Key:           vf.ConfigMapKeyRef.Key,
						},
					}
				}

				if vf.SecretKeyRef != nil {
					e.EnvSource = &types.EnvSource{
						SecretSource: &types.SecretSource{
							Key:        vf.SecretKeyRef.Key,
							SecretName: vf.SecretKeyRef.Name,
						},
					}
				}
			}

			spinContainer.EnvVars = append(spinContainer.EnvVars, e)
		}

		// add all of the volume mounts
		for _, vm := range container.VolumeMounts {
			spinContainer.VolumeMounts = append(spinContainer.VolumeMounts, types.VolumeMount{
				Name:      vm.Name,
				ReadOnly:  vm.ReadOnly,
				MountPath: vm.MountPath,
			})
		}

		c = append(c, spinContainer)
	}

	return c
}
