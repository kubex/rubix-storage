package sql

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/kubex/rubix-storage/rubix"
)

// App listings are stored as one JSON document per app, so listing fields can
// grow without schema changes.

func (p *Provider) GetAppListing(vendorID, appID string) (*rubix.AppListing, error) {
	row := p.queryRow("SELECT `data`,`updated_by`,`updated_at` FROM `app_listings` WHERE `vendor_id`=? AND `app_id`=?", vendorID, appID)
	listing, err := scanAppListing(row.Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, rubix.ErrNoResultFound
	}
	if err != nil {
		return nil, err
	}
	listing.VendorID, listing.AppID = vendorID, appID
	return listing, nil
}

func (p *Provider) GetAppListings() ([]rubix.AppListing, error) {
	rows, err := p.query("SELECT `vendor_id`,`app_id`,`data`,`updated_by`,`updated_at` FROM `app_listings`")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []rubix.AppListing{}
	for rows.Next() {
		var vendorID, appID string
		listing, err := scanAppListing(func(dest ...any) error {
			return rows.Scan(append([]any{&vendorID, &appID}, dest...)...)
		})
		if err != nil {
			return nil, err
		}
		listing.VendorID, listing.AppID = vendorID, appID
		out = append(out, *listing)
	}
	return out, rows.Err()
}

func (p *Provider) StoreAppListing(listing rubix.AppListing) error {
	if err := listing.Validate(); err != nil {
		return err
	}
	listing.UpdatedAt = time.Now().UTC().Truncate(time.Second)
	data, err := json.Marshal(listing)
	if err != nil {
		return err
	}
	query := "INSERT INTO `app_listings` (`vendor_id`,`app_id`,`data`,`updated_by`,`updated_at`) VALUES (?,?,?,?,?)"
	if p.SqlLite {
		query += " ON CONFLICT(`vendor_id`,`app_id`) DO UPDATE SET `data`=excluded.`data`,`updated_by`=excluded.`updated_by`,`updated_at`=excluded.`updated_at`"
	} else {
		query = strings.Replace(query, "INSERT INTO", "REPLACE INTO", 1)
	}
	if _, err = p.primaryConnection.Exec(query, listing.VendorID, listing.AppID, string(data), listing.UpdatedBy, listing.UpdatedAt.Unix()); err != nil {
		return err
	}
	p.update()
	return nil
}

// GetAppInstallCounts is the number of workspaces each app is installed in,
// keyed by "vendor/app".
func (p *Provider) GetAppInstallCounts() (map[string]int, error) {
	rows, err := p.query("SELECT `vendor_id`,`app_id`,COUNT(DISTINCT `workspace_uuid`) FROM `workspace_applications` GROUP BY `vendor_id`,`app_id`")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var vendorID, appID string
		var count int
		if err := rows.Scan(&vendorID, &appID, &count); err != nil {
			return nil, err
		}
		counts[vendorID+"/"+appID] = count
	}
	return counts, rows.Err()
}

func scanAppListing(scan func(dest ...any) error) (*rubix.AppListing, error) {
	var data, updatedBy string
	var updatedAt int64
	if err := scan(&data, &updatedBy, &updatedAt); err != nil {
		return nil, err
	}
	listing := &rubix.AppListing{}
	if err := json.Unmarshal([]byte(data), listing); err != nil {
		return nil, err
	}
	listing.UpdatedBy = updatedBy
	if updatedAt > 0 {
		listing.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	}
	return listing, nil
}
