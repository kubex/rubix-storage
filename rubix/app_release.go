package rubix

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// AppVersion is a version of an app that rubix has seen running, as reported
// by the app's /_kubex/version during health checks.
type AppVersion struct {
	VendorID       string    `json:"vendorID"`
	AppID          string    `json:"appID"`
	ReleaseChannel string    `json:"releaseChannel"`
	Version        string    `json:"version"`
	Commit         string    `json:"commit,omitempty"`
	BuildTime      string    `json:"buildTime,omitempty"`
	FirstSeen      time.Time `json:"firstSeen"`
	LastSeen       time.Time `json:"lastSeen"`
}

// AppRelease is a vendor's release notes for one version. Notes may be written
// before the version is seen running; the marketplace shows them only once it is.
type AppRelease struct {
	VendorID  string    `json:"vendorID"`
	AppID     string    `json:"appID"`
	Version   string    `json:"version"`
	Notes     []string  `json:"notes"`
	UpdatedBy string    `json:"updatedBy,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitzero"`
}

const (
	AppVersionMaxLength = 64
	AppReleaseMaxNotes  = 20
	appReleaseMaxNote   = 300
)

var ErrAppRelease = errors.New("invalid release notes")

// ValidAppVersion is a non-empty version label without whitespace.
func ValidAppVersion(v string) bool {
	return v != "" && len(v) <= AppVersionMaxLength && !strings.ContainsAny(v, " \t\r\n")
}

func (r AppRelease) Validate() error {
	fail := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", ErrAppRelease, fmt.Sprintf(format, args...))
	}
	switch {
	case r.VendorID == "" || r.AppID == "":
		return fail("vendor and app are required")
	case !ValidAppVersion(r.Version):
		return fail("version must be up to %d characters with no spaces", AppVersionMaxLength)
	case len(r.Notes) > AppReleaseMaxNotes:
		return fail("more than %d notes", AppReleaseMaxNotes)
	}
	for _, n := range r.Notes {
		if strings.TrimSpace(n) == "" || len(n) > appReleaseMaxNote {
			return fail("each note is a line of up to %d characters", appReleaseMaxNote)
		}
	}
	return nil
}

// LiveAppVersion is the version most recently seen running, preferring the
// newer arrival when two were last seen together. ok is false when none was seen.
func LiveAppVersion(versions []AppVersion) (live AppVersion, ok bool) {
	for _, v := range versions {
		if !ok || v.LastSeen.After(live.LastSeen) || (v.LastSeen.Equal(live.LastSeen) && v.FirstSeen.After(live.FirstSeen)) {
			live, ok = v, true
		}
	}
	return live, ok
}

// SortAppVersions orders versions newest arrival first.
func SortAppVersions(versions []AppVersion) {
	sort.SliceStable(versions, func(i, j int) bool { return versions[i].FirstSeen.After(versions[j].FirstSeen) })
}
