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
	Name              string
	Icon              string
	State             UserState
	ExtendedState     string
	ExpiryTime        time.Time
	ClearAfterSeconds int32
	ClearEndOfDay     bool
	ClearOnLogout     bool
}
