package rubix

import "time"

// VendorMember is a user's membership of a platform vendor.
type VendorMember struct {
	VendorID string    `json:"vendorID"`
	UserID   string    `json:"userID"`
	IsAdmin  bool      `json:"isAdmin"`
	IsOwner  bool      `json:"isOwner"` // owners are always admins
	AddedBy  string    `json:"addedBy"`
	Since    time.Time `json:"since"`
	Name     string    `json:"name"`  // read-only, from the users table; empty when the user row is missing
	Email    string    `json:"email"` // read-only, from the users table
}
