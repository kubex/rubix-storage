package rubix

import (
	"net"
	"slices"
	"strings"
	"time"
)

type Condition struct {
	RequireMFA             bool `json:"requireMFA"`
	RequireVerifiedAccount bool `json:"requireVerifiedAccount"`
	MaxSessionAgeSeconds   int  `json:"maxSessionAgeSeconds"`

	AllowedLocations []string `json:"allowedLocations"`
	BlockedLocations []string `json:"blockedLocations"`

	AllowedIPGroups []string `json:"allowedIPGroups"`
	BlockedIPGroups []string `json:"blockedIPGroups"`
}

// IPGroupResolver returns the entries (IPs/CIDRs) for a given group ID.
// Returns nil if the group is not found.
type IPGroupResolver func(groupID string) []string

// ipMatchesEntry checks if an IP matches a single entry (IP or CIDR).
func ipMatchesEntry(ip net.IP, entry string) bool {
	if strings.Contains(entry, "/") {
		_, cidr, err := net.ParseCIDR(entry)
		if err != nil {
			return false
		}
		return cidr.Contains(ip)
	}
	parsed := net.ParseIP(entry)
	if parsed == nil {
		return false
	}
	return ip.Equal(parsed)
}

// collectGroupEntries resolves group IDs to a flat list of entries.
func collectGroupEntries(groupIDs []string, resolver IPGroupResolver) []string {
	if resolver == nil {
		return nil
	}
	var all []string
	for _, gid := range groupIDs {
		if entries := resolver(gid); entries != nil {
			all = append(all, entries...)
		}
	}
	return all
}

func CheckCondition(condition Condition, lookup Lookup, ipGroups ...IPGroupResolver) bool {
	if condition.RequireMFA && !lookup.MFA {
		return false
	}

	if condition.RequireVerifiedAccount && !lookup.VerifiedAccount {
		return false
	}

	if condition.MaxSessionAgeSeconds > 0 && time.Now().Unix()-lookup.SessionIssued.Unix() > int64(condition.MaxSessionAgeSeconds) {
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

	// IP Group checks
	var resolver IPGroupResolver
	if len(ipGroups) > 0 {
		resolver = ipGroups[0]
	}

	allowedEntries := collectGroupEntries(condition.AllowedIPGroups, resolver)
	if len(allowedEntries) > 0 {
		if lookup.IpAddress == nil {
			return false
		}
		match := false
		for _, entry := range allowedEntries {
			if ipMatchesEntry(lookup.IpAddress, entry) {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}

	blockedEntries := collectGroupEntries(condition.BlockedIPGroups, resolver)
	if len(blockedEntries) > 0 && lookup.IpAddress != nil {
		for _, entry := range blockedEntries {
			if ipMatchesEntry(lookup.IpAddress, entry) {
				return false
			}
		}
	}

	return true
}
