package rubix

import "time"

// ActivationState represents a completed activation step for an app on a workspace.
// For workspace-scoped steps, UserID is empty.
// For user-scoped steps, UserID is the completing user.
type ActivationState struct {
	Workspace   string    `json:"workspace"`
	UserID      string    `json:"userID"`
	VendorID    string    `json:"vendorID"`
	AppID       string    `json:"appID"`
	StepID      string    `json:"stepID"`
	CompletedAt time.Time `json:"completedAt"`
}
