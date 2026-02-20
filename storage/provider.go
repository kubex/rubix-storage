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
	SetWorkspaceAccessCondition(workspaceUuid string, condition rubix.Condition) error
	GetOIDCProviders(workspace string) ([]rubix.OIDCProvider, error)
	GetOIDCProvider(workspace, uuid string) (*rubix.OIDCProvider, error)
	CreateOIDCProvider(workspace string, provider rubix.OIDCProvider) error
	MutateOIDCProvider(workspace, uuid string, options ...rubix.MutateOIDCProviderOption) error
	DeleteOIDCProvider(workspace, uuid string) error
	SetWorkspaceEmailDomainWhitelist(workspaceUuid string, domains []string) error
	SetWorkspaceMemberApprovalMode(workspaceUuid string, mode string) error

	// SCIM Activity Log
	GetSCIMActivityLog(workspace, providerUUID string, limit int) ([]rubix.SCIMActivityLog, error)
	AddSCIMActivityLog(workspace string, entry rubix.SCIMActivityLog) error

	// Workspace Users (OIDC directory)
	CreateWorkspaceUser(workspace string, user rubix.WorkspaceUser) error
	GetWorkspaceUser(workspace, userID string) (*rubix.WorkspaceUser, error)
	GetWorkspaceUsersByProvider(workspace, providerUUID string) ([]rubix.WorkspaceUser, error)
	UpdateWorkspaceUser(workspace, userID string, opts ...rubix.MutateWorkspaceUserOption) error
	DeleteWorkspaceUser(workspace, userID string) error
	GetResolvedMembers(workspace string, filter rubix.MemberFilter) ([]rubix.ResolvedMember, error)

	SetAuthData(workspaceUuid, userUuid string, value rubix.DataResult, forceUpdate bool) error

	GetSettings(workspace, vendor, app string, keys ...string) ([]rubix.Setting, error)
	SetSetting(workspace, vendor, app, key, value string) error

	AddUserToWorkspace(workspaceID, userID string, as rubix.MembershipType, partnerId string, source ...rubix.MembershipSource) error

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
	CreateRole(workspace, role, name, description string, permissions, users []string, conditions rubix.Condition, scimManaged bool) error
	MutateRole(workspace, role string, options ...rubix.MutateRoleOption) error
	GetRolePermissions(workspace, role string) ([]rubix.RolePermission, error)

	// Role Resources
	GetRoleResources(workspace, role string) ([]rubix.RoleResource, error)
	AddRoleResources(workspace, role string, resources ...rubix.RoleResource) error
	RemoveRoleResources(workspace, role string, resources ...rubix.RoleResource) error

	// Teams
	GetTeam(workspace, team string) (*rubix.Team, error)
	GetTeams(workspace string) ([]rubix.Team, error)
	GetUserTeams(workspace, user string) ([]rubix.UserTeam, error)
	DeleteTeam(workspace, team string) error
	CreateTeam(workspace, team, name, description string, users map[string]rubix.TeamLevel, scimManaged bool) error
	MutateTeam(workspace, team string, options ...rubix.MutateTeamOption) error

	// Brands
	GetBrand(workspace, brand string) (*rubix.Brand, error)
	GetBrands(workspace string) ([]rubix.Brand, error)
	CreateBrand(workspace, brand, name, description string) error
	MutateBrand(workspace, brand string, options ...rubix.MutateBrandOption) error

	// Departments
	GetDepartment(workspace, department string) (*rubix.Department, error)
	GetDepartments(workspace string) ([]rubix.Department, error)
	CreateDepartment(workspace, department, name, description string) error
	MutateDepartment(workspace, department string, options ...rubix.MutateDepartmentOption) error

	// Channels
	GetChannel(workspace, channel string) (*rubix.Channel, error)
	GetChannels(workspace string) ([]rubix.Channel, error)
	CreateChannel(workspace, channel, department, name, description string) error
	MutateChannel(workspace, channel string, options ...rubix.MutateChannelOption) error

	// Distributors
	GetDistributor(workspace, distributor string) (*rubix.Distributor, error)
	GetDistributors(workspace string) ([]rubix.Distributor, error)
	CreateDistributor(workspace, distributor, name, description string) error
	MutateDistributor(workspace, distributor string, options ...rubix.MutateDistributorOption) error

	// BPOs
	GetBPO(workspace, bpo string) (*rubix.BPO, error)
	GetBPOs(workspace string) ([]rubix.BPO, error)
	CreateBPO(workspace, bpo, name, description string) error
	MutateBPO(workspace, bpo string, options ...rubix.MutateBPOOption) error

	Initialize() error
	Connect() error
	Close() error
	Sync() error

	AfterUpdate(func()) error
}
