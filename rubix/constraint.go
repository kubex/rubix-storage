package rubix

import (
	"net"
	"slices"
)

func CheckRoleConstraints(constraints []UserRoleConstraint, lookup Lookup) bool {
	if constraints == nil || len(constraints) == 0 {
		return true
	}

	for _, constraint := range constraints {
		if !isConstraintMet(constraint, lookup) {
			return false
		}
	}

	return true
}

func isConstraintMet(constraint UserRoleConstraint, lookup Lookup) bool {
	if constraint.Type == UserRoleConstraintTypeLocation {
		allowedValues, ok := constraint.Value.([]interface{})
		if !ok {
			return false
		}

		for _, val := range allowedValues {
			location, ok := val.(string)
			if !ok {
				return false
			}

			if location == lookup.GeoLocation {
				return true
			}
		}

		return false
	}

	if constraint.Type == UserRoleConstraintTypeIpAddress {
		if lookup.IpAddress == nil { // Unable to determine IP?
			return false
		}

		allowedValues, ok := constraint.Value.([]interface{})
		if !ok {
			return false
		}

		for _, val := range allowedValues {
			ipString, ok := val.(string)
			if !ok {
				return false
			}

			ip := net.ParseIP(ipString).To16()
			if slices.Equal(ip, lookup.IpAddress.To16()) {
				return true
			}
		}

		return false
	}

	// other types aren't implemented
	return true
}
