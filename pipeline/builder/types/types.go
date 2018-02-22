package types

import (
	corev1 "k8s.io/api/core/v1"
)

// SpinnakerPipeline defines the fields for the top leve object of a spinnaker
// pipeline. Mostly used for constructing JSON
type SpinnakerPipeline struct {
	Triggers []Trigger `json:"triggers"`
	Stages   []Stage   `json:"stages"`
}

// Trigger is an interface to encompass multiple types of Spinnaker triggers
type Trigger interface {
	spinnakerTrigger()
}

// StageMetadata is the common components of a stage in spinnaker such as name
type StageMetadata struct {
	RefID                string         `json:"refId"`
	RequisiteStageRefIds []string       `json:"requisiteStageRefIds,omitempty"`
	Name                 string         `json:"name"`
	Type                 string         `json:"type"`
	Notifications        []Notification `json:"notifications,omitempty"`
	SendNotifications    bool           `json:"sendNotifications"`
}

// JenkinsTrigger constructs the JSON necessary to include a Jenkins trigger
// for a spinnaker pipeline
type JenkinsTrigger struct {
	Enabled      bool   `json:"enabled"`
	Job          string `json:"job"`
	Master       string `json:"master"`
	PropertyFile string `json:"propertyFile"`
	Type         string `json:"type"`
}

var _ Trigger = &JenkinsTrigger{}

// Trigger implements Trigger
func (t *JenkinsTrigger) spinnakerTrigger() {}

// Stage is an interface to represent a Stage struct such as RunJob or Deploy
type Stage interface {
	spinnakerStage()
}

// RunJobStage is used to represent a job that needs to be ran in a pipeline
// configuration
type RunJobStage struct {
	StageMetadata

	Account           string            `json:"account"`
	Annotations       map[string]string `json:"annotations"`
	Application       string            `json:"application"`
	CloudProvider     string            `json:"cloudProvider"`
	CloudProviderType string            `json:"cloudProviderType"`
	Container         *Container        `json:"container"`
	DNSPolicy         string            `json:"dnsPolicy"`
	Labels            map[string]string `json:"labels"`
	Namespace         string            `json:"namespace"`
	VolumeSources     []interface{}     `json:"volumeSources"`
}

func (rjs RunJobStage) spinnakerStage() {}

var _ Stage = RunJobStage{}

// DeployStage handles the creation of a server cluster in a pipeline
type DeployStage struct {
	StageMetadata

	Clusters []Cluster `json:"clusters"`
}

func (ds DeployStage) spinnakerStage() {}

var _ Stage = DeployStage{}

// ManualJudgementStage handles the manual judgement json in a pipeline
type ManualJudgementStage struct {
	StageMetadata

	FailPipeline bool     `json:"failPipeline"`
	Instructions string   `json:"instructions"`
	Inputs       []string `json:"inputs,omitempty"`
}

func (mjs ManualJudgementStage) spinnakerStage() {}

var _ Stage = ManualJudgementStage{}

// Cluster defines a server group to be deployed within a Deploy stage of a
// pipeline
type Cluster struct {
	Account                        string            `json:"account"`
	Application                    string            `json:"application"`
	CloudProvider                  string            `json:"cloudProvider"`
	Containers                     []*Container      `json:"containers"`
	DNSPolicy                      string            `json:"dnsPolicy"`
	Events                         []interface{}     `json:"events"`
	InterestingHealthProviderNames []string          `json:"interestingHealthProviderNames"`
	LoadBalancers                  []string          `json:"loadBalancers"`
	MaxRemainingAsgs               int               `json:"maxRemainingAsgs"`
	NodeSelector                   map[string]string `json:"nodeSelector"`
	PodAnnotations                 map[string]string `json:"podAnnotations"`
	Provider                       string            `json:"provider"`

	// Region is just a kubernetes namespace
	Region    string `json:"region"`
	Namespace string `json:"namespace"`

	ReplicaSetAnnotations         map[string]string `json:"replicaSetAnnotations"`
	ScaleDown                     bool              `json:"scaleDown"`
	SecurityGroups                []interface{}     `json:"securityGroups"`
	Stack                         string            `json:"stack"`
	Details                       string            `json:"freeFormDetails"`
	Strategy                      string            `json:"strategy"`
	TargetSize                    int               `json:"targetSize"`
	TerminationGracePeriodSeconds int               `json:"terminationGracePeriodSeconds"`
	VolumeSources                 []*VolumeSource   `json:"volumeSources"`
	DelayBeforeDisableSec         int               `json:"delayBeforeDisableSec,omitempty"`
}

