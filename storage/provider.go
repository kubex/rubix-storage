package storage

import (
	"github.com/kubex/definitions-go/app"
	"github.com/kubex/rubix-storage/rubix"
)

type Provider interface {
	CreateWorkspace(workspaceUuid, name, alias, domain string) error
	GetWorkspaceUUIDByAlias(alias string) (string, error)
	GetUserWorkspaceUUIDs(userId string) ([]string, error)
	GetWorkspaceMembers(workspaceUuid string, userIDs ...string) ([]rubix.Membership, error)
	RetrieveWorkspaces(workspaceUuids ...string) (map[string]*rubix.Workspace, error)
	RetrieveWorkspace(workspaceUuid string) (*rubix.Workspace, error)
	RetrieveWorkspaceByDomain(domain string) (*rubix.Workspace, error)

	GetAuthData(workspaceUuid, userUuid string, appIDs ...app.GlobalAppID) ([]rubix.DataResult, error)
	SetAuthData(workspaceUuid, userUuid string, value rubix.DataResult, forceUpdate bool) error
	AddUserToWorkspace(workspaceID, userID string, as rubix.MembershipType, partnerId string) error

	GetPermissionStatements(lookup rubix.Lookup, permissions ...app.ScopedKey) ([]app.PermissionStatement, error)
	UserHasPermission(lookup rubix.Lookup, permissions ...app.ScopedKey) (bool, error)

	CreateUser(userID, name, email string) error
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

	Initialize() error
	Connect() error
	Close() error
	Sync() error

	AfterUpdate(func()) error
}
