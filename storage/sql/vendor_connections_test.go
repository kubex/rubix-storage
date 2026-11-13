package sql

import (
	"testing"
	"time"

	"github.com/kubex/rubix-storage/rubix"
)

func TestVendorConnections(t *testing.T) {
	p := newTestProvider(t)
	defer p.Close()

	expires := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	workspace := rubix.VendorConnection{Workspace: "ws", VendorID: "fortifi", Provider: "fortifi", ExternalID: "ORG-1", ExternalName: "Acme", AccessToken: "at", RefreshToken: "rt", TokenType: "Bearer", Scopes: []string{"org", "billing.read"}, ExpiresAt: expires, LinkedBy: "admin"}
	alice := rubix.VendorConnection{Workspace: "ws", UserID: "alice", VendorID: "fortifi", Provider: "fortifi", ExternalID: "EMP-1", AccessToken: "at-a", LinkedBy: "alice"}
	bob := rubix.VendorConnection{Workspace: "ws", UserID: "bob", VendorID: "fortifi", Provider: "fortifi", ExternalID: "EMP-2", AccessToken: "at-b", LinkedBy: "bob"}
	other := rubix.VendorConnection{Workspace: "ws", VendorID: "ch", Provider: "chargehive", ExternalID: "proj", AccessToken: "at-c", LinkedBy: "admin"}
	for _, c := range []rubix.VendorConnection{workspace, alice, bob, other} {
		if err := p.SetVendorConnection(c); err != nil {
			t.Fatal(err)
		}
	}

	got, err := p.GetVendorConnections("ws", "alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("alice sees %d connections, want the two workspace links and her own", len(got))
	}
	var ws, user *rubix.VendorConnection
	for i := range got {
		switch {
		case got[i].UserID == "bob":
			t.Fatal("alice must not see bob's link")
		case got[i].VendorID == "fortifi" && got[i].UserID == "":
			ws = &got[i]
		case got[i].UserID == "alice":
			user = &got[i]
		}
	}
	if ws == nil || ws.ExternalID != "ORG-1" || ws.ExternalName != "Acme" || ws.AccessToken != "at" || ws.RefreshToken != "rt" || len(ws.Scopes) != 2 || !ws.ExpiresAt.Equal(expires) || ws.CreatedAt.IsZero() {
		t.Fatalf("workspace link = %+v", ws)
	}
	if user == nil || user.ExternalID != "EMP-1" || !user.ExpiresAt.IsZero() || len(user.Scopes) != 0 {
		t.Fatalf("user link = %+v", user)
	}

	if all, err := p.ListVendorConnections("ws", "fortifi"); err != nil || len(all) != 3 {
		t.Fatalf("ListVendorConnections = %d rows, %v; want the workspace link and both users", len(all), err)
	}

	workspace.ExternalID, workspace.AccessToken, workspace.LinkedBy = "ORG-2", "at2", "someone-else"
	if err := p.SetVendorConnection(workspace); err != nil {
		t.Fatal(err)
	}
	got, _ = p.GetVendorConnections("ws", "")
	if len(got) != 2 {
		t.Fatalf("workspace-only read returned %d rows, want 2", len(got))
	}
	for _, c := range got {
		if c.VendorID == "fortifi" && (c.ExternalID != "ORG-2" || c.AccessToken != "at2" || c.LinkedBy != "admin") {
			t.Fatalf("replaced link = %+v; want new account and token, original linker", c)
		}
	}

	if err := p.RemoveVendorConnection("ws", "", "fortifi", "fortifi"); err != nil {
		t.Fatal(err)
	}
	if got, _ = p.GetVendorConnections("ws", "alice"); len(got) != 2 {
		t.Fatalf("removing the workspace link left %d rows for alice, want her link and ch", len(got))
	}
	if err := p.RemoveVendorConnections("ws", "fortifi"); err != nil {
		t.Fatal(err)
	}
	if got, _ = p.GetVendorConnections("ws", "bob"); len(got) != 1 || got[0].VendorID != "ch" {
		t.Fatalf("after removing fortifi: %+v", got)
	}
}
