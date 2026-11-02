package sql

import (
	"errors"
	"testing"

	"github.com/kubex/rubix-storage/rubix"
	"github.com/openbyte-os/sdk-go/app"
)

func TestRetrieveAllWorkspaces(t *testing.T) {
	p := newTestProvider(t)
	defer p.Close()
	for _, id := range []string{"scheduled-one", "scheduled-two"} {
		if err := p.CreateWorkspace(id, id, id, id+".local"); err != nil {
			t.Fatal(err)
		}
	}
	if err := p.SetWorkspaceApplication("scheduled-one", "vendor", "app", "beta"); err != nil {
		t.Fatal(err)
	}
	workspaces, err := p.RetrieveAllWorkspaces()
	if err != nil {
		t.Fatal(err)
	}
	if workspaces["scheduled-one"] == nil || workspaces["scheduled-two"] == nil {
		t.Fatalf("missing workspaces: %v", workspaces)
	}
	installed := workspaces["scheduled-one"].InstalledApplications
	id := app.NewID("vendor", "app")
	if len(installed) != 1 || installed[0] != app.NewScopedKey("beta", &id) {
		t.Fatalf("incorrect installations: %v", installed)
	}
}

func TestRetrieveWorkspace_Missing(t *testing.T) {
	p := newTestProvider(t)
	defer p.Close()
	if err := p.CreateWorkspace("present", "present", "present", "present.local"); err != nil {
		t.Fatal(err)
	}

	if ws, err := p.RetrieveWorkspace("absent"); ws != nil || !errors.Is(err, rubix.ErrNoResultFound) {
		t.Fatalf("RetrieveWorkspace: expected (nil, ErrNoResultFound), got (%v, %v)", ws, err)
	}
	if ws, err := p.RetrieveWorkspaceByDomain("absent.local"); ws != nil || !errors.Is(err, rubix.ErrNoResultFound) {
		t.Fatalf("RetrieveWorkspaceByDomain: expected (nil, ErrNoResultFound), got (%v, %v)", ws, err)
	}
	if ws, err := p.RetrieveWorkspace("present"); err != nil || ws == nil {
		t.Fatalf("RetrieveWorkspace(present): got (%v, %v)", ws, err)
	}
}
