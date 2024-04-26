package rubix

import "time"

type UserState string

const (
	UserStateOnline  UserState = "online"
	UserStateOffline UserState = "offline"
	UserStateAway    UserState = "away"
	UserStateBusy    UserState = "busy"
)

type UserStatus struct {
	Name              string    `json:"name"`
	Icon              string    `json:"icon"`
	State             UserState `json:"state"`
	ExtendedState     string    `json:"extendedState"`
	ExpiryTime        time.Time `json:"expiryTime,omitempty"`
	ClearAfterSeconds int32     `json:"clearAfterSeconds,omitempty"`
	ClearEndOfDay     bool      `json:"clearEndOfDay,omitempty"`
	ClearOnLogout     bool      `json:"clearOnLogout,omitempty"`
}
