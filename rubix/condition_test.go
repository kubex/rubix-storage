package rubix

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func mockResolver(groups map[string][]string) IPGroupResolver {
	return func(groupID string) []string {
		return groups[groupID]
	}
}

func TestCheckCondition(t *testing.T) {
	type testCase struct {
		name      string
		condition Condition
		lookup    Lookup
		resolver  IPGroupResolver
		expected  bool
	}

	groups := map[string][]string{
		"office":     {"1.1.1.1", "1.2.3.4"},
		"vpn":        {"10.0.0.0/8"},
		"blocked":    {"9.9.9.9"},
		"cidr-block": {"192.168.0.0/16"},
		"ipv6-group": {"2001:db8::/32"},
	}
	resolver := mockResolver(groups)

	testCases := []testCase{
		// --- Non-IP conditions ---
		{
			name:      "No Conditions",
			condition: Condition{},
			lookup:    Lookup{},
			expected:  true,
		},
		{
			name:      "Require MFA - valid",
			condition: Condition{RequireMFA: true},
			lookup:    Lookup{MFA: true},
			expected:  true,
		},
		{
			name:      "Require MFA - invalid",
			condition: Condition{RequireMFA: true},
			lookup:    Lookup{MFA: false},
			expected:  false,
		},
		{
			name:      "Require Verified - valid",
			condition: Condition{RequireVerifiedAccount: true},
			lookup:    Lookup{VerifiedAccount: true},
			expected:  true,
		},
		{
			name:      "Require Verified - invalid",
			condition: Condition{RequireVerifiedAccount: true},
			lookup:    Lookup{VerifiedAccount: false},
			expected:  false,
		},
		{
			name:      "Max Session Age - valid",
			condition: Condition{MaxSessionAgeSeconds: 1000},
			lookup:    Lookup{SessionIssued: time.Now().Add(-500 * time.Second)},
			expected:  true,
		},
		{
			name:      "Max Session Age - invalid",
			condition: Condition{MaxSessionAgeSeconds: 400},
			lookup:    Lookup{SessionIssued: time.Now().Add(-500 * time.Second)},
			expected:  false,
		},
		// --- Location conditions ---
		{
			name:      "Location matches - single",
			condition: Condition{AllowedLocations: []string{"UK"}},
			lookup:    Lookup{GeoLocation: "UK"},
			expected:  true,
		},
		{
			name:      "Location does not match",
			condition: Condition{AllowedLocations: []string{"FR", "UK"}},
			lookup:    Lookup{GeoLocation: "PL"},
			expected:  false,
		},
		{
			name:      "Blocked location matches",
			condition: Condition{BlockedLocations: []string{"PL", "FR"}},
			lookup:    Lookup{GeoLocation: "FR"},
			expected:  false,
		},
		{
			name:      "Blocked location does not match",
			condition: Condition{BlockedLocations: []string{"PL", "FR"}},
			lookup:    Lookup{GeoLocation: "GB"},
			expected:  true,
		},
		// --- IP Group conditions ---
		{
			name:      "Allowed IP group - match exact IP",
			condition: Condition{AllowedIPGroups: []string{"office"}},
			lookup:    Lookup{IpAddress: net.IP{1, 1, 1, 1}},
			resolver:  resolver,
			expected:  true,
		},
		{
			name:      "Allowed IP group - no match",
			condition: Condition{AllowedIPGroups: []string{"office"}},
			lookup:    Lookup{IpAddress: net.IP{2, 2, 2, 2}},
			resolver:  resolver,
			expected:  false,
		},
		{
			name:      "Allowed IP group - CIDR match",
			condition: Condition{AllowedIPGroups: []string{"vpn"}},
			lookup:    Lookup{IpAddress: net.ParseIP("10.5.3.1")},
			resolver:  resolver,
			expected:  true,
		},
		{
			name:      "Allowed IP group - CIDR no match",
			condition: Condition{AllowedIPGroups: []string{"vpn"}},
			lookup:    Lookup{IpAddress: net.ParseIP("11.0.0.1")},
			resolver:  resolver,
			expected:  false,
		},
		{
			name:      "Blocked IP group - match",
			condition: Condition{BlockedIPGroups: []string{"blocked"}},
			lookup:    Lookup{IpAddress: net.ParseIP("9.9.9.9")},
			resolver:  resolver,
			expected:  false,
		},
		{
			name:      "Blocked IP group - no match",
			condition: Condition{BlockedIPGroups: []string{"blocked"}},
			lookup:    Lookup{IpAddress: net.ParseIP("8.8.8.8")},
			resolver:  resolver,
			expected:  true,
		},
		{
			name:      "Blocked IP group - CIDR match",
			condition: Condition{BlockedIPGroups: []string{"cidr-block"}},
			lookup:    Lookup{IpAddress: net.ParseIP("192.168.1.50")},
			resolver:  resolver,
			expected:  false,
		},
		{
			name:      "Allowed IP group - nil IP",
			condition: Condition{AllowedIPGroups: []string{"office"}},
			lookup:    Lookup{IpAddress: nil},
			resolver:  resolver,
			expected:  false,
		},
		{
			name:      "Multiple allowed groups - match in second group",
			condition: Condition{AllowedIPGroups: []string{"office", "vpn"}},
			lookup:    Lookup{IpAddress: net.ParseIP("10.0.0.5")},
			resolver:  resolver,
			expected:  true,
		},
		{
			name:      "No resolver - allowed groups ignored",
			condition: Condition{AllowedIPGroups: []string{"office"}},
			lookup:    Lookup{IpAddress: net.ParseIP("99.99.99.99")},
			resolver:  nil,
			expected:  true,
		},
		{
			name:      "IPv6 group - match",
			condition: Condition{AllowedIPGroups: []string{"ipv6-group"}},
			lookup:    Lookup{IpAddress: net.ParseIP("2001:db8::1")},
			resolver:  resolver,
			expected:  true,
		},
		{
			name:      "IPv6 group - no match",
			condition: Condition{AllowedIPGroups: []string{"ipv6-group"}},
			lookup:    Lookup{IpAddress: net.ParseIP("2001:db9::1")},
			resolver:  resolver,
			expected:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.resolver != nil {
				assert.Equal(t, tc.expected, CheckCondition(tc.condition, tc.lookup, tc.resolver))
			} else {
				assert.Equal(t, tc.expected, CheckCondition(tc.condition, tc.lookup))
			}
		})
	}
}
