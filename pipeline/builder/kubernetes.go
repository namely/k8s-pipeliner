package builder

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/namely/k8s-pipeliner/pipeline/builder/types"
	"github.com/namely/k8s-pipeliner/pipeline/config"
	"github.com/pkg/errors"

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

// ManifestGroup keeps a collection of containers from a deployment
// and metadata associated with them
type ManifestGroup struct {
	Namespace      string
	Annotations    map[string]string
	PodAnnotations map[string]string
	Containers     []*types.Container
	InitContainers []*types.Container
	VolumeSources  []*types.VolumeSource
}

// ManifestParser handles generating Spinnaker builder types from a kubernetes
// manifest file (deployments)
type ManifestParser struct {
	config   *config.Pipeline
	basePath string
}

// NewManfifestParser initializes and returns a manifest parser for a given pipeline config.
// If a basePath is passed it is used as the path when loading relative file paths
// for manifest definitions
func NewManfifestParser(config *config.Pipeline, basePath ...string) *ManifestParser {
	mp := &ManifestParser{config: config}
	if len(basePath) > 0 {
		mp.basePath = basePath[0]
	}

	return mp
}

func (mp *ManifestParser) ManifestFromScaffold(scaffold config.ContainerScaffold) (runtime.Object, error) {
	path := scaffold.Manifest()
	if !filepath.IsAbs(scaffold.Manifest()) && mp.basePath != "" {
		path = filepath.Join(mp.basePath, scaffold.Manifest())
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode(b, nil, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "marshaling failure: %s", path)
	}

	switch t := obj.(type) {
	case *appsv1.Deployment:
		r, err := mp.InjectDeploymentOverrides(t, scaffold)
		if err != nil {
			return nil, errors.Wrapf(err, "error injecting overrides into deployment manifests")
		}
		return r, nil
	case *corev1.Pod:
		r, err := mp.InjectPodOverrides(t, scaffold)
		if err != nil {
			return nil, errors.Wrapf(err, "error injecting overrides into pod manifests")
		}
		return r, nil
	}

	return obj, nil
}

// InjectDeploymentOverrides takes the manifest ->  injects them into the marshalled manifest
func (mp *ManifestParser) InjectDeploymentOverrides(manifest *appsv1.Deployment, scaffold config.ContainerScaffold) (*appsv1.Deployment, error) {
	replicas := int32(scaffold.GetTargetSize())
	manifest.Spec.Replicas = &replicas

	for pos, container := range manifest.Spec.Template.Spec.Containers {
		manifest.Spec.Template.Spec.Containers[pos] = mp.InjectContainerImageDescription(container, scaffold)
	}
	for pos, container := range manifest.Spec.Template.Spec.InitContainers {
		manifest.Spec.Template.Spec.InitContainers[pos] = mp.InjectContainerImageDescription(container, scaffold)
	}
	return manifest, nil
}

func (mp *ManifestParser) InjectPodOverrides(manifest *corev1.Pod, scaffold config.ContainerScaffold) (*corev1.Pod, error) {
	for pos, container := range manifest.Spec.Containers {
		manifest.Spec.Containers[pos] = mp.InjectContainerImageDescription(container, scaffold)
	}
	for pos, container := range manifest.Spec.InitContainers {
		manifest.Spec.InitContainers[pos] = mp.InjectContainerImageDescription(container, scaffold)
	}
	return manifest, nil
}

func (mp *ManifestParser) InjectContainerImageDescription(container corev1.Container, scaffold config.ContainerScaffold) corev1.Container {
	if ref := scaffold.ImageDescriptionRef(container.Name); ref != nil {
		for _, desc := range mp.config.ImageDescriptions {
			if desc.Name == ref.Name && ref.ContainerName == container.Name {
				container.Image = desc.ImageID
			}
		}
	}
	return container
}

