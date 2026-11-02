package sql

import (
	"testing"
	"time"

	"github.com/kubex/rubix-storage/rubix"
)

func TestVendorMembers(t *testing.T) {
	p := newTestProvider(t)
	defer p.Close()

	for _, u := range []struct{ id, name, email string }{
		{"user-b", "Bravo", "bravo@example.com"},
		{"user-a", "Alpha", "Alpha@Example.com"},
	} {
		if err := p.CreateUser(u.id, u.name, u.email); err != nil {
			t.Fatal(err)
		}
	}
	if err := p.StorePlatformVendor(rubix.PlatformVendor{VendorID: "acme", Name: "Acme"}); err != nil {
		t.Fatal(err)
	}
	if err := p.StorePlatformVendor(rubix.PlatformVendor{VendorID: "beta", Name: "Beta"}); err != nil {
		t.Fatal(err)
	}

	if m, err := p.GetVendorMember("acme", "user-a"); err != nil || m != nil {
		t.Fatalf("GetVendorMember(absent) = %v, %v; want nil, nil", m, err)
	}

	before := time.Now().UTC().Add(-time.Minute)
	if err := p.StoreVendorMember(rubix.VendorMember{VendorID: "acme", UserID: "user-b", IsAdmin: true, AddedBy: "admin-1"}); err != nil {
		t.Fatal(err)
	}
	if err := p.StoreVendorMember(rubix.VendorMember{VendorID: "acme", UserID: "user-a", AddedBy: "admin-1"}); err != nil {
		t.Fatal(err)
	}
	if err := p.StoreVendorMember(rubix.VendorMember{VendorID: "acme", UserID: "ghost", AddedBy: "admin-1"}); err != nil {
		t.Fatal(err)
	}
	if err := p.StoreVendorMember(rubix.VendorMember{VendorID: "beta", UserID: "user-a", AddedBy: "admin-2"}); err != nil {
		t.Fatal(err)
	}

	m, err := p.GetVendorMember("acme", "user-b")
	if err != nil || m == nil {
		t.Fatalf("GetVendorMember() = %v, %v", m, err)
	}
	if !m.IsAdmin || m.AddedBy != "admin-1" || m.Name != "Bravo" || m.Email != "bravo@example.com" {
		t.Fatalf("unexpected member %+v", m)
	}
	if m.Since.Before(before) || m.Since.After(time.Now().UTC().Add(time.Minute)) {
		t.Fatalf("since = %v, want around now", m.Since)
	}
	originalSince := m.Since

	members, err := p.GetVendorMembers("acme")
	if err != nil {
		t.Fatal(err)
	}
	// Ordered by name (missing user row sorts first with an empty name), then email, then user.
	if len(members) != 3 || members[0].UserID != "ghost" || members[1].UserID != "user-a" || members[2].UserID != "user-b" {
		t.Fatalf("GetVendorMembers() = %+v", members)
	}
	if members[0].Name != "" || members[0].Email != "" {
		t.Fatalf("missing user row should give empty name/email, got %+v", members[0])
	}

	memberships, err := p.GetUserVendorMemberships("user-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(memberships) != 2 || memberships[0].VendorID != "acme" || memberships[1].VendorID != "beta" {
		t.Fatalf("GetUserVendorMemberships() = %+v", memberships)
	}

	// Upsert only changes is_admin.
	if err := p.StoreVendorMember(rubix.VendorMember{VendorID: "acme", UserID: "user-b", IsAdmin: false, AddedBy: "someone-else", Since: time.Now().Add(48 * time.Hour)}); err != nil {
		t.Fatal(err)
	}
	m, err = p.GetVendorMember("acme", "user-b")
	if err != nil || m == nil {
		t.Fatalf("GetVendorMember() = %v, %v", m, err)
	}
	if m.IsAdmin || m.AddedBy != "admin-1" || !m.Since.Equal(originalSince) {
		t.Fatalf("upsert changed more than is_admin: %+v (since was %v)", m, originalSince)
	}

	if err := p.RemoveVendorMember("acme", "ghost"); err != nil {
		t.Fatal(err)
	}
	if m, err := p.GetVendorMember("acme", "ghost"); err != nil || m != nil {
		t.Fatalf("removed member still present: %v, %v", m, err)
	}

	// Removing a vendor removes its members only.
	if err := p.RemovePlatformVendor("acme"); err != nil {
		t.Fatal(err)
	}
	if members, err := p.GetVendorMembers("acme"); err != nil || len(members) != 0 {
		t.Fatalf("members after vendor removal = %+v, %v", members, err)
	}
	if memberships, err := p.GetUserVendorMemberships("user-a"); err != nil || len(memberships) != 1 || memberships[0].VendorID != "beta" {
		t.Fatalf("other vendor memberships = %+v, %v", memberships, err)
	}
}

