package rubix

import "time"

// VendorConnection links a workspace, or one of its users, to an account on a
// vendor's auth provider. UserID is empty for the workspace link.
//
// AccessToken and RefreshToken are stored as given: callers seal them first.
type VendorConnection struct {
	Workspace string `json:"workspace"`
	UserID    string `json:"userID"`
	VendorID  string `json:"vendorID"`
	Provider  string `json:"provider"`
	// Account tells apart the accounts a workspace links to a provider that
	// takes several; it is empty for a provider's only link.
	Account string `json:"account"`
	// Brand is the workspace brand the account is for, one account to a
	// brand. It is empty for a provider's only link.
	Brand string `json:"brand"`

	ExternalID   string `json:"externalID"` // the vendor's workspace or user ID
	ExternalName string `json:"externalName"`

	AccessToken  string    `json:"-"`
	RefreshToken string    `json:"-"`
	TokenType    string    `json:"tokenType"`
	Scopes       []string  `json:"scopes"`
	ExpiresAt    time.Time `json:"expiresAt"` // zero when the access token does not expire

	LinkedBy  string    `json:"linkedBy"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