// Container is a representation of a container to be deployed either as a job
// or within a cluster
type Container struct {
	Args             []string         `json:"args"`
	Command          []string         `json:"command"`
	EnvVars          []EnvVar         `json:"envVars"`
	EnvFrom          []EnvFromSource  `json:"envFrom"`
	ImageDescription ImageDescription `json:"imageDescription"`
	ImagePullPolicy  string           `json:"imagePullPolicy"`
	Limits           Resources        `json:"limits"`
	Requests         Resources        `json:"requests"`

	Name  string `json:"name"`
	Ports []Port `json:"ports"`

	VolumeMounts []VolumeMount `json:"volumeMounts"`
}

// EnvFromSource is used to pull in a config map as a list of environment
// variables
type EnvFromSource struct {
	Prefix          string                  `json:"prefix"`
	ConfigMapSource *EnvFromConfigMapSource `json:"configMapRef"`
}

// EnvFromConfigMapSource is used to pull in a configmap for key/value envVars
type EnvFromConfigMapSource struct {
	Name     string `json:"name"`
	Optional bool   `json:"optional"`
}

// VolumeMount describes a mount that should be mounted in to the container
// by referencing a volume source in the pod spec
type VolumeMount struct {
	MountPath string `json:"mountPath"`
	Name      string `json:"name"`
	ReadOnly  bool   `json:"readOnly"`
}

// Resources for the container either as a limit or request
type Resources struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

// Port is a Container port to expose
type Port struct {
	ContainerPort int32  `json:"containerPort"`
	Name          string `json:"name"`
	Protocol      string `json:"protocol"`
}

// EnvVar represents an environment variable that is injected into the container
// when it starts. It can contain many sources
type EnvVar struct {
	EnvSource *EnvSource `json:"envSource,omitempty"`
	Name      string     `json:"name"`
	Value     string     `json:"value,omitempty"`
}

// EnvSource encapsulates the different possible places an environment var
// can come from (secrets or configmaps)
type EnvSource struct {
	SecretSource    *SecretSource    `json:"secretSource,omitempty"`
	ConfigMapSource *ConfigMapSource `json:"configMapSource,omitempty"`
}

// SecretSource is a env var from a secret map in k8s
type SecretSource struct {
	SecretName string `json:"secretName"`
	Key        string `json:"key"`
}

// ConfigMapSource is a env var from a config map in k8s
type ConfigMapSource struct {
	ConfigMapName string `json:"configMapName"`
	Key           string `json:"key"`
}

// ImageDescription is used to tell spinnaker which image to use for a stage
// of a pipeline in a cluster definition or job
type ImageDescription struct {
	Account      string `json:"account"`
	FromTrigger  bool   `json:"fromTrigger"`
	ImageID      string `json:"imageId"`
	Registry     string `json:"registry"`
	Repository   string `json:"repository"`
	Tag          string `json:"tag"`
	Organization string `json:"organization"`
}

// Notification is a struct for defining a notification for a stage or pipeline
type Notification struct {
	Address string                         `json:"address"`
	Level   string                         `json:"level"`
	Type    string                         `json:"type"`
	When    []string                       `json:"when"`
	Message map[string]NotificationMessage `json:"message"`
}

// NotificationMessage is for providing text to a stage failure type
type NotificationMessage struct {
	Text string `json:"text"`
}

// VolumeSource defines a pod volume source that can be referenced by containers
type VolumeSource struct {
	Name string `json:"name"`
	Type string `json:"type"`

	EmptyDir  *EmptyDirVolumeSource  `json:"emptyDir,omitempty"`
	ConfigMap *ConfigMapVolumeSource `json:"configMap,omitempty"`
	Secret    *SecretVolumeSource    `json:"secret,omitempty"`
}

// EmptyDirVolumeSource defines a empty directory volume source for a pod:
// https://kubernetes.io/docs/api-reference/v1.9/#emptydirvolumesource-v1-core
type EmptyDirVolumeSource struct {
	Medium string `json:"medium"`
}

// ConfigMapVolumeSource type for referencing configmaps in volumes
type ConfigMapVolumeSource struct {
	ConfigMapName string             `json:"configMapName"`
	DefaultMode   *int32             `json:"defaultMode,omitempty"`
	Items         []corev1.KeyToPath `json:"items"`
}

// SecretVolumeSource for referencing secret types in volumes
type SecretVolumeSource struct {
	SecretName string             `json:"secretName"`
	Items      []corev1.KeyToPath `json:"items"`
}