func TestGetUserByEmailAndID(t *testing.T) {
	p := newTestProvider(t)
	defer p.Close()

	if err := p.CreateUser("user-a", "Alpha", "Alpha@Example.com"); err != nil {
		t.Fatal(err)
	}

	for _, email := range []string{"alpha@example.com", "  ALPHA@EXAMPLE.COM ", "Alpha@Example.com"} {
		u, err := p.GetUserByEmail(email)
		if err != nil || u == nil || u.UserID != "user-a" || u.Name != "Alpha" {
			t.Fatalf("GetUserByEmail(%q) = %+v, %v", email, u, err)
		}
	}
	if u, err := p.GetUserByEmail("nobody@example.com"); err != nil || u != nil {
		t.Fatalf("GetUserByEmail(unknown) = %+v, %v; want nil, nil", u, err)
	}
	if u, err := p.GetUserByEmail(" "); err != nil || u != nil {
		t.Fatalf("GetUserByEmail(blank) = %+v, %v; want nil, nil", u, err)
	}

	if u, err := p.GetUserByID("user-a"); err != nil || u == nil || u.Email != "Alpha@Example.com" {
		t.Fatalf("GetUserByID() = %+v, %v", u, err)
	}
	if u, err := p.GetUserByID("missing"); err != nil || u != nil {
		t.Fatalf("GetUserByID(missing) = %+v, %v; want nil, nil", u, err)
	}
}

func TestVendorMemberOwner(t *testing.T) {
	p := newTestProvider(t)
	defer p.Close()

	// Owner implies admin even when IsAdmin is not set.
	if err := p.StoreVendorMember(rubix.VendorMember{VendorID: "acme", UserID: "owner", IsOwner: true, AddedBy: "admin-1"}); err != nil {
		t.Fatal(err)
	}
	m, err := p.GetVendorMember("acme", "owner")
	if err != nil || m == nil || !m.IsOwner || !m.IsAdmin {
		t.Fatalf("owner = %+v, %v; want owner and admin", m, err)
	}
	since := m.Since

	// Demote to admin: is_owner updates, added_by and since are kept.
	if err := p.StoreVendorMember(rubix.VendorMember{VendorID: "acme", UserID: "owner", IsAdmin: true, AddedBy: "other"}); err != nil {
		t.Fatal(err)
	}
	m, _ = p.GetVendorMember("acme", "owner")
	if m.IsOwner || !m.IsAdmin || m.AddedBy != "admin-1" || !m.Since.Equal(since) {
		t.Fatalf("after demote to admin = %+v", m)
	}

	// Demote to member.
	if err := p.StoreVendorMember(rubix.VendorMember{VendorID: "acme", UserID: "owner"}); err != nil {
		t.Fatal(err)
	}
	m, _ = p.GetVendorMember("acme", "owner")
	if m.IsOwner || m.IsAdmin {
		t.Fatalf("after demote to member = %+v", m)
	}

	// Promote straight to owner through the upsert.
	if err := p.StoreVendorMember(rubix.VendorMember{VendorID: "acme", UserID: "owner", IsOwner: true}); err != nil {
		t.Fatal(err)
	}
	members, err := p.GetVendorMembers("acme")
	if err != nil || len(members) != 1 || !members[0].IsOwner || !members[0].IsAdmin || members[0].AddedBy != "admin-1" {
		t.Fatalf("members after promote = %+v, %v", members, err)
	}
	memberships, err := p.GetUserVendorMemberships("owner")
	if err != nil || len(memberships) != 1 || !memberships[0].IsOwner {
		t.Fatalf("memberships = %+v, %v", memberships, err)
	}
}

func TestPlatformVendorSystemVendor(t *testing.T) {
	p := newTestProvider(t)
	defer p.Close()

	find := func(id string) rubix.PlatformVendor {
		t.Helper()
		vendors, err := p.GetPlatformVendors()
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range vendors {
			if v.VendorID == id {
				return v
			}
		}
		t.Fatalf("vendor %s not found", id)
		return rubix.PlatformVendor{}
	}

	if err := p.StorePlatformVendor(rubix.PlatformVendor{VendorID: "acme", Name: "Acme", SystemVendor: true}); err != nil {
		t.Fatal(err)
	}
	if err := p.StorePlatformVendor(rubix.PlatformVendor{VendorID: "beta", Name: "Beta"}); err != nil {
		t.Fatal(err)
	}
	if v := find("acme"); !v.SystemVendor || v.Name != "Acme" {
		t.Fatalf("acme = %+v", v)
	}
	if v := find("beta"); v.SystemVendor {
		t.Fatalf("beta = %+v", v)
	}

	// The upsert writes every column, so SystemVendor follows the stored value.
	if err := p.StorePlatformVendor(rubix.PlatformVendor{VendorID: "acme", Name: "Acme Ltd", SystemVendor: true}); err != nil {
		t.Fatal(err)
	}
	if v := find("acme"); !v.SystemVendor || v.Name != "Acme Ltd" {
		t.Fatalf("acme after rename = %+v", v)
	}
	if err := p.StorePlatformVendor(rubix.PlatformVendor{VendorID: "acme", Name: "Acme Ltd"}); err != nil {
		t.Fatal(err)
	}
	if v := find("acme"); v.SystemVendor {
		t.Fatalf("acme after clearing = %+v", v)
	}
}
