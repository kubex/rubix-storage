package rubix

type Condition struct {
	RequireMFA             bool `json:"requireMFA"`
	RequireVerifiedAccount bool `json:"requireVerifiedAccount"`
	MaxSessionAgeSeconds   int  `json:"maxSessionAgeSeconds"`

	AllowedLocations []string `json:"allowedLocations"`
	BlockedLocations []string `json:"blockedLocations"`

	AllowedIPs []string `json:"allowedIPs"`
	BlockedIPs []string `json:"blockedIPs"`
}
