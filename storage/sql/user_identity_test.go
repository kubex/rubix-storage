package sql

import (
	"errors"
	"testing"

	"github.com/kubex/rubix-storage/rubix"
)

func TestUserIdentities(t *testing.T) {
	p := newTestProvider(t)
	defer func() { _ = p.Close() }()

	ws := "ws-ident"
	oidcUser := "oidc_8f14e45f-ceea-467e-9a3c-0123456789ab_00uabcdefghijklmnop123456789"
	link := func(eid, uid string, replace bool) error {
		return p.LinkUserIdentity(ws, rubix.UserIdentity{
			Vendor: "fortifi", Scope: "FID:ORG:1", ExternalID: eid, UserID: uid, Source: rubix.IdentitySourceApp,
		}, replace)
	}

	if err := link("FID:EMPL:9", oidcUser, false); err != nil {
		t.Fatalf("link: %v", err)
	}
	if err := link("FID:EMPL:9", oidcUser, false); err != nil {
		t.Fatalf("relink same user should be a no-op: %v", err)
	}
	if err := link("FID:EMPL:9", "u2", false); !errors.Is(err, rubix.ErrIdentityConflict) {
		t.Fatalf("expected ErrIdentityConflict, got %v", err)
	}
	if err := link("FID:EMPL:10", "u2", false); err != nil {
		t.Fatalf("link second: %v", err)
	}

	got, err := p.ResolveUserIdentities(ws, "fortifi", "FID:ORG:1", "FID:EMPL:9", "FID:EMPL:10", "FID:EMPL:404")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(got) != 2 || got["FID:EMPL:9"] != oidcUser || got["FID:EMPL:10"] != "u2" {
		t.Fatalf("unexpected resolve result: %v", got)
	}

	if got, _ := p.ResolveUserIdentities(ws, "fortifi", "FID:ORG:2", "FID:EMPL:9"); len(got) != 0 {
		t.Fatalf("scope must isolate identities, got %v", got)
	}
	if got, _ := p.ResolveUserIdentities("other-ws", "fortifi", "FID:ORG:1", "FID:EMPL:9"); len(got) != 0 {
		t.Fatalf("workspace must isolate identities, got %v", got)
	}

	if err := link("FID:EMPL:9", "u3", true); err != nil {
		t.Fatalf("replace: %v", err)
	}
	if got, _ := p.ResolveUserIdentities(ws, "fortifi", "FID:ORG:1", "FID:EMPL:9"); got["FID:EMPL:9"] != "u3" {
		t.Fatalf("replace not applied, got %v", got)
	}

	ids, err := p.GetUserIdentities(ws, "u2", "u3")
	if err != nil {
		t.Fatalf("GetUserIdentities: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 identities, got %+v", ids)
	}
	for _, id := range ids {
		if id.Vendor != "fortifi" || id.Scope != "FID:ORG:1" || id.Source != rubix.IdentitySourceApp || id.CreatedAt.IsZero() {
			t.Fatalf("unexpected identity: %+v", id)
		}
	}

	if err := p.UnlinkUserIdentity(ws, "fortifi", "FID:ORG:1", "FID:EMPL:10"); err != nil {
		t.Fatalf("unlink: %v", err)
	}
	if got, _ := p.ResolveUserIdentities(ws, "fortifi", "FID:ORG:1", "FID:EMPL:10"); len(got) != 0 {
		t.Fatalf("unlink not applied, got %v", got)
	}

	if err := p.AddUserToWorkspace(ws, "u3", rubix.MembershipTypeMember, ""); err != nil {
		t.Fatalf("AddUserToWorkspace: %v", err)
	}
	if err := p.RemoveUserFromWorkspace(ws, "u3"); err != nil {
		t.Fatalf("RemoveUserFromWorkspace: %v", err)
	}
	if got, _ := p.ResolveUserIdentities(ws, "fortifi", "FID:ORG:1", "FID:EMPL:9"); len(got) != 0 {
		t.Fatalf("removing a member must drop their identities, got %v", got)
	}
}
