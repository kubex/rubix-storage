package storage

import (
	"github.com/kubex/definitions-go/app"
	"github.com/kubex/rubix-storage/rubix"
)

type Provider interface {
	GetWorkspaceUUIDByAlias(alias string) (string, error)
	GetUserWorkspaceUUIDs(userId string) ([]string, error)
	GetWorkspaceMembers(workspaceUuid, userID string) ([]rubix.Membership, error)
	RetrieveWorkspace(workspaceUuid string) (*rubix.Workspace, error)
	GetAuthData(workspaceUuid, userUuid string, appIDs ...app.GlobalAppID) ([]rubix.DataResult, error)
	AddMemberToWorkspace(workspaceID, userID string) error

	GetPermissionStatements(lookup rubix.Lookup, permissions ...app.ScopedKey) ([]app.PermissionStatement, error)
	UserHasPermission(lookup rubix.Lookup, permissions ...app.ScopedKey) (bool, error)

	CreateUser(userID, name string) error
	SetUserStatus(workspaceUuid, userUuid string, status rubix.UserStatus) (bool, error)
	GetUserStatus(workspaceUuid, userUuid string) (rubix.UserStatus, error)
	ClearUserStatusID(workspaceUuid, userUuid, statusID string) error
	ClearUserStatusLogout(workspaceUuid, userUuid string) error
	MutateUser(workspace, user string, options ...rubix.MutateUserOption) error

	SetMembershipType(workspace, user string, accountType rubix.MembershipType) error
	SetMembershipState(workspace, user string, accountType rubix.MembershipState) error
	RemoveUserFromWorkspace(workspace, user string) error

	GetRole(workspace, role string) (*rubix.Role, error)
	GetRoles(workspace string) ([]rubix.Role, error)
	GetUserRoles(workspace, user string) ([]rubix.UserRole, error)
	DeleteRole(workspace, role string) error
	CreateRole(workspace, role, title, description string, permissions, users []string) error
	MutateRole(workspace, role string, options ...rubix.MutateRoleOption) error

	Connect() error
	Close() error
}
