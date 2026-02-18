package sql

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kubex/definitions-go/app"
	"github.com/kubex/rubix-storage/rubix"
)

func newTestProvider(t *testing.T) *Provider {
	t.Helper()
	tdir := t.TempDir()
	dbPath := filepath.Join(tdir, "rubix_it.db")
	// libsql driver supports file: DSN for local sqlite databases
	dsn := "file:" + dbPath
	p := &Provider{SqlLite: true, PrimaryDSN: dsn}
	if err := p.Initialize(); err != nil {
		t.Fatalf("init provider: %v", err)
	}
	// Ensure file created
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("sqlite file not created: %v", err)
	}
	return p
}

func membershipLastUpdate(t *testing.T, p *Provider, ws, user string) time.Time {
	row := p.primaryConnection.QueryRow("SELECT lastUpdate FROM workspace_memberships WHERE workspace = ? AND user = ?", ws, user)
	var tsStr string
	if err := row.Scan(&tsStr); err != nil {
		t.Fatalf("scan membership lastUpdate: %v", err)
	}
	return timeFromString(tsStr)
}

func roleLastUpdate(t *testing.T, p *Provider, ws, role string) time.Time {
	row := p.primaryConnection.QueryRow("SELECT lastUpdate FROM roles WHERE workspace = ? AND role = ?", ws, role)
	var tsStr string
	if err := row.Scan(&tsStr); err != nil {
		t.Fatalf("scan role lastUpdate: %v", err)
	}
	return timeFromString(tsStr)
}

