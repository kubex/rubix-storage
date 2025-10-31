package sql

import "github.com/kubex/rubix-storage/rubix"

type permissionResult struct {
	PermissionKey  string
	Resource       string
	Allow          bool
	RoleConditions rubix.Condition
	Meta           map[string][]string
}
