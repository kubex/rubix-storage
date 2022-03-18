package rubix

import (
	"github.com/kubex/definitions-go/app"
)

type Provider interface {
	GetUserWorkspaceAliases(userId string) ([]string, error)
	GetWorkspaceUserIDs(workspaceUuid string) ([]string, error)
	RetrieveWorkspace(workspaceUuid string) (*Workspace, error)
	GetAuthData(lookup Lookup) (map[string]string, error)
	GetPermissionStatements(lookup Lookup, permissions ...app.ScopedKey) ([]app.PermissionStatement, error)
	UserHasPermission(lookup Lookup, permissions ...app.ScopedKey) (bool, error)
}