// ContainersFromScaffold loads a kubernetes manifest file and generates
// spinnaker pipeline containers config from it.
func (mp *ManifestParser) ContainersFromScaffold(scaffold config.ContainerScaffold) (*ManifestGroup, error) {

	path := scaffold.Manifest()
	if !filepath.IsAbs(scaffold.Manifest()) && mp.basePath != "" {
		path = filepath.Join(mp.basePath, scaffold.Manifest())
	}

	f, err := os.Open(path)
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
		return nil, errors.Wrapf(err, "marshaling failure: %s", path)
	}

	var mg ManifestGroup
	var resource runtime.Object

	switch g.Kind {
	case "Deployment":
		resource = &appsv1.Deployment{}
		if err := scheme.Scheme.Convert(obj, resource, nil); err != nil {
			return nil, err
		}
	case "Pod":
		resource = obj
	}

	switch t := resource.(type) {
	case *appsv1.Deployment:
		mg.Containers = mp.deploymentContainers(t.Spec.Template.Spec, scaffold)
		mg.InitContainers = mp.deploymentInitContainers(t.Spec.Template.Spec, scaffold)
		mg.Annotations = t.Annotations
		mg.PodAnnotations = t.Spec.Template.Annotations
		mg.Namespace = t.GetNamespace()
		mg.VolumeSources = mp.volumeSources(t.Spec.Template.Spec.Volumes)
	case *corev1.Pod:
		mg.Containers = mp.deploymentContainers(t.Spec, scaffold)
		mg.InitContainers = mp.deploymentInitContainers(t.Spec, scaffold)
		mg.Annotations = t.Annotations
		mg.PodAnnotations = t.Annotations
		mg.Namespace = t.GetNamespace()
		mg.VolumeSources = mp.volumeSources(t.Spec.Volumes)
	default:
		return nil, fmt.Errorf("type not supported: %T", t)
	}

	if mg.PodAnnotations == nil {
		mg.PodAnnotations = make(map[string]string)
	}

	return &mg, nil
}

// converts kubernetes volume sources into builder types
func (mp *ManifestParser) volumeSources(vols []corev1.Volume) []*types.VolumeSource {
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

		if ed := vol.PersistentVolumeClaim; ed != nil {
			spinVol.PersistentVolumeClaim = &types.PersistentVolumeClaimVolumeSource{
				ClaimName: ed.ClaimName,
			}
			spinVol.Type = "PERSISTENTVOLUMECLAIM"
		}

		if ed := vol.HostPath; ed != nil {
			spinVol.HostPath = &types.HostPathVolumeSource{
				Path: ed.Path,
			}
			spinVol.Type = "HOSTPATH"
		}

		vs = append(vs, spinVol)
	}

	return vs
}

func (mp *ManifestParser) deploymentContainers(podspec corev1.PodSpec, scaffold config.ContainerScaffold) []*types.Container {
	var c []*types.Container

	for _, container := range podspec.Containers {
		c = append(c, mp.parseContainer(container, scaffold))
	}

	return c
}

func (mp *ManifestParser) deploymentInitContainers(podspec corev1.PodSpec, scaffold config.ContainerScaffold) []*types.Container {
	var c []*types.Container

	for _, container := range podspec.InitContainers {
		c = append(c, mp.parseContainer(container, scaffold))
	}

	return c
}

