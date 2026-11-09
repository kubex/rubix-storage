package sql

import (
	"errors"
	"testing"
	"time"

	"github.com/kubex/rubix-storage/rubix"
)

func TestAppVersions(t *testing.T) {
	p := newTestProvider(t)
	defer p.Close()

	record := func(version, commit, channel string) {
		t.Helper()
		if err := p.RecordAppVersion(rubix.AppVersion{VendorID: "acme", AppID: "crm", ReleaseChannel: channel, Version: version, Commit: commit}); err != nil {
			t.Fatal(err)
		}
	}
	record("1.0.0", "aaa", "")
	first, _ := p.GetAppVersions("acme", "crm", "")
	time.Sleep(1100 * time.Millisecond)
	record("1.0.0", "aab", "")
	record("2.0.0-beta", "bbb", "beta")

	versions, err := p.GetAppVersions("acme", "crm", "")
	if err != nil || len(versions) != 1 {
		t.Fatalf("default channel versions: %+v %v", versions, err)
	}
	if v := versions[0]; v.Commit != "aab" || !v.FirstSeen.Equal(first[0].FirstSeen) || !v.LastSeen.After(v.FirstSeen) {
		t.Fatalf("re-recording keeps first seen and refreshes the rest: %+v", v)
	}

	time.Sleep(1100 * time.Millisecond)
	record("1.1.0", "ccc", "")
	live, err := p.GetLiveAppVersions("")
	if err != nil || live["acme/crm"].Version != "1.1.0" {
		t.Fatalf("live = %+v %v", live, err)
	}

	// A rollback makes the older version live again.
	time.Sleep(1100 * time.Millisecond)
	record("1.0.0", "aab", "")
	if live, _ = p.GetLiveAppVersions(""); live["acme/crm"].Version != "1.0.0" {
		t.Fatalf("rolled back live = %+v", live["acme/crm"])
	}

	if err := p.RecordAppVersion(rubix.AppVersion{VendorID: "acme", AppID: "crm", Version: "1 0"}); !errors.Is(err, rubix.ErrAppRelease) {
		t.Fatal("version with a space accepted", err)
	}
}

func TestAppReleases(t *testing.T) {
	p := newTestProvider(t)
	defer p.Close()

	for _, r := range []rubix.AppRelease{
		{VendorID: "acme", AppID: "crm", Version: "1.1.0", Notes: []string{"Faster search"}, UpdatedBy: "u1"},
		{VendorID: "acme", AppID: "crm", Version: "1.2.0", Notes: []string{"Queued feature"}, UpdatedBy: "u1"},
	} {
		if err := p.StoreAppRelease(r); err != nil {
			t.Fatal(err)
		}
	}
	if err := p.StoreAppRelease(rubix.AppRelease{VendorID: "acme", AppID: "crm", Version: "1.1.0", Notes: []string{"Faster search", "Bulk edit"}}); err != nil {
		t.Fatal("update:", err)
	}
	releases, err := p.GetAppReleases("acme", "crm")
	if err != nil || len(releases) != 2 {
		t.Fatalf("releases: %+v %v", releases, err)
	}
	for _, r := range releases {
		if r.Version == "1.1.0" && len(r.Notes) != 2 {
			t.Fatalf("updated notes: %+v", r)
		}
	}

	if err := p.StoreAppRelease(rubix.AppRelease{VendorID: "acme", AppID: "crm", Version: "1.2.0"}); err != nil {
		t.Fatal("delete:", err)
	}
	if releases, _ = p.GetAppReleases("acme", "crm"); len(releases) != 1 {
		t.Fatalf("empty notes should remove the release: %+v", releases)
	}

	if err := p.StoreAppRelease(rubix.AppRelease{VendorID: "acme", AppID: "crm", Version: "2.0", Notes: []string{" "}}); !errors.Is(err, rubix.ErrAppRelease) {
		t.Fatal("blank note accepted", err)
	}
}
