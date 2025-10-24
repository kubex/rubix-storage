package rubix

import "slices"

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
		allowedValues, ok := constraint.Value.([]string)
		if !ok {
			return false
		}

		return slices.Contains(allowedValues, lookup.GeoLocation)
	}

	// other types aren't implemented
	return true
}
