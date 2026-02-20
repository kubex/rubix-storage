package rubix

import "time"

// ResolvedMember combines membership data with user identity from the correct source.
type ResolvedMember struct {
	Membership
	Source      string // "native" or "oidc"
	ProviderID  string // OIDC provider UUID, empty for native
	SCIMManaged bool
	AutoCreated bool
	LastSync    time.Time
}

// MemberFilter controls which members are returned by GetResolvedMembers.
type MemberFilter struct {
	Source       string   // "", "native", "oidc"
	ProviderUUID string   // filter to specific OIDC provider
	UserIDs      []string // specific user IDs
}
