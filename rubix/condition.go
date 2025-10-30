package rubix

import (
	"net"
	"time"
)

type Condition struct {
	RequireMFA             bool `json:"requireMFA"`
	RequireVerifiedAccount bool `json:"requireVerifiedAccount"`
	MaxSessionAgeSeconds   int  `json:"maxSessionAgeSeconds"`

	AllowedLocations []string `json:"allowedLocations"`
	BlockedLocations []string `json:"blockedLocations"`

	AllowedIPs []string `json:"allowedIPs"`
	BlockedIPs []string `json:"blockedIPs"`
}

func CheckCondition(condition Condition, lookup Lookup) bool {
	if condition.RequireMFA && !lookup.MFA {
		return false
	}

	if condition.RequireVerifiedAccount && !lookup.VerifiedAccount {
		return false
	}

	if condition.MaxSessionAgeSeconds < 0 && time.Now().Unix()-lookup.SessionIssued.Unix() > int64(condition.MaxSessionAgeSeconds) {
		return false
	}

	if len(condition.AllowedLocations) > 0 {
		if !slices.Contains(condition.AllowedLocations, lookup.GeoLocation) {
			return false
		}
	}

	if len(condition.BlockedLocations) > 0 {
		if slices.Contains(condition.BlockedLocations, lookup.GeoLocation) {
			return false
		}
	}

	if len(condition.AllowedIPs) > 0 {
		match := false
		for _, val := range condition.AllowedIPs {
			ip := net.ParseIP(val).To16()
			if slices.Equal(ip, lookup.IpAddress.To16()) {
				match = true
				break
			}
		}

		if !match {
			return false
		}
	}

	if len(condition.BlockedIPs) > 0 {
		for _, val := range condition.BlockedIPs {
			ip := net.ParseIP(val).To16()
			if slices.Equal(ip, lookup.IpAddress.To16()) {
				return false
			}
		}
	}

	return true
}
