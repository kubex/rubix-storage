package rubix

import (
	"errors"
	"time"
)

// ErrIdentityConflict is returned when an external identity is already linked to a different user.
var ErrIdentityConflict = errors.New("identity linked to another user")

type IdentitySource string

const (
	IdentitySourceApp      IdentitySource = "app"
	IdentitySourceBackfill IdentitySource = "backfill"
	IdentitySourceAdmin    IdentitySource = "admin"
)

// UserIdentity links an ID owned by an external vendor (e.g. a Fortifi employee FID) to a Rubix user.
// Scope disambiguates IDs that are only unique within a vendor tenant, such as a Fortifi org.
type UserIdentity struct {
	Workspace  string         `json:"workspace"`
	Vendor     string         `json:"vendor"`
	Scope      string         `json:"scope"`
	ExternalID string         `json:"externalId"`
	UserID     string         `json:"userId"`
	Source     IdentitySource `json:"source"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}
