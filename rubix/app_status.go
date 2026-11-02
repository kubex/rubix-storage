package rubix

import "strings"

// AppStatus is the lifecycle status of a platform application.
type AppStatus string

const (
	AppStatusActive     AppStatus = "active"
	AppStatusPreRelease AppStatus = "pre-release"
	AppStatusSuspended  AppStatus = "suspended"
	AppStatusArchived   AppStatus = "archived"
)

// Blocked reports whether the app is unavailable in every workspace.
func (s AppStatus) Blocked() bool {
	return s == AppStatusSuspended || s == AppStatusArchived
}

// ParseAppStatus maps a stored or submitted value to an AppStatus; "" is active.
// ok is false for unknown values.
func ParseAppStatus(value string) (AppStatus, bool) {
	switch AppStatus(strings.ToLower(strings.TrimSpace(value))) {
	case "", AppStatusActive:
		return AppStatusActive, true
	case AppStatusPreRelease:
		return AppStatusPreRelease, true
	case AppStatusSuspended:
		return AppStatusSuspended, true
	case AppStatusArchived:
		return AppStatusArchived, true
	}
	return AppStatusActive, false
}
