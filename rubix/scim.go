package rubix

import "time"

// SCIMActivityLog records a SCIM provisioning operation
type SCIMActivityLog struct {
	ID           string    `json:"id"`
	ProviderUUID string    `json:"providerUUID"`
	Workspace    string    `json:"workspace"`
	Timestamp    time.Time `json:"timestamp"`
	Operation    string    `json:"operation"`
	Resource     string    `json:"resource"`
	ResourceID   string    `json:"resourceID"`
	Status       string    `json:"status"`
	Detail       string    `json:"detail"`
}
