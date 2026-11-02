package sql

import (
	"testing"

	"github.com/kubex/rubix-storage/rubix"
)

func TestPlatformApplicationStatus(t *testing.T) {
	p := newTestProvider(t)
	defer p.Close()

	find := func(appID string) rubix.PlatformApplication {
		t.Helper()
		apps, err := p.GetPlatformApplications()
		if err != nil {
			t.Fatal(err)
		}
		for _, a := range apps {
			if a.AppID == appID {
				return a
			}
		}
		t.Fatalf("app %s not found", appID)
		return rubix.PlatformApplication{}
	}

	if err := p.StorePlatformApplication(rubix.PlatformApplication{VendorID: "acme", AppID: "plain", Endpoint: "https://plain"}); err != nil {
		t.Fatal(err)
	}
	plain := find("plain")
	if plain.Status != "" {
		t.Fatalf("default status = %q, want empty", plain.Status)
	}
	if status, ok := rubix.ParseAppStatus(string(plain.Status)); !ok || status != rubix.AppStatusActive || plain.Status.Blocked() {
		t.Fatalf("empty status should parse as active, got %q %v", status, ok)
	}

	for _, status := range []rubix.AppStatus{rubix.AppStatusPreRelease, rubix.AppStatusSuspended, rubix.AppStatusArchived, rubix.AppStatusActive} {
		if err := p.StorePlatformApplication(rubix.PlatformApplication{VendorID: "acme", AppID: "billing", Endpoint: "https://billing", Status: status}); err != nil {
			t.Fatal(err)
		}
		if got := find("billing").Status; got != status {
			t.Fatalf("status = %q, want %q", got, status)
		}
	}
}
