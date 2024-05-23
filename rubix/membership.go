package rubix

import "time"

type MembershipType string

const (
	MembershipTypeOwner   MembershipType = "owner"   // Full access
	MembershipTypeMember  MembershipType = "member"  // Permissions only
	MembershipTypeSupport MembershipType = "support" // Support agent
)

type MembershipState int

const (
	MembershipStatePending   MembershipState = 0
	MembershipStateActive    MembershipState = 1
	MembershipStateSuspended MembershipState = 2
	MembershipStateArchived  MembershipState = 3
	MembershipStateRemoved   MembershipState = 4
)

type Membership struct {
	User      string
	Workspace string
	Type      MembershipType
	PartnerID string
	Since     time.Time
	State     MembershipState
	StteSince time.Time
}
