package storage

import (
	"github.com/kubex/rubix-storage/rubix"
	"github.com/openbyte-os/sdk-go/app"
	"time"
)

type Provider interface {
	GetGatewaySourceIPs(scope rubix.GatewaySourceScope, vendor, app string) ([]rubix.GatewaySourceIP, error)
	GetEffectiveGatewaySourceIPs(vendor, app string) ([]rubix.GatewaySourceIP, error)
	AddGatewaySourceIP(entry rubix.GatewaySourceIP) (rubix.GatewaySourceIP, error)
	DeleteGatewaySourceIP(scope rubix.GatewaySourceScope, vendor, app, id string) error

	// Gateway credentials are global to an app. Raw tokens are returned once on
	// create/rotate; listing and verification expose metadata only.
	CreateGatewayCredential(vendor, app, name, actor string, expires time.Time) (rubix.IssuedGatewayCredential, error)
	GetGatewayCredentials(vendor, app string) ([]rubix.GatewayCredential, error)
	VerifyGatewayCredential(token string) (*rubix.GatewayCredential, error)
	RotateGatewayCredential(vendor, app, id, actor string, overlap time.Duration) (rubix.IssuedGatewayCredential, error)
	RevokeGatewayCredential(vendor, app, id string) error

	CreateWorkspace(workspaceUuid, name, alias, domain string) error
	GetWorkspaceUUIDByAlias(alias string) (string, error)
	GetUserWorkspaceUUIDs(userId string) ([]string, error)
	GetWorkspaceMembers(workspaceUuid string, userIDs ...string) ([]rubix.Membership, error)
	RetrieveAllWorkspaces() (map[string]*rubix.Workspace, error)
	RetrieveWorkspaces(workspaceUuids ...string) (map[string]*rubix.Workspace, error)
	// RetrieveWorkspace and RetrieveWorkspaceByDomain return rubix.ErrNoResultFound
	// when no workspace matches.
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
	SetWorkspaceEmailDomainApproval(workspaceUuid string, approval map[string]string) error
	SetWorkspaceMemberApprovalMode(workspaceUuid string, mode string) error
	SetWorkspaceName(workspaceUuid, name string) error
	SetWorkspaceIcon(workspaceUuid, icon string) error
	SetWorkspaceDefaultApp(workspaceUuid string, defaultApp string) error
	SetWorkspaceMetricTickers(workspaceUuid string, tickers rubix.MetricTickers) error
	SetWorkspaceSystemVendors(workspaceUuid string, vendors []string) error
	SetWorkspaceInstalledApplications(workspaceUuid string, apps []app.ScopedKey) error

	// Workspace Applications (relational table)
	GetWorkspaceApplications(workspaceUuid string) ([]app.ScopedKey, error)
	SetWorkspaceApplication(workspaceUuid, vendorID, appID, releaseChannel string) error
	RemoveWorkspaceApplication(workspaceUuid, vendorID, appID string) error

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
	DeleteSetting(workspace, vendor, app, key string) error
	StoreTaskRun(run rubix.TaskRun, maxRuns int, notBefore time.Time) error
	GetTaskRuns(taskKeys []string, notBefore time.Time) ([]rubix.TaskRun, error)
	PruneTaskRuns(notBefore time.Time) error
	SetSetting(workspace, vendor, app, key, value string) error

	AddUserToWorkspace(workspaceID, userID string, as rubix.MembershipType, partnerId string, source ...rubix.MembershipSource) error

	GetPermissionStatements(lookup rubix.Lookup, permissions ...app.ScopedKey) ([]app.PermissionStatement, error)
	UserHasPermission(lookup rubix.Lookup, permissions ...app.ScopedKey) (bool, error)

	CreateUser(userID, name, email string) error
	GetUser(workspace, userID string) (*rubix.User, error)
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

	// BPO Management
	GetBPOManagers(workspace, bpo string) ([]string, error)
	SetBPOManagers(workspace, bpo string, users []string) error
	GetBPOTeams(workspace, bpo string) ([]string, error)
	SetBPOTeams(workspace, bpo string, teams []string) error
	GetBPORoles(workspace, bpo string) ([]string, error)
	SetBPORoles(workspace, bpo string, roles []string) error
	GetManagedBPOs(workspace, user string) ([]string, error)
	SetMemberPartnerID(workspace, user, partnerID string) error

	// IP Groups
	GetIPGroup(workspace, groupID string) (*rubix.IPGroup, error)
	GetIPGroups(workspace string) ([]rubix.IPGroup, error)
	CreateIPGroup(workspace string, group rubix.IPGroup) error
	MutateIPGroup(workspace, groupID string, options ...rubix.MutateIPGroupOption) error
	DeleteIPGroup(workspace, groupID string) error

	// App Activation
	CompleteActivationStep(workspace, user, vendor, app, stepID string) error
	ResetActivationSteps(workspace, vendor, app string) error
	GetActivationState(workspace, user, vendor, app string) ([]rubix.ActivationState, error)

	// Platform Applications
	GetPlatformApplications() ([]rubix.PlatformApplication, error)
	// StorePlatformApplication writes every column, including Status; callers updating an
	// application must carry the stored Status forward.
	StorePlatformApplication(application rubix.PlatformApplication) error
	RemovePlatformApplication(vendorID, appID, releaseChannel string) error

	// App Listings are an app's marketplace presentation, one per app.
	// GetVendorAppOrders is each vendor's preferred order of its apps on its
	// marketplace page, keyed by vendor ID. Vendors without one are absent.
	GetVendorAppOrders() (map[string][]string, error)
	// StoreVendorAppOrder replaces the vendor's preferred app order; an empty
	// list removes it.
	StoreVendorAppOrder(order rubix.VendorAppOrder) error
	GetAppListing(vendorID, appID string) (*rubix.AppListing, error)
	GetAppListings() ([]rubix.AppListing, error)
	StoreAppListing(listing rubix.AppListing) error
	// GetAppInstallCounts is the number of workspaces each app is installed in, keyed by "vendor/app".
	GetAppInstallCounts() (map[string]int, error)

	// App Versions are the versions rubix has seen each app running, from health checks.
	RecordAppVersion(version rubix.AppVersion) error
	GetAppVersions(vendorID, appID, releaseChannel string) ([]rubix.AppVersion, error)
	// GetLiveAppVersions is each app's live version on the channel, keyed by "vendor/app".
	GetLiveAppVersions(releaseChannel string) (map[string]rubix.AppVersion, error)
	// App Releases are vendor release notes per version, written before or after the version is seen.
	GetAppReleases(vendorID, appID string) ([]rubix.AppRelease, error)
	// StoreAppRelease saves a version's notes; no notes removes them.
	StoreAppRelease(release rubix.AppRelease) error
	// App Listing Assets are images uploaded for listings, referenced as "asset:<name>".
	StoreAppListingAsset(asset rubix.AppListingAsset) error
	GetAppListingAsset(vendorID, appID, name string) (*rubix.AppListingAsset, error)
	// PruneAppListingAssets removes the app's assets whose names are not in keep.
	PruneAppListingAssets(vendorID, appID string, keep []string) error

	// Platform Vendors
	GetPlatformVendors() ([]rubix.PlatformVendor, error)
	// StorePlatformVendor writes every vendor column, including SystemVendor; callers updating a
	// vendor must carry the stored SystemVendor value forward.
	StorePlatformVendor(vendor rubix.PlatformVendor) error
	RemovePlatformVendor(vendorID string) error

	// Vendor Members
	// GetVendorMembers is ordered by name, email, then user ID.
	GetVendorMembers(vendorID string) ([]rubix.VendorMember, error)
	// GetVendorMember returns nil, nil when the user is not a member.
	GetVendorMember(vendorID, userID string) (*rubix.VendorMember, error)
	// GetUserVendorMemberships is ordered by vendor ID.
	GetUserVendorMemberships(userID string) ([]rubix.VendorMember, error)
	// StoreVendorMember upserts a member; an existing member keeps AddedBy and Since and only IsAdmin and IsOwner change.
	// IsOwner implies IsAdmin: an owner is always stored as an admin.
	StoreVendorMember(member rubix.VendorMember) error
	RemoveVendorMember(vendorID, userID string) error
	// GetUserByEmail matches the global users table case-insensitively; nil, nil when none.
	GetUserByEmail(email string) (*rubix.User, error)
	// GetUserByID reads the global users table; nil, nil when none.
	GetUserByID(userID string) (*rubix.User, error)

	// Service Providers
	GetServiceProvider(workspace, serviceID string) (*rubix.ServiceProvider, error)
	GetServiceProviders(workspace string) ([]rubix.ServiceProvider, error)
	GetServiceProvidersByType(workspace, serviceProvider string) ([]rubix.ServiceProvider, error)
	CreateServiceProvider(workspace string, sp rubix.ServiceProvider) error
	MutateServiceProvider(workspace, serviceID string, options ...rubix.MutateServiceProviderOption) error
	DeleteServiceProvider(workspace, serviceID string) error

	// Blueprints
	GetBlueprints() ([]rubix.Blueprint, error)
	GetBlueprint(vendorID, appID, blueprintID string) (*rubix.Blueprint, error)
	StoreBlueprint(blueprint rubix.Blueprint) error
	RemoveBlueprint(vendorID, appID, blueprintID string) error

	GetBlueprintVersions(vendorID, appID, blueprintID string) ([]rubix.BlueprintVersion, error)
	GetBlueprintVersion(vendorID, appID, blueprintID, version string) (*rubix.BlueprintVersion, error)
	StoreBlueprintVersion(version rubix.BlueprintVersion) error

	GetWorkspaceBlueprints(workspaceUUID string) ([]rubix.WorkspaceBlueprint, error)
	SubscribeWorkspaceBlueprint(sub rubix.WorkspaceBlueprint) error
	UnsubscribeWorkspaceBlueprint(workspaceUUID, vendorID, appID, blueprintID string) error
	UpdateWorkspaceBlueprintStatus(workspaceUUID, vendorID, appID, blueprintID, status string) error
	UpdateWorkspaceBlueprintVersion(workspaceUUID, vendorID, appID, blueprintID, version string) error

	GetWorkspaceBlueprintResources(workspaceUUID, vendorID, appID, blueprintID string) ([]rubix.WorkspaceBlueprintResource, error)
	SetWorkspaceBlueprintResource(resource rubix.WorkspaceBlueprintResource) error
	RemoveWorkspaceBlueprintResource(workspaceUUID, vendorID, appID, blueprintID, resourceType, resourceKey string) error

	Initialize() error
	Connect() error
	Close() error
	Sync() error

	AfterUpdate(func()) error
}
