package sql

type permissionResult struct {
	PermissionKey string
	Resource      string
	Allow         bool
}
