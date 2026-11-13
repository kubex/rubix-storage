package sql

import (
	"strings"
	"time"

	"github.com/kubex/rubix-storage/rubix"
)

const vendorConnectionColumns = "`workspace`,`user`,`vendor_id`,`provider`,`external_id`,`external_name`,`access_token`,`refresh_token`,`token_type`,`scopes`,`expires_at`,`linked_by`,`created_at`,`updated_at`"

// GetVendorConnections returns the workspace links and, when user is set, that
// user's links.
func (p *Provider) GetVendorConnections(workspace, user string) ([]rubix.VendorConnection, error) {
	return p.queryVendorConnections("`workspace` = ? AND (`user` = '' OR `user` = ?)", workspace, user)
}

// ListVendorConnections returns every link, of every user, a workspace holds
// with a vendor.
func (p *Provider) ListVendorConnections(workspace, vendorID string) ([]rubix.VendorConnection, error) {
	return p.queryVendorConnections("`workspace` = ? AND `vendor_id` = ?", workspace, vendorID)
}

func (p *Provider) queryVendorConnections(where string, args ...any) ([]rubix.VendorConnection, error) {
	rows, err := p.query("SELECT "+vendorConnectionColumns+" FROM `vendor_connections` WHERE "+where+" ORDER BY `vendor_id`,`provider`,`user`", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []rubix.VendorConnection{}
	for rows.Next() {
		var c rubix.VendorConnection
		var scopes string
		var expires, created, updated int64
		if err := rows.Scan(&c.Workspace, &c.UserID, &c.VendorID, &c.Provider, &c.ExternalID, &c.ExternalName, &c.AccessToken, &c.RefreshToken, &c.TokenType, &scopes, &expires, &c.LinkedBy, &created, &updated); err != nil {
			return nil, err
		}
		c.Scopes = strings.Fields(scopes)
		if expires > 0 {
			c.ExpiresAt = time.Unix(expires, 0).UTC()
		}
		c.CreatedAt, c.UpdatedAt = time.Unix(created, 0).UTC(), time.Unix(updated, 0).UTC()
		out = append(out, c)
	}
	return out, rows.Err()
}

// SetVendorConnection creates or replaces a link. LinkedBy and CreatedAt are
// kept from the first write.
func (p *Provider) SetVendorConnection(c rubix.VendorConnection) error {
	now := time.Now().Unix()
	var expires int64
	if !c.ExpiresAt.IsZero() {
		expires = c.ExpiresAt.Unix()
	}
	query := "INSERT INTO `vendor_connections` (" + vendorConnectionColumns + ") VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)"
	if p.SqlLite {
		query += " ON CONFLICT(`workspace`,`user`,`vendor_id`,`provider`) DO UPDATE SET `external_id`=excluded.`external_id`,`external_name`=excluded.`external_name`,`access_token`=excluded.`access_token`,`refresh_token`=excluded.`refresh_token`,`token_type`=excluded.`token_type`,`scopes`=excluded.`scopes`,`expires_at`=excluded.`expires_at`,`updated_at`=excluded.`updated_at`"
	} else {
		query += " ON DUPLICATE KEY UPDATE `external_id`=VALUES(`external_id`),`external_name`=VALUES(`external_name`),`access_token`=VALUES(`access_token`),`refresh_token`=VALUES(`refresh_token`),`token_type`=VALUES(`token_type`),`scopes`=VALUES(`scopes`),`expires_at`=VALUES(`expires_at`),`updated_at`=VALUES(`updated_at`)"
	}
	_, err := p.primaryConnection.Exec(query, c.Workspace, c.UserID, c.VendorID, c.Provider, c.ExternalID, c.ExternalName, c.AccessToken, c.RefreshToken, c.TokenType, strings.Join(c.Scopes, " "), expires, c.LinkedBy, now, now)
	if err == nil {
		p.update()
	}
	return err
}

// RemoveVendorConnection removes one link; an empty user removes the workspace
// link and leaves user links in place.
func (p *Provider) RemoveVendorConnection(workspace, user, vendorID, provider string) error {
	_, err := p.primaryConnection.Exec("DELETE FROM `vendor_connections` WHERE `workspace` = ? AND `user` = ? AND `vendor_id` = ? AND `provider` = ?", workspace, user, vendorID, provider)
	if err == nil {
		p.update()
	}
	return err
}

// RemoveVendorConnections removes every link a workspace and its users hold
// with a vendor.
func (p *Provider) RemoveVendorConnections(workspace, vendorID string) error {
	_, err := p.primaryConnection.Exec("DELETE FROM `vendor_connections` WHERE `workspace` = ? AND `vendor_id` = ?", workspace, vendorID)
	if err == nil {
		p.update()
	}
	return err
}
