package rubix

import (
	"errors"
	"time"
)

var ErrGatewayCredentialInvalid = errors.New("invalid, expired, or revoked gateway credential")
var ErrGatewayCredentialInput = errors.New("invalid gateway credential details")

// GatewayCredential is safe to list: neither the token nor its digest is exposed.
// Credentials identify an app across workspaces; installation and grants are
// checked independently by Rubix on every request.
type GatewayCredential struct {
	ID         string    `json:"id"`
	VendorID   string    `json:"vendorID"`
	AppID      string    `json:"appID"`
	Name       string    `json:"name"`
	Prefix     string    `json:"prefix"`
	CreatedBy  string    `json:"createdBy"`
	CreatedAt  time.Time `json:"createdAt"`
	ExpiresAt  time.Time `json:"expiresAt,omitzero"`
	RevokedAt  time.Time `json:"revokedAt,omitzero"`
	ReplacedBy string    `json:"replacedBy,omitempty"`
}

func (c GatewayCredential) Active() bool {
	return c.RevokedAt.IsZero() && (c.ExpiresAt.IsZero() || time.Now().Before(c.ExpiresAt))
}

// IssuedGatewayCredential contains the raw token only in the creation/rotation
// response. Never persist or log this value.
type IssuedGatewayCredential struct {
	Credential GatewayCredential `json:"credential"`
	Token      string            `json:"token"`
}
