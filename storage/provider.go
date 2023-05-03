package storage

import (
	"github.com/kubex/definitions-go/app"
	"github.com/kubex/rubix-storage/rubix"
)

type Provider interface {
	GetWorkspaceUUIDByAlias(alias string) (string, error)
	GetUserWorkspaceUUIDs(userId string) ([]string, error)
	GetWorkspaceUserIDs(workspaceUuid string) ([]string, error)
	RetrieveWorkspace(workspaceUuid string) (*rubix.Workspace, error)
	GetAuthData(lookups ...rubix.Lookup) (map[string]string, error)
	GetPermissionStatements(lookup rubix.Lookup, permissions ...app.ScopedKey) ([]app.PermissionStatement, error)
	UserHasPermission(lookup rubix.Lookup, permissions ...app.ScopedKey) (bool, error)

	Connect() error
	Close() error
}
