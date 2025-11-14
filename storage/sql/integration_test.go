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
}
