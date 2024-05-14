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
	GetAuthData(workspaceUuid, userUuid string, appIDs ...app.GlobalAppID) ([]rubix.DataResult, error)

	GetPermissionStatements(lookup rubix.Lookup, permissions ...app.ScopedKey) ([]app.PermissionStatement, error)
	UserHasPermission(lookup rubix.Lookup, permissions ...app.ScopedKey) (bool, error)

	SetUserStatus(workspaceUuid, userUuid string, status rubix.UserStatus) (rubix.UserStatus, bool, error)
	GetUserStatus(workspaceUuid, userUuid string) (rubix.UserStatus, error)
	ClearUserStatusID(workspaceUuid, userUuid, statusID string) error

	Connect() error
	Close() error
}