func (mp *ManifestParser) parseContainer(container corev1.Container, scaffold config.ContainerScaffold) *types.Container {
	spinContainer := &types.Container{}

	// add the image description first off using the annotations on the container
	var imageDescription config.ImageDescription

	if ref := scaffold.ImageDescriptionRef(container.Name); ref != nil {
		for _, desc := range mp.config.ImageDescriptions {
			if desc.Name == ref.Name && ref.ContainerName == container.Name {
				imageDescription = desc
				break
			}
		}
	}

	spinContainer.ImageDescription = types.ImageDescription{
		Account:      imageDescription.Account,
		ImageID:      imageDescription.ImageID,
		Tag:          imageDescription.Tag,
		Repository:   imageDescription.Repository,
		Registry:     imageDescription.Registry,
		Organization: imageDescription.Organization,
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
				if vf.ConfigMapKeyRef.Optional == nil {
					vf.ConfigMapKeyRef.Optional = newFalse()
				}

				e.EnvSource = &types.EnvSource{
					ConfigMapSource: &types.ConfigMapSource{
						ConfigMapName: vf.ConfigMapKeyRef.Name,
						Key:           vf.ConfigMapKeyRef.Key,
						Optional:      *vf.ConfigMapKeyRef.Optional,
					},
				}
			}

			if vf.SecretKeyRef != nil {
				if vf.SecretKeyRef.Optional == nil {
					vf.SecretKeyRef.Optional = newFalse()
				}
				e.EnvSource = &types.EnvSource{
					SecretSource: &types.SecretSource{
						Key:        vf.SecretKeyRef.Key,
						SecretName: vf.SecretKeyRef.Name,
						Optional:   *vf.SecretKeyRef.Optional,
					},
				}
			}

			if vf.FieldRef != nil {
				e.EnvSource = &types.EnvSource{
					FieldRefSource: &types.FieldRefSource{
						FieldPath: vf.FieldRef.FieldPath,
					},
				}
			}
		}

		spinContainer.EnvVars = append(spinContainer.EnvVars, e)
	}

	for _, envFrom := range container.EnvFrom {
		var e types.EnvFromSource
		e.Prefix = envFrom.Prefix

		if cmRef := envFrom.ConfigMapRef; cmRef != nil {
			e.ConfigMapSource = &types.EnvFromConfigMapSource{
				Name: cmRef.Name,
			}
		}

		if secRef := envFrom.SecretRef; secRef != nil {
			e.SecretSource = &types.EnvFromSecretSource{
				Name: secRef.Name,
			}
		}

		spinContainer.EnvFrom = append(spinContainer.EnvFrom, e)
	}

	if probe := container.LivenessProbe; probe != nil {
		spinContainer.LivenessProbe = spinnakerProbeHandler(probe)
	}

	if probe := container.ReadinessProbe; probe != nil {
		spinContainer.ReadinessProbe = spinnakerProbeHandler(probe)
	}

	// add all of the volume mounts
	for _, vm := range container.VolumeMounts {
		spinContainer.VolumeMounts = append(spinContainer.VolumeMounts, types.VolumeMount{
			Name:      vm.Name,
			ReadOnly:  vm.ReadOnly,
			MountPath: vm.MountPath,
		})
	}

	// append security context
	if sc := container.SecurityContext; sc != nil {
		ssc := &types.SecurityContext{
			Privileged:             sc.Privileged,
			ReadOnlyRootFileSystem: sc.ReadOnlyRootFilesystem,
			RunAsUser:              sc.RunAsUser,
		}

		if caps := sc.Capabilities; caps != nil {
			ssc.Capabilities = &types.SecurityContextCapabilities{}

			for _, add := range caps.Add {
				ssc.Capabilities.Add = append(ssc.Capabilities.Add, string(add))
			}

			for _, drop := range caps.Drop {
				ssc.Capabilities.Drop = append(ssc.Capabilities.Drop, string(drop))
			}
		}

		spinContainer.SecurityContext = ssc
	}

	return spinContainer
}

func spinnakerProbeHandler(probe *corev1.Probe) *types.Probe {
	h := types.ProbeHandler{}

	if httpGet := probe.HTTPGet; httpGet != nil {
		h.Type = "HTTP"
		h.HTTPGetAction = &types.HTTPGetAction{
			Path:      httpGet.Path,
			Port:      httpGet.Port.IntValue(),
			URIScheme: string(httpGet.Scheme),
		}

		for _, header := range httpGet.HTTPHeaders {
			h.HTTPGetAction.HTTPHeaders = append(h.HTTPGetAction.HTTPHeaders, types.HTTPGetActionHeaders{
				Name:  header.Name,
				Value: header.Value,
			})
		}
	}

	if exec := probe.Exec; exec != nil {
		h.ExecAction = &types.ExecAction{
			Commands: exec.Command,
		}
		h.Type = "EXEC"
	}

	if tcp := probe.TCPSocket; tcp != nil {
		h.TCPSocketAction = &types.TCPSocketAction{
			Port: tcp.Port.IntValue(),
		}
		h.Type = "TCP"
	}

	return &types.Probe{
		FailureThreshold:    probe.FailureThreshold,
		InitialDelaySeconds: probe.InitialDelaySeconds,
		PeriodSeconds:       probe.PeriodSeconds,
		SuccessThreshold:    probe.SuccessThreshold,
		TimeoutSeconds:      probe.TimeoutSeconds,
		Handler:             h,
	}
}

func newFalse() *bool {
	b := false
	return &b
}
