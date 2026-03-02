package rubix

import "time"

// Blueprint represents a cached blueprint from the registry.
type Blueprint struct {
	ID            string    `json:"id"`            // "vendor/blueprint-name"
	VendorID      string    `json:"vendorID"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Icon          string    `json:"icon"`
	LatestVersion string    `json:"latestVersion"`
	SourceURL     string    `json:"sourceURL"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// BlueprintVersion represents an immutable version of a blueprint.
type BlueprintVersion struct {
	BlueprintID string    `json:"blueprintID"`
	Version     string    `json:"version"`
	Definition  string    `json:"definition"` // JSON-encoded BlueprintDefinition
	ContentHash string    `json:"contentHash"`
	CreatedAt   time.Time `json:"createdAt"`
}

// WorkspaceBlueprint tracks a workspace's subscription to a blueprint.
type WorkspaceBlueprint struct {
	WorkspaceUUID     string    `json:"workspaceUUID"`
	BlueprintID       string    `json:"blueprintID"`
	SubscribedVersion string    `json:"subscribedVersion"`
	Status            string    `json:"status"` // "active", "update_available", "drifted"
	SubscribedAt      time.Time `json:"subscribedAt"`
}

// WorkspaceBlueprintResource tracks per-resource state for a blueprint subscription.
type WorkspaceBlueprintResource struct {
	WorkspaceUUID string    `json:"workspaceUUID"`
	BlueprintID   string    `json:"blueprintID"`
	ResourceType  string    `json:"resourceType"` // "app", "setting", "role", "integration"
	ResourceKey   string    `json:"resourceKey"`
	DesiredValue  string    `json:"desiredValue"`
	AppliedValue  string    `json:"appliedValue"`
	Status        string    `json:"status"` // "in_sync", "drifted", "dismissed", "pending"
	LastSyncedAt  time.Time `json:"lastSyncedAt"`
}
