package rubix

import "time"

// SCIMGroupMapping maps a SCIM group from an OIDC provider to a Rubix team
type SCIMGroupMapping struct {
	ProviderUUID  string `json:"providerUUID"`
	ScimGroupID   string `json:"scimGroupID"`
	ScimGroupName string `json:"scimGroupName"`
	RubixTeamID   string `json:"rubixTeamID"`
	DefaultLevel  string `json:"defaultLevel"`
}

// SCIMRoleMapping maps a SCIM attribute from an OIDC provider to a Rubix role
type SCIMRoleMapping struct {
	ProviderUUID  string `json:"providerUUID"`
	ScimAttribute string `json:"scimAttribute"`
	RubixRoleID   string `json:"rubixRoleID"`
}

// SCIMActivityLog records a SCIM provisioning operation
type SCIMActivityLog struct {
	ID           int64     `json:"id"`
	ProviderUUID string    `json:"providerUUID"`
	Workspace    string    `json:"workspace"`
	Timestamp    time.Time `json:"timestamp"`
	Operation    string    `json:"operation"`
	Resource     string    `json:"resource"`
	ResourceID   string    `json:"resourceID"`
	Status       string    `json:"status"`
	Detail       string    `json:"detail"`
}
