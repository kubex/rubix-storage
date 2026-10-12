package sql

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/kubex/rubix-storage/rubix"
)

func (p *Provider) LinkUserIdentity(workspace string, id rubix.UserIdentity, replace bool) error {
	return p.linkUserIdentity(workspace, id, replace, true)
}

func (p *Provider) linkUserIdentity(workspace string, id rubix.UserIdentity, replace, retryOnRace bool) error {
	var existing string
	err := p.primaryConnection.QueryRow(
		"SELECT user_id FROM workspace_user_identities WHERE workspace = ? AND vendor = ? AND scope = ? AND external_id = ?",
		workspace, id.Vendor, id.Scope, id.ExternalID,
	).Scan(&existing)

	now := time.Now().UTC()
	switch {
	case errors.Is(err, sql.ErrNoRows):
		_, err = p.primaryConnection.Exec(
			"INSERT INTO workspace_user_identities (workspace, vendor, scope, external_id, user_id, source, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			workspace, id.Vendor, id.Scope, id.ExternalID, id.UserID, string(id.Source), now, now,
		)
		if retryOnRace && p.isDuplicateConflict(err) {
			return p.linkUserIdentity(workspace, id, replace, false)
		}
	case err != nil:
		return err
	case existing == id.UserID:
		return nil
	case !replace:
		return rubix.ErrIdentityConflict
	default:
		_, err = p.primaryConnection.Exec(
			"UPDATE workspace_user_identities SET user_id = ?, source = ?, updated_at = ? WHERE workspace = ? AND vendor = ? AND scope = ? AND external_id = ?",
			id.UserID, string(id.Source), now, workspace, id.Vendor, id.Scope, id.ExternalID,
		)
	}
	if err != nil {
		return err
	}
	p.update()
	return nil
}

func (p *Provider) UnlinkUserIdentity(workspace, vendor, scope, externalID string) error {
	_, err := p.primaryConnection.Exec(
		"DELETE FROM workspace_user_identities WHERE workspace = ? AND vendor = ? AND scope = ? AND external_id = ?",
		workspace, vendor, scope, externalID,
	)
	if err != nil {
		return err
	}
	p.update()
	return nil
}

// ResolveUserIdentities maps external IDs to user IDs; unlinked IDs are absent from the result.
func (p *Provider) ResolveUserIdentities(workspace, vendor, scope string, externalIDs ...string) (map[string]string, error) {
	resolved := map[string]string{}
	values := []any{workspace, vendor, scope}
	var placeholders []string
	for _, eid := range externalIDs {
		if eid == "" {
			continue
		}
		values = append(values, eid)
		placeholders = append(placeholders, "?")
	}
	if len(placeholders) == 0 {
		return resolved, nil
	}

	rows, err := p.primaryConnection.Query(
		"SELECT external_id, user_id FROM workspace_user_identities WHERE workspace = ? AND vendor = ? AND scope = ? AND external_id IN ("+strings.Join(placeholders, ",")+")",
		values...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var eid, uid string
		if err := rows.Scan(&eid, &uid); err != nil {
			return nil, err
		}
		resolved[eid] = uid
	}
	return resolved, rows.Err()
}

func (p *Provider) GetUserIdentities(workspace string, userIDs ...string) ([]rubix.UserIdentity, error) {
	values := []any{workspace}
	var placeholders []string
	for _, uid := range userIDs {
		if uid == "" {
			continue
		}
		values = append(values, uid)
		placeholders = append(placeholders, "?")
	}
	if len(placeholders) == 0 {
		return nil, nil
	}

	rows, err := p.primaryConnection.Query(
		"SELECT workspace, vendor, scope, external_id, user_id, source, created_at, updated_at FROM workspace_user_identities WHERE workspace = ? AND user_id IN ("+strings.Join(placeholders, ",")+")",
		values...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []rubix.UserIdentity
	for rows.Next() {
		var it rubix.UserIdentity
		var source string
		createdAt := sql.NullString{}
		updatedAt := sql.NullString{}
		if err := rows.Scan(&it.Workspace, &it.Vendor, &it.Scope, &it.ExternalID, &it.UserID, &source, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		it.Source = rubix.IdentitySource(source)
		if createdAt.Valid && createdAt.String != "" {
			it.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt.String)
		}
		if updatedAt.Valid && updatedAt.String != "" {
			it.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt.String)
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func (p *Provider) deleteUserIdentities(workspace, userID string) error {
	_, err := p.primaryConnection.Exec("DELETE FROM workspace_user_identities WHERE workspace = ? AND user_id = ?", workspace, userID)
	return err
}
