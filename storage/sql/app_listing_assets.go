package sql

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/kubex/rubix-storage/rubix"
)

func (p *Provider) StoreAppListingAsset(asset rubix.AppListingAsset) error {
	if err := asset.Validate(); err != nil {
		return err
	}
	query := "INSERT INTO `app_listing_assets` (`vendor_id`,`app_id`,`name`,`content_type`,`data`,`updated_at`) VALUES (?,?,?,?,?,?)"
	if p.SqlLite {
		query += " ON CONFLICT(`vendor_id`,`app_id`,`name`) DO UPDATE SET `content_type`=excluded.`content_type`,`data`=excluded.`data`,`updated_at`=excluded.`updated_at`"
	} else {
		query = strings.Replace(query, "INSERT INTO", "REPLACE INTO", 1)
	}
	_, err := p.primaryConnection.Exec(query, asset.VendorID, asset.AppID, asset.Name, asset.ContentType, asset.Data, time.Now().UTC().Unix())
	return err
}

func (p *Provider) GetAppListingAsset(vendorID, appID, name string) (*rubix.AppListingAsset, error) {
	asset := &rubix.AppListingAsset{VendorID: vendorID, AppID: appID, Name: name}
	var updated int64
	err := p.queryRow("SELECT `content_type`,`data`,`updated_at` FROM `app_listing_assets` WHERE `vendor_id`=? AND `app_id`=? AND `name`=?", vendorID, appID, name).
		Scan(&asset.ContentType, &asset.Data, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, rubix.ErrNoResultFound
	}
	if err != nil {
		return nil, err
	}
	asset.UpdatedAt = time.Unix(updated, 0).UTC()
	return asset, nil
}

func (p *Provider) PruneAppListingAssets(vendorID, appID string, keep []string) error {
	query := "DELETE FROM `app_listing_assets` WHERE `vendor_id`=? AND `app_id`=?"
	args := []any{vendorID, appID}
	if len(keep) > 0 {
		query += " AND `name` NOT IN (" + strings.TrimSuffix(strings.Repeat("?,", len(keep)), ",") + ")"
		for _, name := range keep {
			args = append(args, name)
		}
	}
	_, err := p.primaryConnection.Exec(query, args...)
	return err
}
