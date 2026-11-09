package sql

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/kubex/rubix-storage/rubix"
)

// RecordAppVersion notes that the version is running now: a new version is
// added with FirstSeen and LastSeen set to now, a known one has LastSeen,
// Commit and BuildTime refreshed.
func (p *Provider) RecordAppVersion(v rubix.AppVersion) error {
	if v.VendorID == "" || v.AppID == "" || !rubix.ValidAppVersion(v.Version) {
		return rubix.ErrAppRelease
	}
	now := time.Now().UTC().Unix()
	query := "INSERT INTO `app_versions` (`vendor_id`,`app_id`,`release_channel`,`version`,`commit_hash`,`build_time`,`first_seen`,`last_seen`) VALUES (?,?,?,?,?,?,?,?)"
	if p.SqlLite {
		query += " ON CONFLICT(`vendor_id`,`app_id`,`release_channel`,`version`) DO UPDATE SET `commit_hash`=excluded.`commit_hash`,`build_time`=excluded.`build_time`,`last_seen`=excluded.`last_seen`"
	} else {
		query += " ON DUPLICATE KEY UPDATE `commit_hash`=VALUES(`commit_hash`),`build_time`=VALUES(`build_time`),`last_seen`=VALUES(`last_seen`)"
	}
	_, err := p.primaryConnection.Exec(query, v.VendorID, v.AppID, v.ReleaseChannel, v.Version, v.Commit, v.BuildTime, now, now)
	return err
}

// GetAppVersions lists the versions seen for an app on a release channel,
// newest arrival first.
func (p *Provider) GetAppVersions(vendorID, appID, releaseChannel string) ([]rubix.AppVersion, error) {
	return p.queryAppVersions("`vendor_id`=? AND `app_id`=? AND `release_channel`=?", vendorID, appID, releaseChannel)
}

// GetLiveAppVersions is each app's live version on a release channel, keyed
// by "vendor/app".
func (p *Provider) GetLiveAppVersions(releaseChannel string) (map[string]rubix.AppVersion, error) {
	all, err := p.queryAppVersions("`release_channel`=?", releaseChannel)
	if err != nil {
		return nil, err
	}
	byApp := map[string][]rubix.AppVersion{}
	for _, v := range all {
		key := v.VendorID + "/" + v.AppID
		byApp[key] = append(byApp[key], v)
	}
	live := map[string]rubix.AppVersion{}
	for key, versions := range byApp {
		if v, ok := rubix.LiveAppVersion(versions); ok {
			live[key] = v
		}
	}
	return live, nil
}

func (p *Provider) queryAppVersions(where string, args ...any) ([]rubix.AppVersion, error) {
	rows, err := p.primaryConnection.Query("SELECT `vendor_id`,`app_id`,`release_channel`,`version`,`commit_hash`,`build_time`,`first_seen`,`last_seen` FROM `app_versions` WHERE "+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []rubix.AppVersion{}
	for rows.Next() {
		var v rubix.AppVersion
		var first, last int64
		if err := rows.Scan(&v.VendorID, &v.AppID, &v.ReleaseChannel, &v.Version, &v.Commit, &v.BuildTime, &first, &last); err != nil {
			return nil, err
		}
		v.FirstSeen, v.LastSeen = time.Unix(first, 0).UTC(), time.Unix(last, 0).UTC()
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rubix.SortAppVersions(out)
	return out, nil
}

// GetAppReleases lists an app's release notes, seen versions and queued ones alike.
func (p *Provider) GetAppReleases(vendorID, appID string) ([]rubix.AppRelease, error) {
	rows, err := p.primaryConnection.Query("SELECT `version`,`notes`,`updated_by`,`updated_at` FROM `app_releases` WHERE `vendor_id`=? AND `app_id`=? ORDER BY `updated_at` DESC", vendorID, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []rubix.AppRelease{}
	for rows.Next() {
		r := rubix.AppRelease{VendorID: vendorID, AppID: appID}
		var notes string
		var updated int64
		if err := rows.Scan(&r.Version, &notes, &r.UpdatedBy, &updated); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(notes), &r.Notes); err != nil {
			return nil, err
		}
		if updated > 0 {
			r.UpdatedAt = time.Unix(updated, 0).UTC()
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// StoreAppRelease saves the version's notes; no notes removes them.
func (p *Provider) StoreAppRelease(r rubix.AppRelease) error {
	if len(r.Notes) == 0 {
		if r.VendorID == "" || r.AppID == "" || !rubix.ValidAppVersion(r.Version) {
			return rubix.ErrAppRelease
		}
		_, err := p.primaryConnection.Exec("DELETE FROM `app_releases` WHERE `vendor_id`=? AND `app_id`=? AND `version`=?", r.VendorID, r.AppID, r.Version)
		return err
	}
	if err := r.Validate(); err != nil {
		return err
	}
	notes, err := json.Marshal(r.Notes)
	if err != nil {
		return err
	}
	query := "INSERT INTO `app_releases` (`vendor_id`,`app_id`,`version`,`notes`,`updated_by`,`updated_at`) VALUES (?,?,?,?,?,?)"
	if p.SqlLite {
		query += " ON CONFLICT(`vendor_id`,`app_id`,`version`) DO UPDATE SET `notes`=excluded.`notes`,`updated_by`=excluded.`updated_by`,`updated_at`=excluded.`updated_at`"
	} else {
		query = strings.Replace(query, "INSERT INTO", "REPLACE INTO", 1)
	}
	_, err = p.primaryConnection.Exec(query, r.VendorID, r.AppID, r.Version, string(notes), r.UpdatedBy, time.Now().UTC().Unix())
	return err
}
