package rubix

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsConstraintMet(t *testing.T) {
	type testCase struct {
		name      string
		condition Condition
		lookup    Lookup
		expected  bool
	}

	testCases := []testCase{
		{
			name: "IP match - single",
			condition: Condition{
				AllowedIPs: []string{"1.1.1.1"},
			},
			lookup:   Lookup{IpAddress: net.IP{1, 1, 1, 1}},
			expected: true,
		},
		{
			name: "IP match multiple",
			condition: Condition{
				AllowedIPs: []string{"1.2.3.4", "1.1.1.1"},
			},
			lookup:   Lookup{IpAddress: net.IP{1, 1, 1, 1}},
			expected: true,
		},
		{
			name: "IP does not match",
			condition: Condition{
				AllowedIPs: []string{"1.2.3.4", "1.1.1.1"},
			},
			lookup:   Lookup{IpAddress: net.IP{2, 1, 1, 1}},
			expected: false,
		},
		{
			name: "IP is nil",
			condition: Condition{
				AllowedIPs: []string{"1.2.3.4", "1.1.1.1"},
			},
			lookup:   Lookup{IpAddress: nil},
			expected: false,
		},
		{
			name: "Location matches - single",
			condition: Condition{
				AllowedLocations: []string{"UK"},
			},
			lookup:   Lookup{GeoLocation: "UK"},
			expected: true,
		},
		{
			name: "Location matches - multiple",
			condition: Condition{
				AllowedLocations: []string{"FR", "UK"},
			},
			lookup:   Lookup{GeoLocation: "UK"},
			expected: true,
		},
		{
			name: "Location does not match",
			condition: Condition{
				AllowedLocations: []string{"FR", "UK"},
			},
			lookup:   Lookup{GeoLocation: "PL"},
			expected: false,
		},
		{
			name: "No Geo",
			condition: Condition{
				AllowedLocations: []string{"FR", "UK"},
			},
			lookup:   Lookup{},
			expected: false,
		},
		{
			name:      "No Conditions",
			condition: Condition{},
			lookup:    Lookup{},
			expected:  true,
		},
		{
			name: "Max Session Age - valid",
			condition: Condition{
				MaxSessionAgeSeconds: 1000,
			},
			lookup: Lookup{
				SessionIssued: time.Now().Add(-500 * time.Second),
			},
			expected: true,
		},
		{
			name: "Max Session Age - invalid",
			condition: Condition{
				MaxSessionAgeSeconds: 400,
			},
			lookup: Lookup{
				SessionIssued: time.Now().Add(-500 * time.Second),
			},
			expected: false,
		},
		{
			name: "Blocked location matches",
			condition: Condition{
				BlockedLocations: []string{"PL", "FR"},
			},
			lookup: Lookup{
				GeoLocation: "FR",
			},
			expected: false,
		},
		{
			name: "Blocked location does not match",
			condition: Condition{
				BlockedLocations: []string{"PL", "FR"},
			},
			lookup: Lookup{
				GeoLocation: "GB",
			},
			expected: true,
		},
		{
			name: "Blocked IP matches",
			condition: Condition{
				BlockedIPs: []string{"1.2.3.4", "1.1.1.1"},
			},
			lookup: Lookup{
				IpAddress: net.IP{1, 1, 1, 1},
			},
			expected: false,
		},
		{
			name: "Blocked IP does not match",
			condition: Condition{
				BlockedIPs: []string{"1.2.3.4", "1.1.1.1"},
			},
			lookup: Lookup{
				IpAddress: net.IP{2, 1, 1, 1},
			},
			expected: true,
		},
		{
			name: "Require MFA - invalid",
			condition: Condition{
				RequireMFA: true,
			},
			lookup: Lookup{
				MFA: false,
			},
			expected: false,
		},
		{
			name: "Require MFA - valid",
			condition: Condition{
				RequireMFA: true,
			},
			lookup: Lookup{
				MFA: true,
			},
			expected: true,
		},

		{
			name: "Require Verified - invalid",
			condition: Condition{
				RequireVerifiedAccount: true,
			},
			lookup: Lookup{
				VerifiedAccount: false,
			},
			expected: false,
		},
		{
			name: "Require Verified - valid",
			condition: Condition{
				RequireVerifiedAccount: true,
			},
			lookup: Lookup{
				VerifiedAccount: true,
			},
			expected: true,
		},
		{
			name: "IPv6",
			condition: Condition{
				AllowedIPs: []string{"009b:35eb:8f09:2712:401f:51c3:ccea:8a21", "560f:6261:b7ff:cfaf:0af3:06e4:2063:3e8e"},
			},
			lookup: Lookup{
				IpAddress: net.ParseIP("560f:6261:b7ff:cfaf:0af3:06e4:2063:3e8e"),
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, CheckCondition(tc.condition, tc.lookup))
		})
	}
}