func TestIntegration_SQLite_EndToEnd(t *testing.T) {
	p := newTestProvider(t)
	defer func() { _ = p.Close() }()

	ws := "ws-123"
	if err := p.CreateWorkspace(ws, "Workspace", "acme", "acme.local"); err != nil {
		t.Fatalf("CreateWorkspace: %v", err)
	}
	if got, err := p.GetWorkspaceUUIDByAlias("acme"); err != nil || got != ws {
		t.Fatalf("GetWorkspaceUUIDByAlias: got %q err %v", got, err)
	}

	// Workspace access condition
	cond := rubix.Condition{RequireMFA: true, AllowedLocations: []string{"US", "GB"}}
	if err := p.SetWorkspaceAccessCondition(ws, cond); err != nil {
		t.Fatalf("SetWorkspaceAccessCondition: %v", err)
	}
	wsObj, err := p.RetrieveWorkspace(ws)
	if err != nil || wsObj == nil {
		t.Fatalf("RetrieveWorkspace after setting condition: %v", err)
	}
	if !wsObj.AccessCondition.RequireMFA {
		t.Fatalf("expected RequireMFA=true, got false")
	}
	if len(wsObj.AccessCondition.AllowedLocations) != 2 || wsObj.AccessCondition.AllowedLocations[0] != "US" {
		t.Fatalf("expected AllowedLocations=[US GB], got %v", wsObj.AccessCondition.AllowedLocations)
	}

	// Users and membership
	if err := p.CreateUser("u1", "Alice", "alice@example.com"); err != nil {
		t.Fatalf("CreateUser u1: %v", err)
	}
	if err := p.CreateUser("u2", "Bob", "bob@example.com"); err != nil {
		t.Fatalf("CreateUser u2: %v", err)
	}
	if err := p.AddUserToWorkspace(ws, "u1", rubix.MembershipTypeOwner, ""); err != nil {
		t.Fatalf("AddUserToWorkspace u1: %v", err)
	}
	if err := p.AddUserToWorkspace(ws, "u2", rubix.MembershipTypeMember, ""); err != nil {
		t.Fatalf("AddUserToWorkspace u2: %v", err)
	}
	members, err := p.GetWorkspaceMembers(ws)
	if err != nil || len(members) != 2 {
		t.Fatalf("GetWorkspaceMembers: len=%d err=%v", len(members), err)
	}

	// Roles
	permRead := "perm.read"
	permWrite := "perm.write"
	if err := p.CreateRole(ws, "r-admin", "Admin", "All the power", []string{permRead}, []string{"u1"}, rubix.Condition{}); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	// Capture timestamps before role/user changes
	memBefore := membershipLastUpdate(t, p, ws, "u2")
	roleBefore := roleLastUpdate(t, p, ws, "r-admin")
	// Ensure time can advance for CURRENT_TIMESTAMP resolution
	time.Sleep(1100 * time.Millisecond)
	if err := p.MutateRole(ws, "r-admin",
		rubix.WithUsersToAdd("u2"),
		rubix.WithPermsToAdd(permWrite),
		rubix.WithPermsToRemove(permRead),
	); err != nil {
		t.Fatalf("MutateRole: %v", err)
	}
	role, err := p.GetRole(ws, "r-admin")
	if err != nil {
		t.Fatalf("GetRole: %v", err)
	}
	if role.Name != "Admin" || role.ID != "r-admin" {
		t.Fatalf("GetRole mismatch: %+v", role)
	}
	roles, err := p.GetRoles(ws)
	if err != nil || len(roles) != 1 {
		t.Fatalf("GetRoles: len=%d err=%v", len(roles), err)
	}
	ur, err := p.GetUserRoles(ws, "u2")
	if err != nil || len(ur) != 1 || ur[0].Role != "r-admin" {
		t.Fatalf("GetUserRoles: %+v err=%v", ur, err)
	}
	// Verify timestamps bumped
	memAfterAdd := membershipLastUpdate(t, p, ws, "u2")
	if !memAfterAdd.After(memBefore) {
		t.Fatalf("workspace_memberships.lastUpdate did not increase after adding user to role: before=%v after=%v", memBefore, memAfterAdd)
	}
	roleAfterPerms := roleLastUpdate(t, p, ws, "r-admin")
	if !roleAfterPerms.After(roleBefore) {
		t.Fatalf("roles.lastUpdate did not increase after permission change: before=%v after=%v", roleBefore, roleAfterPerms)
	}
	// Permission checks: role should include the write permission
	role, err = p.GetRole(ws, "r-admin")
	if err != nil {
		t.Fatalf("GetRole (after perms): %v", err)
	}
	found := false
	for _, rp := range role.Permissions {
		if rp.Permission == permWrite && rp.Allow {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected role to have write permission, got %+v", role.Permissions)
	}
	// Change permission options should bump role lastUpdate
	time.Sleep(1100 * time.Millisecond)
	if err := p.MutateRole(ws, "r-admin", rubix.WithPermOptionToAdd(map[string]map[string][]string{
		permWrite: {"scope": []string{"team:eng"}},
	})); err != nil {
		t.Fatalf("MutateRole options: %v", err)
	}
	roleAfterOpts := roleLastUpdate(t, p, ws, "r-admin")
	if !roleAfterOpts.After(roleAfterPerms) {
		t.Fatalf("roles.lastUpdate did not increase after options change: before=%v after=%v", roleAfterPerms, roleAfterOpts)
	}
	// Removing the user from role should bump membership lastUpdate again
	time.Sleep(1100 * time.Millisecond)
	if err := p.MutateRole(ws, "r-admin", rubix.WithUsersToRemove("u2")); err != nil {
		t.Fatalf("MutateRole remove user: %v", err)
	}
	memAfterRem := membershipLastUpdate(t, p, ws, "u2")
	if !memAfterRem.After(memAfterAdd) {
		t.Fatalf("workspace_memberships.lastUpdate did not increase after removing user from role: before=%v after=%v", memAfterAdd, memAfterRem)
	}

	// Teams with levels
	if err := p.CreateTeam(ws, "engineering", "Engineering", "Eng team", map[string]rubix.TeamLevel{"u1": rubix.TeamLevelOwner}); err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	if err := p.MutateTeam(ws, "engineering",
		rubix.WithTeamUsersToAdd(rubix.TeamLevelMember, "u2"),
	); err != nil {
		t.Fatalf("MutateTeam add: %v", err)
	}
	if err := p.MutateTeam(ws, "engineering",
		rubix.WithTeamUsersLevel(rubix.TeamLevelManager, "u2"),
	); err != nil {
		t.Fatalf("MutateTeam level: %v", err)
	}
	g, err := p.GetTeam(ws, "engineering")
	if err != nil || len(g.Members) != 2 {
		t.Fatalf("GetTeam members=%d err=%v", len(g.Members), err)
	}
	ugs, err := p.GetUserTeams(ws, "u2")
	if err != nil || len(ugs) != 1 || ugs[0].Level != rubix.TeamLevelManager {
		t.Fatalf("GetUserTeams: %+v err=%v", ugs, err)
	}

	// Brands / Departments / Channels
	if err := p.CreateBrand(ws, "acme", "ACME", "ACME Corp"); err != nil {
		t.Fatalf("CreateBrand: %v", err)
	}
	if err := p.MutateBrand(ws, "acme", rubix.WithBrandDescription("Updated")); err != nil {
		t.Fatalf("MutateBrand: %v", err)
	}
	b, err := p.GetBrand(ws, "acme")
	if err != nil || b.Description != "Updated" {
		t.Fatalf("GetBrand: %+v err=%v", b, err)
	}
	if err := p.CreateDepartment(ws, "support", "Support", "Customer support"); err != nil {
		t.Fatalf("CreateDepartment: %v", err)
	}
	if err := p.MutateDepartment(ws, "support", rubix.WithDepartmentName("Customer Support")); err != nil {
		t.Fatalf("MutateDepartment: %v", err)
	}
	d, err := p.GetDepartment(ws, "support")
	if err != nil || d.Name != "Customer Support" {
		t.Fatalf("GetDepartment: %+v err=%v", d, err)
	}
	if err := p.CreateChannel(ws, "email", "support", "Email", "Inbound email"); err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}
	if err := p.MutateChannel(ws, "email", rubix.WithChannelDescription("Inbound email queue")); err != nil {
		t.Fatalf("MutateChannel: %v", err)
	}
	ch, err := p.GetChannel(ws, "email")
	if err != nil || ch.Description != "Inbound email queue" {
		t.Fatalf("GetChannel: %+v err=%v", ch, err)
	}

	// Distributors
	if err := p.CreateDistributor(ws, "dist-1", "Distributor One", "First distributor"); err != nil {
		t.Fatalf("CreateDistributor: %v", err)
	}
	if err := p.MutateDistributor(ws, "dist-1",
		rubix.WithDistributorDescription("Updated distributor"),
		rubix.WithDistributorWebsiteURL("https://example.com"),
		rubix.WithDistributorLogoURL("https://example.com/logo.png"),
	); err != nil {
		t.Fatalf("MutateDistributor: %v", err)
	}
	dist, err := p.GetDistributor(ws, "dist-1")
	if err != nil || dist.Description != "Updated distributor" || dist.WebsiteURL != "https://example.com" || dist.LogoURL != "https://example.com/logo.png" {
		t.Fatalf("GetDistributor: %+v err=%v", dist, err)
	}
	dists, err := p.GetDistributors(ws)
	if err != nil || len(dists) != 1 {
		t.Fatalf("GetDistributors: len=%d err=%v", len(dists), err)
	}

	// BPOs
	if err := p.CreateBPO(ws, "bpo-1", "BPO One", "First BPO"); err != nil {
		t.Fatalf("CreateBPO: %v", err)
	}
	if err := p.MutateBPO(ws, "bpo-1",
		rubix.WithBPODescription("Updated BPO"),
		rubix.WithBPOWebsiteURL("https://bpo.example.com"),
		rubix.WithBPOLogoURL("https://bpo.example.com/logo.png"),
	); err != nil {
		t.Fatalf("MutateBPO: %v", err)
	}
	bpo, err := p.GetBPO(ws, "bpo-1")
	if err != nil || bpo.Description != "Updated BPO" || bpo.WebsiteURL != "https://bpo.example.com" || bpo.LogoURL != "https://bpo.example.com/logo.png" {
		t.Fatalf("GetBPO: %+v err=%v", bpo, err)
	}
	bpos, err := p.GetBPOs(ws)
	if err != nil || len(bpos) != 1 {
		t.Fatalf("GetBPOs: len=%d err=%v", len(bpos), err)
	}

	// Role Resources
	roleBeforeRes := roleLastUpdate(t, p, ws, "r-admin")
	time.Sleep(1100 * time.Millisecond)
	if err := p.AddRoleResources(ws, "r-admin",
		rubix.RoleResource{Resource: "acme", ResourceType: rubix.ResourceTypeBrand},
		rubix.RoleResource{Resource: "support", ResourceType: rubix.ResourceTypeDepartment},
		rubix.RoleResource{Resource: "email", ResourceType: rubix.ResourceTypeChannel},
	); err != nil {
		t.Fatalf("AddRoleResources: %v", err)
	}
	resList, err := p.GetRoleResources(ws, "r-admin")
	if err != nil {
		t.Fatalf("GetRoleResources: %v", err)
	}
	if len(resList) != 3 {
		t.Fatalf("expected 3 role resources, got %d", len(resList))
	}
	wantTypes := map[string]rubix.ResourceType{"acme": rubix.ResourceTypeBrand, "support": rubix.ResourceTypeDepartment, "email": rubix.ResourceTypeChannel}
	for _, rr := range resList {
		if want, ok := wantTypes[rr.Resource]; !ok || rr.ResourceType != want {
			t.Fatalf("unexpected resource entry: %+v", rr)
		}
	}
	// GetRole includes resources
	roleWithRes, err := p.GetRole(ws, "r-admin")
	if err != nil {
		t.Fatalf("GetRole with resources: %v", err)
	}
	if len(roleWithRes.Resources) < 3 {
		t.Fatalf("expected role to have at least 3 resources, got %d", len(roleWithRes.Resources))
	}
	// lastUpdate bumped on add
	roleAfterAdd := roleLastUpdate(t, p, ws, "r-admin")
	if !roleAfterAdd.After(roleBeforeRes) {
		t.Fatalf("roles.lastUpdate did not increase after adding resources: before=%v after=%v", roleBeforeRes, roleAfterAdd)
	}
	// Adding duplicates should not bump lastUpdate
	time.Sleep(1100 * time.Millisecond)
	if err := p.AddRoleResources(ws, "r-admin",
		rubix.RoleResource{Resource: "acme", ResourceType: rubix.ResourceTypeBrand},
		rubix.RoleResource{Resource: "support", ResourceType: rubix.ResourceTypeDepartment},
		rubix.RoleResource{Resource: "email", ResourceType: rubix.ResourceTypeChannel},
	); err != nil {
		t.Fatalf("AddRoleResources duplicates: %v", err)
	}
	roleAfterDupAdd := roleLastUpdate(t, p, ws, "r-admin")
	if roleAfterDupAdd.After(roleAfterAdd) {
		t.Fatalf("roles.lastUpdate increased after duplicate resource add: before=%v after=%v", roleAfterAdd, roleAfterDupAdd)
	}
	// Remove one resource and verify lastUpdate increases
	time.Sleep(1100 * time.Millisecond)
	if err := p.RemoveRoleResources(ws, "r-admin", rubix.RoleResource{Resource: "email"}); err != nil {
		t.Fatalf("RemoveRoleResources: %v", err)
	}
	resList, err = p.GetRoleResources(ws, "r-admin")
	if err != nil {
		t.Fatalf("GetRoleResources after remove: %v", err)
	}
	if len(resList) != 2 {
		t.Fatalf("expected 2 role resources after removal, got %d", len(resList))
	}
	for _, rr := range resList {
		if rr.Resource == "email" {
			t.Fatalf("resource 'email' should have been removed")
		}
	}
	roleAfterRemRes := roleLastUpdate(t, p, ws, "r-admin")
	if !roleAfterRemRes.After(roleAfterDupAdd) {
		t.Fatalf("roles.lastUpdate did not increase after removing resource: before=%v after=%v", roleAfterDupAdd, roleAfterRemRes)
	}

	// Auth data
	ga := app.GlobalAppID{VendorID: "vendor", AppID: "app"}
	if err := p.SetAuthData(ws, "u1", rubix.DataResult{VendorID: ga.VendorID, AppID: ga.AppID, Key: "token", Value: "abc"}, true); err != nil {
		t.Fatalf("SetAuthData u1: %v", err)
	}
	if err := p.SetAuthData(ws, "", rubix.DataResult{VendorID: ga.VendorID, AppID: ga.AppID, Key: "cfg", Value: "xyz"}, true); err != nil {
		t.Fatalf("SetAuthData ws: %v", err)
	}
	ad, err := p.GetAuthData(ws, "u1", ga)
	if err != nil || len(ad) < 1 {
		t.Fatalf("GetAuthData: %v len=%d", err, len(ad))
	}

	// User status lifecycle (overlay)
	applied := rubix.UserStatus{State: rubix.UserStateBusy, ExtendedState: "in-call", ID: "status-1", ClearAfterSeconds: 60}
	ok, err := p.SetUserStatus(ws, "u1", applied)
	if err != nil || !ok {
		t.Fatalf("SetUserStatus: ok=%v err=%v", ok, err)
	}
	st, err := p.GetUserStatus(ws, "u1")
	if err != nil {
		t.Fatalf("GetUserStatus: %v", err)
	}
	if len(st.Overlays) != 1 || st.Overlays[0].ID != "status-1" || st.Overlays[0].State != rubix.UserStateBusy {
		t.Fatalf("expected one busy overlay, got %+v", st)
	}
	if err := p.ClearUserStatusID(ws, "u1", "status-1"); err != nil {
		t.Fatalf("ClearUserStatusID: %v", err)
	}
	st, err = p.GetUserStatus(ws, "u1")
	if err != nil {
		t.Fatalf("GetUserStatus after clear: %v", err)
	}
	if len(st.Overlays) != 0 {
		t.Fatalf("expected overlays cleared, got %+v", st)
	}

	// Membership state/type transitions
	if err := p.SetMembershipType(ws, "u2", rubix.MembershipTypeSupport); err != nil {
		t.Fatalf("SetMembershipType: %v", err)
	}
	if err := p.SetMembershipState(ws, "u2", rubix.MembershipStateSuspended); err != nil {
		t.Fatalf("SetMembershipState: %v", err)
	}
	if err := p.RemoveUserFromWorkspace(ws, "u2"); err != nil {
		t.Fatalf("RemoveUserFromWorkspace: %v", err)
	}
	members, err = p.GetWorkspaceMembers(ws)
	if err != nil || len(members) != 1 { // u2 is removed
		t.Fatalf("GetWorkspaceMembers after removal: len=%d err=%v", len(members), err)
	}

	// OIDC Providers
	oidc1 := rubix.OIDCProvider{
		Uuid:         "oidc-1",
		ProviderName: "Okta",
		DisplayName:  "Okta SSO Login",
		ClientID:     "client-abc",
		ClientSecret: "secret-xyz",
		ClientKeys:   `{"key":"value"}`,
		IssuerURL:    "https://okta.example.com",
		BpoID:        "bpo-1",
	}
	if err := p.CreateOIDCProvider(ws, oidc1); err != nil {
		t.Fatalf("CreateOIDCProvider: %v", err)
	}
	oidc2 := rubix.OIDCProvider{
		Uuid:         "oidc-2",
		ProviderName: "Auth0",
		ClientID:     "client-def",
		IssuerURL:    "https://auth0.example.com",
	}
	if err := p.CreateOIDCProvider(ws, oidc2); err != nil {
		t.Fatalf("CreateOIDCProvider 2: %v", err)
	}

	// Duplicate should return ErrDuplicate
	if err := p.CreateOIDCProvider(ws, oidc1); err != rubix.ErrDuplicate {
		t.Fatalf("expected ErrDuplicate, got %v", err)
	}

	// GetOIDCProviders
	providers, err := p.GetOIDCProviders(ws)
	if err != nil || len(providers) != 2 {
		t.Fatalf("GetOIDCProviders: len=%d err=%v", len(providers), err)
	}

	// GetOIDCProvider
	got, err := p.GetOIDCProvider(ws, "oidc-1")
	if err != nil {
		t.Fatalf("GetOIDCProvider: %v", err)
	}
	if got.Uuid != "oidc-1" || got.Workspace != ws || got.ProviderName != "Okta" || got.DisplayName != "Okta SSO Login" || got.ClientID != "client-abc" || got.ClientSecret != "secret-xyz" || got.ClientKeys != `{"key":"value"}` || got.IssuerURL != "https://okta.example.com" || got.BpoID != "bpo-1" {
		t.Fatalf("GetOIDCProvider mismatch: %+v", got)
	}

	// GetOIDCProvider not found
	if _, err := p.GetOIDCProvider(ws, "nonexistent"); err != rubix.ErrNoResultFound {
		t.Fatalf("expected ErrNoResultFound, got %v", err)
	}

	// MutateOIDCProvider no-op (zero options)
	if err := p.MutateOIDCProvider(ws, "oidc-1"); err != nil {
		t.Fatalf("MutateOIDCProvider no-op: %v", err)
	}

	// MutateOIDCProvider
	if err := p.MutateOIDCProvider(ws, "oidc-1",
		rubix.WithOIDCProviderName("Okta SSO"),
		rubix.WithOIDCDisplayName("Sign in with Okta"),
		rubix.WithOIDCClientSecret("new-secret"),
		rubix.WithOIDCBpoID("bpo-2"),
	); err != nil {
		t.Fatalf("MutateOIDCProvider: %v", err)
	}
	got, err = p.GetOIDCProvider(ws, "oidc-1")
	if err != nil {
		t.Fatalf("GetOIDCProvider after mutate: %v", err)
	}
	if got.ProviderName != "Okta SSO" || got.DisplayName != "Sign in with Okta" || got.ClientSecret != "new-secret" || got.BpoID != "bpo-2" {
		t.Fatalf("MutateOIDCProvider mismatch: %+v", got)
	}

	// MutateOIDCProvider not found
	if err := p.MutateOIDCProvider(ws, "nonexistent", rubix.WithOIDCProviderName("X")); err != rubix.ErrNoResultFound {
		t.Fatalf("expected ErrNoResultFound on mutate, got %v", err)
	}

	// DeleteOIDCProvider
	if err := p.DeleteOIDCProvider(ws, "oidc-2"); err != nil {
		t.Fatalf("DeleteOIDCProvider: %v", err)
	}
	providers, err = p.GetOIDCProviders(ws)
	if err != nil || len(providers) != 1 {
		t.Fatalf("GetOIDCProviders after delete: len=%d err=%v", len(providers), err)
	}
	if providers[0].Uuid != "oidc-1" {
		t.Fatalf("expected oidc-1 to remain, got %s", providers[0].Uuid)
	}
}
