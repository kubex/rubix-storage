package sql

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
	"time"

	"github.com/kubex/rubix-storage/rubix"
)

func (p *Provider) queryGatewaySourceIPs(where string, args ...any) ([]rubix.GatewaySourceIP, error) {
	rows, err := p.primaryConnection.Query("SELECT `id`,`scope`,`vendor_id`,`app_id`,`network`,`description`,`created_by`,`created_at` FROM `gateway_source_ips` WHERE "+where+" ORDER BY `scope`,`network`,`id`", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []rubix.GatewaySourceIP{}
	for rows.Next() {
		var entry rubix.GatewaySourceIP
		var created int64
		if err := rows.Scan(&entry.ID, &entry.Scope, &entry.VendorID, &entry.AppID, &entry.Network, &entry.Description, &entry.CreatedBy, &created); err != nil {
			return nil, err
		}
		entry.CreatedAt = credentialDate(created)
		out = append(out, entry)
	}
	return out, rows.Err()
}
func (p *Provider) GetGatewaySourceIPs(scope rubix.GatewaySourceScope, vendor, application string) ([]rubix.GatewaySourceIP, error) {
	if !rubix.ValidGatewaySourceScope(scope, vendor, application) {
		return nil, rubix.ErrGatewaySourceIP
	}
	return p.queryGatewaySourceIPs("`scope`=? AND `vendor_id`=? AND `app_id`=?", scope, vendor, application)
}

// GetEffectiveGatewaySourceIPs returns the union of global, vendor, and app
// entries. No entry belonging only to a different vendor/app can participate.
func (p *Provider) GetEffectiveGatewaySourceIPs(vendor, application string) ([]rubix.GatewaySourceIP, error) {
	if !rubix.ValidGatewaySourceScope(rubix.GatewaySourceApp, vendor, application) {
		return nil, rubix.ErrGatewaySourceIP
	}
	return p.queryGatewaySourceIPs("(`scope`='global' AND `vendor_id`='' AND `app_id`='') OR (`scope`='vendor' AND `vendor_id`=? AND `app_id`='') OR (`scope`='app' AND `vendor_id`=? AND `app_id`=?)", vendor, vendor, application)
}
func (p *Provider) AddGatewaySourceIP(entry rubix.GatewaySourceIP) (rubix.GatewaySourceIP, error) {
	if !rubix.ValidGatewaySourceScope(entry.Scope, entry.VendorID, entry.AppID) || entry.CreatedBy == "" || len(entry.CreatedBy) > 255 || len(entry.Description) > 255 {
		return rubix.GatewaySourceIP{}, rubix.ErrGatewaySourceIP
	}
	network, err := rubix.NormalizeIPNetwork(entry.Network)
	if err != nil {
		return rubix.GatewaySourceIP{}, err
	}
	id := make([]byte, 16)
	if _, err = rand.Read(id); err != nil {
		return rubix.GatewaySourceIP{}, err
	}
	entry.ID = base64.RawURLEncoding.EncodeToString(id)
	entry.Network = network
	entry.Description = strings.TrimSpace(entry.Description)
	entry.CreatedAt = time.Now().UTC().Truncate(time.Second)
	_, err = p.primaryConnection.Exec("INSERT INTO `gateway_source_ips` (`id`,`scope`,`vendor_id`,`app_id`,`network`,`description`,`created_by`,`created_at`) VALUES (?,?,?,?,?,?,?,?)", entry.ID, entry.Scope, entry.VendorID, entry.AppID, entry.Network, entry.Description, entry.CreatedBy, entry.CreatedAt.Unix())
	if err != nil {
		return rubix.GatewaySourceIP{}, err
	}
	return entry, nil
}
func (p *Provider) DeleteGatewaySourceIP(scope rubix.GatewaySourceScope, vendor, application, id string) error {
	if !rubix.ValidGatewaySourceScope(scope, vendor, application) {
		return rubix.ErrGatewaySourceIP
	}
	result, err := p.primaryConnection.Exec("DELETE FROM `gateway_source_ips` WHERE `id`=? AND `scope`=? AND `vendor_id`=? AND `app_id`=?", id, scope, vendor, application)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return rubix.ErrGatewaySourceIPNotFound
	}
	return nil
}
