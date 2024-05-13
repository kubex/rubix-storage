package rubix

import "time"

type UserState string

const (
	UserStateOnline  UserState = "online"
	UserStateOffline UserState = "offline"
	UserStateAway    UserState = "away"
	UserStateBusy    UserState = "busy"
	UserStateHiatus  UserState = "hiatus"
)

type UserStatus struct {
	Name              string    `json:"name"`
	Icon              string    `json:"icon"`
	State             UserState `json:"state"`
	ExtendedState     string    `json:"extendedState"`
	AppliedTime       time.Time `json:"appliedTime,omitempty"`
	ExpiryTime        time.Time `json:"expiryTime,omitempty"`
	ClearAfterSeconds int32     `json:"clearAfterSeconds,omitempty"`
	ClearEndOfDay     bool      `json:"clearEndOfDay,omitempty"`
	ClearOnLogout     bool      `json:"clearOnLogout,omitempty"`
	ID                string    `json:"id,omitempty"`
	AfterID           string    `json:"afterId,omitempty"`

	Overlays []UserStatus `json:"overlays,omitempty"`
}
