package rubix

import "time"

type UserState string

const (
	UserStateOnline  UserState = "online"
	UserStateOffline UserState = "offline"
	UserStateAway    UserState = "away"
	UserStateBusy    UserState = "busy"
	UserStateHiatus  UserState = "hiatus"
	UserStateActive  UserState = "active"
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

const OverlayAfterID = "overlay"

func (u *UserStatus) expiryFromNow() {
	u.ExpiryTime = time.Now().Add(time.Second * time.Duration(u.ClearAfterSeconds))
}
func (u *UserStatus) expiryFrom(at time.Time) {
	u.ExpiryTime = at.Add(time.Second * time.Duration(u.ClearAfterSeconds))
}

func (u *UserStatus) Repair() {

	overlayMap := make(map[string]UserStatus)
	overlayCount := 0

	for _, overlay := range u.Overlays {
		if !overlay.ExpiryTime.IsZero() && overlay.ExpiryTime.Before(time.Now()) {
			continue
		}
		overlayMap[overlay.ID] = overlay
		overlayCount++
	}

	u.Overlays = nil

	for _, overlay := range overlayMap {
		if overlay.AfterID != "" {
			if parentOverlay, ok := overlayMap[overlay.AfterID]; ok {
				if !parentOverlay.ExpiryTime.IsZero() {
					overlay.expiryFrom(parentOverlay.ExpiryTime)
				}
			} else if overlay.ExpiryTime.IsZero() && overlay.ClearAfterSeconds > 0 {
				overlay.expiryFromNow()
			}
		}

		u.Overlays = append(u.Overlays, overlay)
	}

	if u.ClearAfterSeconds > 0 && u.AfterID == OverlayAfterID && u.ExpiryTime.IsZero() && overlayCount == 0 {
		u.expiryFromNow()
	}

}
