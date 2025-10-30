package rubix

import (
	"net"
	"testing"

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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, CheckCondition(tc.condition, tc.lookup))
		})
	}
}
