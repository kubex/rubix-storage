package rubix

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckRoleConstraints(t *testing.T) {
	type testCase struct {
		name        string
		constraints []UserRoleConstraint
		lookup      Lookup
		expected    bool
	}

	testCases := []testCase{
		{
			name:        "constraints is empty",
			constraints: []UserRoleConstraint{},
			lookup:      Lookup{},
			expected:    true,
		},
		{
			name:        "constraints is nil",
			constraints: nil,
			lookup:      Lookup{},
			expected:    true,
		},
		{
			name: "single constraints is met",
			constraints: []UserRoleConstraint{
				{
					Type:  UserRoleConstraintTypeIpAddress,
					Value: []interface{}{"1.1.1.1"},
				},
			},
			lookup:   Lookup{IpAddress: net.IP{1, 1, 1, 1}},
			expected: true,
		},
		{
			name: "multiple constraints are met",
			constraints: []UserRoleConstraint{
				{
					Type:  UserRoleConstraintTypeIpAddress,
					Value: []interface{}{"1.1.1.1"},
				},
				{
					Type:  UserRoleConstraintTypeLocation,
					Value: []interface{}{"FR", "UK"},
				},
			},
			lookup:   Lookup{IpAddress: net.IP{1, 1, 1, 1}, GeoLocation: "FR"},
			expected: true,
		},
		{
			name: "single constraints is not met",
			constraints: []UserRoleConstraint{
				{
					Type:  UserRoleConstraintTypeIpAddress,
					Value: []interface{}{"2.1.1.1"},
				},
			},
			lookup:   Lookup{IpAddress: net.IP{1, 1, 1, 1}},
			expected: false,
		},
		{
			name: "single constraints are not met",
			constraints: []UserRoleConstraint{
				{
					Type:  UserRoleConstraintTypeIpAddress,
					Value: []interface{}{"2.1.1.1"},
				},
				{
					Type:  UserRoleConstraintTypeLocation,
					Value: []interface{}{"FR", "UK"},
				},
			},
			lookup:   Lookup{IpAddress: net.IP{1, 1, 1, 1}, GeoLocation: "PL"},
			expected: false,
		},
		{
			name: "some constraints are met while others are not",
			constraints: []UserRoleConstraint{
				{
					Type:  UserRoleConstraintTypeIpAddress,
					Value: []interface{}{"1.1.1.1"},
				},
				{
					Type:  UserRoleConstraintTypeLocation,
					Value: []interface{}{"FR", "UK"},
				},
			},
			lookup:   Lookup{IpAddress: net.IP{1, 1, 1, 1}, GeoLocation: "PL"},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, CheckRoleConstraints(tc.constraints, tc.lookup))
		})
	}
}

func TestIsConstraintMet(t *testing.T) {
	type testCase struct {
		name       string
		constraint UserRoleConstraint
		lookup     Lookup
		expected   bool
	}

	testCases := []testCase{
		{
			name: "IP match - single",
			constraint: UserRoleConstraint{
				Type:  UserRoleConstraintTypeIpAddress,
				Value: []interface{}{"1.1.1.1"},
			},
			lookup:   Lookup{IpAddress: net.IP{1, 1, 1, 1}},
			expected: true,
		},
		{
			name: "IP match multiple",
			constraint: UserRoleConstraint{
				Type:  UserRoleConstraintTypeIpAddress,
				Value: []interface{}{"1.2.3.4", "1.1.1.1"},
			},
			lookup:   Lookup{IpAddress: net.IP{1, 1, 1, 1}},
			expected: true,
		},
		{
			name: "IP does not match",
			constraint: UserRoleConstraint{
				Type:  UserRoleConstraintTypeIpAddress,
				Value: []interface{}{"1.2.3.4", "1.1.1.1"},
			},
			lookup:   Lookup{IpAddress: net.IP{2, 1, 1, 1}},
			expected: false,
		},
		{
			name: "IP is nil",
			constraint: UserRoleConstraint{
				Type:  UserRoleConstraintTypeIpAddress,
				Value: []interface{}{"1.2.3.4", "1.1.1.1"},
			},
			lookup:   Lookup{IpAddress: nil},
			expected: false,
		},
		{
			name: "Location matches - single",
			constraint: UserRoleConstraint{
				Type:  UserRoleConstraintTypeLocation,
				Value: []interface{}{"UK"},
			},
			lookup:   Lookup{GeoLocation: "UK"},
			expected: true,
		},
		{
			name: "Location matches - multiple",
			constraint: UserRoleConstraint{
				Type:  UserRoleConstraintTypeLocation,
				Value: []interface{}{"FR", "UK"},
			},
			lookup:   Lookup{GeoLocation: "UK"},
			expected: true,
		},
		{
			name: "Location does not match",
			constraint: UserRoleConstraint{
				Type:  UserRoleConstraintTypeLocation,
				Value: []interface{}{"FR", "UK"},
			},
			lookup:   Lookup{GeoLocation: "PL"},
			expected: false,
		},
		{
			name: "No Geo",
			constraint: UserRoleConstraint{
				Type:  UserRoleConstraintTypeLocation,
				Value: []interface{}{"FR", "UK"},
			},
			lookup:   Lookup{},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, isConstraintMet(tc.constraint, tc.lookup))
		})
	}
}
