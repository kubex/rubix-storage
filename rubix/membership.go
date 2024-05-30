package rubix

import (
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type MembershipType string

func (mt MembershipType) String() string {
	if mt == "" {
		return "Unknown"
	}
	return cases.Title(language.English, cases.Compact).String(string(mt))
}

const (
	MembershipTypeOwner   MembershipType = "owner"   // Full access
	MembershipTypeMember  MembershipType = "member"  // Permissions only
	MembershipTypeSupport MembershipType = "support" // Support agent
)

type MembershipState int

func (ms MembershipState) String() string {
	switch ms {
	case MembershipStatePending:
		return "Pending"
	case MembershipStateActive:
		return "Active"
	case MembershipStateSuspended:
		return "Suspended"
	case MembershipStateArchived:
		return "Archived"
	case MembershipStateRemoved:
		return "Removed"
	default:
		return "Unknown"
	}
}

const (
	MembershipStatePending   MembershipState = 0
	MembershipStateActive    MembershipState = 1
	MembershipStateSuspended MembershipState = 2
	MembershipStateArchived  MembershipState = 3
	MembershipStateRemoved   MembershipState = 4
)

type Membership struct {
	UserID     string
	Workspace  string
	Type       MembershipType
	PartnerID  string
	Since      time.Time
	State      MembershipState
	StateSince time.Time
}
