package sql

import (
	"encoding/json"
	"strings"

	"github.com/kubex/rubix-storage/rubix"
)

func (p *Provider) GetPlatformApplications() ([]rubix.PlatformApplication, error) {
	rows, err := p.primaryConnection.Query("SELECT vendor_id, app_id, release_channel, signature_key, endpoint, simple_app, framed, allow_scripts, cookie_passthrough, globally_available, allowed_workspaces, workspace_available, api_endpoint, mcp_endpoint, discovered FROM platform_applications")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []rubix.PlatformApplication
	for rows.Next() {
		var a rubix.PlatformApplication
		var cookieJSON string
		var allowedJSON string
		if err := rows.Scan(&a.VendorID, &a.AppID, &a.ReleaseChannel, &a.SignatureKey, &a.Endpoint, &a.SimpleApp, &a.Framed, &a.AllowScripts, &cookieJSON, &a.GloballyAvailable, &allowedJSON, &a.WorkspaceAvailable, &a.ApiEndpoint, &a.McpEndpoint, &a.Discovered); err != nil {
			return nil, err
		}
		if cookieJSON != "" {
			_ = json.Unmarshal([]byte(cookieJSON), &a.CookiePassthrough)
		}
		if allowedJSON != "" {
			_ = json.Unmarshal([]byte(allowedJSON), &a.AllowedWorkspaces)
		}
		apps = append(apps, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return apps, nil
}

func (p *Provider) StorePlatformApplication(application rubix.PlatformApplication) error {
	cookieJSON := ""
	if len(application.CookiePassthrough) > 0 {
		if b, err := json.Marshal(application.CookiePassthrough); err == nil {
			cookieJSON = string(b)
		}
	}

	allowedJSON := ""
	if len(application.AllowedWorkspaces) > 0 {
		if b, err := json.Marshal(application.AllowedWorkspaces); err == nil {
			allowedJSON = string(b)
		}
	}

	query := "INSERT INTO platform_applications (vendor_id, app_id, release_channel, signature_key, endpoint, simple_app, framed, allow_scripts, cookie_passthrough, globally_available, allowed_workspaces, workspace_available, api_endpoint, mcp_endpoint, discovered) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	if p.SqlLite {
		query += " ON CONFLICT(vendor_id, app_id, release_channel) DO UPDATE SET signature_key=excluded.signature_key, endpoint=excluded.endpoint, simple_app=excluded.simple_app, framed=excluded.framed, allow_scripts=excluded.allow_scripts, cookie_passthrough=excluded.cookie_passthrough, globally_available=excluded.globally_available, allowed_workspaces=excluded.allowed_workspaces, workspace_available=excluded.workspace_available, api_endpoint=excluded.api_endpoint, mcp_endpoint=excluded.mcp_endpoint, discovered=excluded.discovered"
	} else {
		query = strings.Replace(query, "INSERT INTO", "REPLACE INTO", 1)
	}

	_, err := p.primaryConnection.Exec(query, application.VendorID, application.AppID, application.ReleaseChannel, application.SignatureKey, application.Endpoint, application.SimpleApp, application.Framed, application.AllowScripts, cookieJSON, application.GloballyAvailable, allowedJSON, application.WorkspaceAvailable, application.ApiEndpoint, application.McpEndpoint, application.Discovered)
	if err == nil {
		p.update()
	}
	return err
}

func (p *Provider) RemovePlatformApplication(vendorID, appID, releaseChannel string) error {
	_, err := p.primaryConnection.Exec("DELETE FROM platform_applications WHERE vendor_id = ? AND app_id = ? AND release_channel = ?", vendorID, appID, releaseChannel)
	if err == nil {
		p.update()
	}
	return err
}

func (p *Provider) GetPlatformVendors() ([]rubix.PlatformVendor, error) {
	rows, err := p.primaryConnection.Query("SELECT vendor_id, name, description, logo_url, icon, discovery, discovery_token FROM platform_vendors")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vendors []rubix.PlatformVendor
	for rows.Next() {
		var v rubix.PlatformVendor
		if err := rows.Scan(&v.VendorID, &v.Name, &v.Description, &v.LogoURL, &v.Icon, &v.Discovery, &v.DiscoveryToken); err != nil {
			return nil, err
		}
		vendors = append(vendors, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return vendors, nil
}

func (p *Provider) StorePlatformVendor(vendor rubix.PlatformVendor) error {
	query := "INSERT INTO platform_vendors (vendor_id, name, description, logo_url, icon, discovery, discovery_token) VALUES (?, ?, ?, ?, ?, ?, ?)"
	if p.SqlLite {
		query += " ON CONFLICT(vendor_id) DO UPDATE SET name=excluded.name, description=excluded.description, logo_url=excluded.logo_url, icon=excluded.icon, discovery=excluded.discovery, discovery_token=excluded.discovery_token"
	} else {
		query = strings.Replace(query, "INSERT INTO", "REPLACE INTO", 1)
	}

	_, err := p.primaryConnection.Exec(query, vendor.VendorID, vendor.Name, vendor.Description, vendor.LogoURL, vendor.Icon, vendor.Discovery, vendor.DiscoveryToken)
	if err == nil {
		p.update()
	}
	return err
}

func (p *Provider) RemovePlatformVendor(vendorID string) error {
	_, err := p.primaryConnection.Exec("DELETE FROM platform_vendors WHERE vendor_id = ?", vendorID)
	if err == nil {
		p.update()
	}
	return err
}
