package sql

import (
	"database/sql"
	"errors"
	"time"

	"github.com/kubex/rubix-storage/rubix"
)

func (p *Provider) GetBlueprints() ([]rubix.Blueprint, error) {
	rows, err := p.primaryConnection.Query("SELECT id, vendor_id, name, description, icon, latest_version, source_url, created_at, updated_at FROM blueprints ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blueprints []rubix.Blueprint
	for rows.Next() {
		var b rubix.Blueprint
		if err := rows.Scan(&b.ID, &b.VendorID, &b.Name, &b.Description, &b.Icon, &b.LatestVersion, &b.SourceURL, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		blueprints = append(blueprints, b)
	}
	return blueprints, rows.Err()
}

func (p *Provider) GetBlueprint(id string) (*rubix.Blueprint, error) {
	var b rubix.Blueprint
	err := p.primaryConnection.QueryRow("SELECT id, vendor_id, name, description, icon, latest_version, source_url, created_at, updated_at FROM blueprints WHERE id = ?", id).
		Scan(&b.ID, &b.VendorID, &b.Name, &b.Description, &b.Icon, &b.LatestVersion, &b.SourceURL, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (p *Provider) StoreBlueprint(blueprint rubix.Blueprint) error {
	now := time.Now()
	if p.SqlLite {
		_, err := p.primaryConnection.Exec(
			"INSERT INTO blueprints (id, vendor_id, name, description, icon, latest_version, source_url, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET vendor_id=excluded.vendor_id, name=excluded.name, description=excluded.description, icon=excluded.icon, latest_version=excluded.latest_version, source_url=excluded.source_url, updated_at=excluded.updated_at",
			blueprint.ID, blueprint.VendorID, blueprint.Name, blueprint.Description, blueprint.Icon, blueprint.LatestVersion, blueprint.SourceURL, now, now)
		if err == nil {
			p.update()
		}
		return err
	}
	_, err := p.primaryConnection.Exec(
		"INSERT INTO blueprints (id, vendor_id, name, description, icon, latest_version, source_url, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE vendor_id=VALUES(vendor_id), name=VALUES(name), description=VALUES(description), icon=VALUES(icon), latest_version=VALUES(latest_version), source_url=VALUES(source_url), updated_at=VALUES(updated_at)",
		blueprint.ID, blueprint.VendorID, blueprint.Name, blueprint.Description, blueprint.Icon, blueprint.LatestVersion, blueprint.SourceURL, now, now)
	if err == nil {
		p.update()
	}
	return err
}

func (p *Provider) RemoveBlueprint(id string) error {
	_, err := p.primaryConnection.Exec("DELETE FROM blueprints WHERE id = ?", id)
	if err == nil {
		p.update()
	}
	return err
}

func (p *Provider) GetBlueprintVersions(blueprintID string) ([]rubix.BlueprintVersion, error) {
	rows, err := p.primaryConnection.Query("SELECT blueprint_id, version, definition, content_hash, created_at FROM blueprint_versions WHERE blueprint_id = ? ORDER BY created_at DESC", blueprintID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []rubix.BlueprintVersion
	for rows.Next() {
		var v rubix.BlueprintVersion
		if err := rows.Scan(&v.BlueprintID, &v.Version, &v.Definition, &v.ContentHash, &v.CreatedAt); err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}
	return versions, rows.Err()
}

func (p *Provider) GetBlueprintVersion(blueprintID, version string) (*rubix.BlueprintVersion, error) {
	var v rubix.BlueprintVersion
	err := p.primaryConnection.QueryRow("SELECT blueprint_id, version, definition, content_hash, created_at FROM blueprint_versions WHERE blueprint_id = ? AND version = ?", blueprintID, version).
		Scan(&v.BlueprintID, &v.Version, &v.Definition, &v.ContentHash, &v.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (p *Provider) StoreBlueprintVersion(version rubix.BlueprintVersion) error {
	now := time.Now()
	if p.SqlLite {
		_, err := p.primaryConnection.Exec(
			"INSERT INTO blueprint_versions (blueprint_id, version, definition, content_hash, created_at) VALUES (?, ?, ?, ?, ?) ON CONFLICT(blueprint_id, version) DO UPDATE SET definition=excluded.definition, content_hash=excluded.content_hash",
			version.BlueprintID, version.Version, version.Definition, version.ContentHash, now)
		if err == nil {
			p.update()
		}
		return err
	}
	_, err := p.primaryConnection.Exec(
		"INSERT INTO blueprint_versions (blueprint_id, version, definition, content_hash, created_at) VALUES (?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE definition=VALUES(definition), content_hash=VALUES(content_hash)",
		version.BlueprintID, version.Version, version.Definition, version.ContentHash, now)
	if err == nil {
		p.update()
	}
	return err
}

func (p *Provider) GetWorkspaceBlueprints(workspaceUUID string) ([]rubix.WorkspaceBlueprint, error) {
	rows, err := p.primaryConnection.Query("SELECT workspace_uuid, blueprint_id, subscribed_version, status, subscribed_at FROM workspace_blueprints WHERE workspace_uuid = ?", workspaceUUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []rubix.WorkspaceBlueprint
	for rows.Next() {
		var s rubix.WorkspaceBlueprint
		if err := rows.Scan(&s.WorkspaceUUID, &s.BlueprintID, &s.SubscribedVersion, &s.Status, &s.SubscribedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

func (p *Provider) SubscribeWorkspaceBlueprint(sub rubix.WorkspaceBlueprint) error {
	now := time.Now()
	if p.SqlLite {
		_, err := p.primaryConnection.Exec(
			"INSERT INTO workspace_blueprints (workspace_uuid, blueprint_id, subscribed_version, status, subscribed_at) VALUES (?, ?, ?, ?, ?) ON CONFLICT(workspace_uuid, blueprint_id) DO UPDATE SET subscribed_version=excluded.subscribed_version, status=excluded.status",
			sub.WorkspaceUUID, sub.BlueprintID, sub.SubscribedVersion, sub.Status, now)
		if err == nil {
			p.update()
		}
		return err
	}
	_, err := p.primaryConnection.Exec(
		"REPLACE INTO workspace_blueprints (workspace_uuid, blueprint_id, subscribed_version, status, subscribed_at) VALUES (?, ?, ?, ?, ?)",
		sub.WorkspaceUUID, sub.BlueprintID, sub.SubscribedVersion, sub.Status, now)
	if err == nil {
		p.update()
	}
	return err
}

func (p *Provider) UnsubscribeWorkspaceBlueprint(workspaceUUID, blueprintID string) error {
	_, err := p.primaryConnection.Exec("DELETE FROM workspace_blueprints WHERE workspace_uuid = ? AND blueprint_id = ?", workspaceUUID, blueprintID)
	if err != nil {
		return err
	}
	_, err = p.primaryConnection.Exec("DELETE FROM workspace_blueprint_resources WHERE workspace_uuid = ? AND blueprint_id = ?", workspaceUUID, blueprintID)
	if err == nil {
		p.update()
	}
	return err
}

func (p *Provider) UpdateWorkspaceBlueprintStatus(workspaceUUID, blueprintID, status string) error {
	_, err := p.primaryConnection.Exec("UPDATE workspace_blueprints SET status = ? WHERE workspace_uuid = ? AND blueprint_id = ?", status, workspaceUUID, blueprintID)
	if err == nil {
		p.update()
	}
	return err
}

func (p *Provider) UpdateWorkspaceBlueprintVersion(workspaceUUID, blueprintID, version string) error {
	_, err := p.primaryConnection.Exec("UPDATE workspace_blueprints SET subscribed_version = ?, status = 'active' WHERE workspace_uuid = ? AND blueprint_id = ?", version, workspaceUUID, blueprintID)
	if err == nil {
		p.update()
	}
	return err
}

func (p *Provider) GetWorkspaceBlueprintResources(workspaceUUID, blueprintID string) ([]rubix.WorkspaceBlueprintResource, error) {
	rows, err := p.primaryConnection.Query("SELECT workspace_uuid, blueprint_id, resource_type, resource_key, desired_value, applied_value, status, last_synced_at FROM workspace_blueprint_resources WHERE workspace_uuid = ? AND blueprint_id = ?", workspaceUUID, blueprintID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resources []rubix.WorkspaceBlueprintResource
	for rows.Next() {
		var r rubix.WorkspaceBlueprintResource
		if err := rows.Scan(&r.WorkspaceUUID, &r.BlueprintID, &r.ResourceType, &r.ResourceKey, &r.DesiredValue, &r.AppliedValue, &r.Status, &r.LastSyncedAt); err != nil {
			return nil, err
		}
		resources = append(resources, r)
	}
	return resources, rows.Err()
}

func (p *Provider) SetWorkspaceBlueprintResource(resource rubix.WorkspaceBlueprintResource) error {
	now := time.Now()
	if p.SqlLite {
		_, err := p.primaryConnection.Exec(
			"INSERT INTO workspace_blueprint_resources (workspace_uuid, blueprint_id, resource_type, resource_key, desired_value, applied_value, status, last_synced_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(workspace_uuid, blueprint_id, resource_type, resource_key) DO UPDATE SET desired_value=excluded.desired_value, applied_value=excluded.applied_value, status=excluded.status, last_synced_at=excluded.last_synced_at",
			resource.WorkspaceUUID, resource.BlueprintID, resource.ResourceType, resource.ResourceKey, resource.DesiredValue, resource.AppliedValue, resource.Status, now)
		if err == nil {
			p.update()
		}
		return err
	}
	_, err := p.primaryConnection.Exec(
		"INSERT INTO workspace_blueprint_resources (workspace_uuid, blueprint_id, resource_type, resource_key, desired_value, applied_value, status, last_synced_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE desired_value=VALUES(desired_value), applied_value=VALUES(applied_value), status=VALUES(status), last_synced_at=VALUES(last_synced_at)",
		resource.WorkspaceUUID, resource.BlueprintID, resource.ResourceType, resource.ResourceKey, resource.DesiredValue, resource.AppliedValue, resource.Status, now)
	if err == nil {
		p.update()
	}
	return err
}

func (p *Provider) RemoveWorkspaceBlueprintResource(workspaceUUID, blueprintID, resourceType, resourceKey string) error {
	_, err := p.primaryConnection.Exec("DELETE FROM workspace_blueprint_resources WHERE workspace_uuid = ? AND blueprint_id = ? AND resource_type = ? AND resource_key = ?",
		workspaceUUID, blueprintID, resourceType, resourceKey)
	if err == nil {
		p.update()
	}
	return err
}
