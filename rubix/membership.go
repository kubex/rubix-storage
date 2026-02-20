package rubix

import (
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type MembershipType string

func (mt MembershipType) Display() string {
	if mt == "" {
		return "Unknown"
	}
	return cases.Title(language.English, cases.Compact).String(string(mt))
}

// Int is used to compare your level with another users level
func (mt MembershipType) Int() int {
	switch mt {
	case MembershipTypeOwner:
		return 30
	case MembershipTypeMember:
		return 20
	case MembershipTypeSupport:
		return 10
	default:
		return 0
	}
}

const (
	MembershipTypeOwner   MembershipType = "owner"   // Full access
	MembershipTypeMember  MembershipType = "member"  // Permissions only
	MembershipTypeSupport MembershipType = "support" // Support agent
)

type MembershipState int

func (ms MembershipState) Display() string {
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
	case MembershipStateRejected:
		return "Rejected"
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
	MembershipStateRejected  MembershipState = 5
)

type MembershipSource string

const (
	MembershipSourceAdmin       MembershipSource = "admin"
	MembershipSourceOIDC        MembershipSource = "oidc"
	MembershipSourceSCIM        MembershipSource = "scim"
	MembershipSourceSelfRequest MembershipSource = "self_request"
)

type Membership struct {
	UserID     string
	Name       string
	Email      string
	Workspace  string
	Type       MembershipType
	PartnerID  string
	Since      time.Time
	State      MembershipState
	StateSince time.Time
	Source     MembershipSource
}
