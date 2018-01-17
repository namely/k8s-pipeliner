package spinnaker

// Stage is an interface to represent a Stage struct such as RunJob or Deploy
type Stage interface {
	spinnakerStage()
}

// Pipeline is a generated structure from JSON that spinnaker
// accepts for creating / modifying a pipeline
type Pipeline struct {
	AppConfig struct {
	} `json:"appConfig"`
	KeepWaitingPipelines bool    `json:"keepWaitingPipelines"`
	LastModifiedBy       string  `json:"lastModifiedBy"`
	LimitConcurrent      bool    `json:"limitConcurrent"`
	Stages               []Stage `json:"stages"`
	Triggers             []struct {
		Account      string `json:"account"`
		Enabled      bool   `json:"enabled"`
		Organization string `json:"organization"`
		Registry     string `json:"registry"`
		Repository   string `json:"repository"`
		Tag          string `json:"tag"`
		Type         string `json:"type"`
	} `json:"triggers"`
	UpdateTs string `json:"updateTs"`
}

// RunJobStage is used to represent a job that needs to be ran in a pipeline
// configuration
type RunJobStage struct {
	Account     string `json:"account"`
	Annotations struct {
	} `json:"annotations"`
	Application       string `json:"application"`
	CloudProvider     string `json:"cloudProvider"`
	CloudProviderType string `json:"cloudProviderType"`
	Container         struct {
		Args    []string `json:"args"`
		Command []string `json:"command"`
		EnvVars []struct {
			Name      string `json:"name"`
			Value     string `json:"value,omitempty"`
			EnvSource struct {
				ConfigMapSource struct {
					ConfigMapName string `json:"configMapName"`
					Key           string `json:"key"`
				} `json:"configMapSource"`
			} `json:"envSource,omitempty"`
		} `json:"envVars"`
		ImageDescription struct {
			Account      string `json:"account"`
			FromTrigger  bool   `json:"fromTrigger"`
			Organization string `json:"organization"`
			Registry     string `json:"registry"`
			Repository   string `json:"repository"`
			Tag          string `json:"tag"`
		} `json:"imageDescription"`
		Name         string        `json:"name"`
		VolumeMounts []interface{} `json:"volumeMounts"`
	} `json:"container"`
	DNSPolicy string `json:"dnsPolicy"`
	Labels    struct {
	} `json:"labels"`
	Name                 string        `json:"name"`
	Namespace            string        `json:"namespace"`
	RefID                string        `json:"refId"`
	RequisiteStageRefIds []interface{} `json:"requisiteStageRefIds"`
	Type                 string        `json:"type"`
	VolumeSources        []interface{} `json:"volumeSources"`
}

func (rjs RunJobStage) spinnakerStage() {}

var _ Stage = RunJobStage{}

// DeployStage handles the creation of a server cluster in a pipeline
type DeployStage struct {
	Clusters             []Cluster `json:"clusters"`
	Name                 string    `json:"name"`
	RefID                string    `json:"refId,omitempty"`
	RequisiteStageRefIds []string  `json:"requisiteStageRefIds,omitempty"`
	Type                 string    `json:"type"`
}

func (ds DeployStage) spinnakerStage() {}

var _ Stage = DeployStage{}

type Cluster struct {
	Account                        string                 `json:"account"`
	Application                    string                 `json:"application"`
	CloudProvider                  string                 `json:"cloudProvider"`
	Containers                     []DeployStageContainer `json:"containers"`
	DNSPolicy                      string                 `json:"dnsPolicy"`
	Events                         []interface{}          `json:"events"`
	FreeFormDetails                string                 `json:"freeFormDetails"`
	InterestingHealthProviderNames []string               `json:"interestingHealthProviderNames"`
	LoadBalancers                  []string               `json:"loadBalancers"`
	MaxRemainingAsgs               int                    `json:"maxRemainingAsgs"`
	Namespace                      string                 `json:"namespace"`
	NodeSelector                   struct{}               `json:"nodeSelector"`
	PodAnnotations                 struct{}               `json:"podAnnotations"`
	Provider                       string                 `json:"provider"`
	Region                         string                 `json:"region"`
	ReplicaSetAnnotations          struct{}               `json:"replicaSetAnnotations"`
	ScaleDown                      bool                   `json:"scaleDown"`
	SecurityGroups                 []interface{}          `json:"securityGroups"`
	Stack                          string                 `json:"stack"`
	Strategy                       string                 `json:"strategy"`
	TargetSize                     int                    `json:"targetSize"`
	TerminationGracePeriodSeconds  int                    `json:"terminationGracePeriodSeconds"`
	VolumeSources                  []interface{}          `json:"volumeSources"`
	DelayBeforeDisableSec          int                    `json:"delayBeforeDisableSec,omitempty"`
}

// ManualJudgementStage handles the creation of a manual judgement
type ManualJudgementStage struct {
	FailPipeline         bool          `json:"failPipeline"`
	Instructions         string        `json:"instructions"`
	JudgmentInputs       []interface{} `json:"judgmentInputs"`
	Name                 string        `json:"name"`
	Notifications        []interface{} `json:"notifications"`
	RefID                string        `json:"refId,omitempty"`
	RequisiteStageRefIds []string      `json:"requisiteStageRefIds,omitempty"`
	Type                 string        `json:"type"`
}

func (mjs ManualJudgementStage) spinnakerStage() {}

var _ Stage = ManualJudgementStage{}

type DeployStageContainer struct {
	Args             []string         `json:"args"`
	Command          []string         `json:"command"`
	EnvVars          []EnvVar         `json:"envVars"`
	ImageDescription ImageDescription `json:"imageDescription"`
	ImagePullPolicy  string           `json:"imagePullPolicy"`
	Limits           struct {
		CPU string `json:"cpu"`
	} `json:"limits"`
	Name  string `json:"name"`
	Ports []struct {
		ContainerPort int32  `json:"containerPort"`
		Name          string `json:"name"`
		Protocol      string `json:"protocol"`
	} `json:"ports"`
	Requests struct {
		CPU string `json:"cpu"`
	} `json:"requests"`
	VolumeMounts []interface{} `json:"volumeMounts"`
}

type EnvVar struct {
	EnvSource *EnvSource `json:"envSource,omitempty"`
	Name      string     `json:"name"`
	Value     string     `json:"value,omitempty"`
}

type EnvSource struct {
	SecretSource    *SecretSource    `json:"secretSource,omitempty"`
	ConfigMapSource *ConfigMapSource `json:"configMapSource,omitempty"`
}

type SecretSource struct {
	SecretName string `json:"secretName"`
	Key        string `json:"key"`
}

type ConfigMapSource struct {
	ConfigMapName string `json:"configMapName"`
	Key           string `json:"key"`
}

type ImageDescription struct {
	Account     string `json:"account"`
	FromTrigger bool   `json:"fromTrigger"`
	ImageID     string `json:"imageId"`
	Registry    string `json:"registry"`
	Repository  string `json:"repository"`
	Tag         string `json:"tag"`
}
