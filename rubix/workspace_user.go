package rubix

import "time"

type WorkspaceUser struct {
	UserID       string    `json:"userID"`
	Workspace    string    `json:"workspace"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	OIDCProvider string    `json:"oidcProvider"`
	SCIMManaged  bool      `json:"scimManaged"`
	AutoCreated  bool      `json:"autoCreated"`
	LastSyncTime time.Time `json:"lastSyncTime"`
	CreatedAt    time.Time `json:"createdAt"`
}

type MutateWorkspaceUserPayload struct {
	Name         *string
	Email        *string
	SCIMManaged  *bool
	AutoCreated  *bool
	LastSyncTime *time.Time
}

type MutateWorkspaceUserOption func(*MutateWorkspaceUserPayload)

func WithWorkspaceUserName(name string) MutateWorkspaceUserOption {
	return func(p *MutateWorkspaceUserPayload) { p.Name = &name }
}

func WithWorkspaceUserEmail(email string) MutateWorkspaceUserOption {
	return func(p *MutateWorkspaceUserPayload) { p.Email = &email }
}

func WithWorkspaceUserSCIMManaged(managed bool) MutateWorkspaceUserOption {
	return func(p *MutateWorkspaceUserPayload) { p.SCIMManaged = &managed }
}

func WithWorkspaceUserAutoCreated(autoCreated bool) MutateWorkspaceUserOption {
	return func(p *MutateWorkspaceUserPayload) { p.AutoCreated = &autoCreated }
}

func WithWorkspaceUserLastSyncTime(t time.Time) MutateWorkspaceUserOption {
	return func(p *MutateWorkspaceUserPayload) { p.LastSyncTime = &t }
}
