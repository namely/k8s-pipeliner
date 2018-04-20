package types

// Trigger is an interface to encompass multiple types of Spinnaker triggers
type Trigger interface {
	spinnakerTrigger()
}

// TriggerObject contains the fields that all triggers must have
type TriggerObject struct {
	Enabled bool   `json:"enabled"`
	Type    string `json:"type"`
}

// StageMetadata is the common components of a stage in spinnaker such as name
type StageMetadata struct {
	RefID                string         `json:"refId,omitempty"`
	RequisiteStageRefIds []string       `json:"requisiteStageRefIds"`
	Name                 string         `json:"name"`
	Type                 string         `json:"type"`
	Notifications        []Notification `json:"notifications,omitempty"`
	SendNotifications    bool           `json:"sendNotifications"`
}

// JenkinsTrigger constructs the JSON necessary to include a Jenkins trigger
// for a spinnaker pipeline
type JenkinsTrigger struct {
	TriggerObject

	Job          string `json:"job"`
	Master       string `json:"master"`
	PropertyFile string `json:"propertyFile"`
}

var _ Trigger = (*JenkinsTrigger)(nil)

// Trigger implements Trigger
func (t *JenkinsTrigger) spinnakerTrigger() {}

// WebhookTrigger constructs the JSON for a webhook trigger in Spinnaker
// pipelines
type WebhookTrigger struct {
	TriggerObject
	Source string `json:"source"`
}

var _ Trigger = (*WebhookTrigger)(nil)

func (t *WebhookTrigger) spinnakerTrigger() {}
