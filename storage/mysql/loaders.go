package mysql

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/kubex/definitions-go/app"
	"github.com/kubex/rubix-storage/rubix"
	"golang.org/x/sync/errgroup"
)

func (p *Provider) GetWorkspaceUUIDByAlias(alias string) (string, error) {
	q := p.primaryConnection.QueryRow("SELECT uuid FROM workspaces WHERE alias = ?", alias)
	located := ""
	err := q.Scan(&located)
	return located, err
}

func (p *Provider) GetUserWorkspaceUUIDs(userId string) ([]string, error) {
	rows, err := p.primaryConnection.Query("SELECT workspace FROM workspace_memberships WHERE user = ?", userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	workspaces := []string{}
	for rows.Next() {
		var workspace string
		if err := rows.Scan(&workspace); err != nil {
			return nil, err
		}
		workspaces = append(workspaces, workspace)
	}
	return workspaces, nil
}

func (p *Provider) GetWorkspaceMembers(workspaceUuid string) ([]rubix.WorkspaceMembership, error) {

	rows, err := p.primaryConnection.Query("SELECT user, workspace, since FROM workspace_memberships WHERE workspace = ?", workspaceUuid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []rubix.WorkspaceMembership
	for rows.Next() {
		var member rubix.WorkspaceMembership
		if err := rows.Scan(&member.User, &member.Workspace, &member.Since); err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, nil
}

func (p *Provider) RetrieveWorkspace(workspaceUuid string) (*rubix.Workspace, error) {
	q := p.primaryConnection.QueryRow("SELECT uuid, alias, domain, name, icon, installedApplications FROM workspaces WHERE uuid = ?", workspaceUuid)
	located := rubix.Workspace{}
	installedApplicationsJson := ""
	err := q.Scan(&located.Uuid, &located.Alias, &located.Domain, &located.Name, &located.Icon, &installedApplicationsJson)
	json.Unmarshal([]byte(installedApplicationsJson), &located.InstalledApplications)
	return &located, err
}

func (p *Provider) GetAuthData(workspaceUuid, userUuid string, appIDs ...app.GlobalAppID) ([]rubix.DataResult, error) {
	subQuery := " AND ("
	for i, appID := range appIDs {
		if i == 0 {
			subQuery += ""
		} else {
			subQuery += " OR "
		}
		subQuery += "(`vendor` = '" + appID.VendorID + "' AND (`app` = '" + appID.AppID + "' OR `app` IS NULL))"
	}
	subQuery += ")"

	order := "ORDER BY user ASC, app ASC, `key` ASC"
	rows, err := p.primaryConnection.Query("SELECT `vendor`, `app`, `key`, `value` FROM auth_data WHERE workspace = ? AND (user = ? OR user IS NULL) "+subQuery+order, workspaceUuid, userUuid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []rubix.DataResult
	for rows.Next() {
		var vendor, key, value string
		var app sql.NullString
		if err := rows.Scan(&vendor, &app, &key, &value); err != nil {
			return nil, err
		}
		result = append(result, rubix.DataResult{
			VendorID: vendor,
			AppID:    app.String,
			Key:      key,
			Value:    value,
		})
	}
	return result, nil
}

func (p *Provider) GetPermissionStatements(lookup rubix.Lookup, permissions ...app.ScopedKey) ([]app.PermissionStatement, error) {
	if len(permissions) == 0 {
		return nil, nil
	}

	params := []interface{}{lookup.UserUUID, lookup.WorkspaceUUID}
	for _, perm := range permissions {
		params = append(params, perm.String())
	}

	query := "SELECT rp.permission,rp.resource, rp.allow" +
		" FROM user_roles AS ur" +
		" INNER JOIN roles AS r ON ur.role = r.role AND ur.workspace = r.workspace" +
		" INNER JOIN role_permissions AS rp ON rp.role = r.role AND rp.workspace = r.workspace" +
		" WHERE rp.resource = '' " + // Resource not supported in query
		" AND ur.user = ? AND ur.workspace = ?" +
		" AND rp.permission IN (?" + strings.Repeat(",?", len(permissions)-1) + ")"

	rows, err := p.primaryConnection.Query(query, params...)
	if err != nil {
		log.Println(err)
		panic(err)
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]permissionResult)
	for rows.Next() {
		newResult := permissionResult{}
		if err := rows.Scan(&newResult.PermissionKey, &newResult.Resource, &newResult.Allow); err != nil {
			return nil, err
		}
		if _, ok := result[newResult.PermissionKey]; !ok || !newResult.Allow {
			result[newResult.PermissionKey] = newResult
		}
	}

	var statements []app.PermissionStatement
	for _, res := range result {
		effect := app.PermissionEffectAllow
		if !res.Allow {
			effect = app.PermissionEffectDeny
		}
		statements = append(statements, app.PermissionStatement{
			Effect:     effect,
			Permission: app.ScopedKeyFromString(res.PermissionKey),
			Resource:   "",
		})
	}

	return statements, nil
}

func (p *Provider) UserHasPermission(lookup rubix.Lookup, permissions ...app.ScopedKey) (bool, error) {
	if len(permissions) == 0 {
		return true, nil
	}

	statements, err := p.GetPermissionStatements(lookup, permissions...)
	if err != nil {
		return false, err
	}

	requireAll := make(map[string]bool)
	for _, s := range statements {
		requireAll[s.Permission.String()] = s.Effect != app.PermissionEffectDeny
	}

	for _, perm := range permissions {
		if allow, has := requireAll[perm.String()]; !allow || !has {
			return false, nil
		}
	}

	return true, nil
}

// GetRole todo, can do async?
func (p *Provider) GetRole(workspace, role string) (*rubix.Role, error) {

	row := p.primaryConnection.QueryRow("SELECT workspace, role, name, description FROM roles WHERE workspace = ? AND role = ?", workspace, role)

	var ret rubix.Role
	err := row.Scan(&ret.Workspace, &ret.Role, &ret.Name, &ret.Description)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, rubix.ErrNoResultFound
	}
	if err != nil {
		return nil, err
	}

	// Get users
	rows, err := p.primaryConnection.Query("SELECT user FROM user_roles WHERE workspace = ? AND role = ?", workspace, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {

		var user string
		err = rows.Scan(&user)
		if err != nil {
			return nil, err
		}

		ret.Users = append(ret.Users, user)
	}

	// Get permissions
	rows, err = p.primaryConnection.Query("SELECT permission FROM role_permissions WHERE workspace = ? AND role = ?", workspace, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {

		var permission string
		err = rows.Scan(&permission)
		if err != nil {
			return nil, err
		}

		ret.Permissions = append(ret.Permissions, permission)
	}

	return &ret, nil
}

func (p *Provider) GetRoles(workspace string) ([]rubix.Role, error) {

	rows, err := p.primaryConnection.Query("SELECT workspace, role, name, description FROM roles WHERE workspace = ?", workspace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []rubix.Role
	for rows.Next() {

		var role rubix.Role
		err = rows.Scan(&role.Workspace, &role.Role, &role.Name, &role.Description)
		if err != nil {
			return nil, err
		}

		roles = append(roles, role)
	}

	return roles, nil
}

func (p *Provider) CreateRole(workspace, role, name, description string, permissions, users []string) error {

	res, err := p.primaryConnection.Exec("INSERT INTO roles (workspace, role, name, description) VALUES (?, ?, ?, ?)", workspace, role, name, description)
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}

	if id == 0 {
		return errors.New("role not created")
	}

	return p.MutateRole(workspace, name, rubix.WithUsersToAdd(users...), rubix.WithPermsToAdd(permissions...))
}

// MutateRole - todo, can do these in async
func (p *Provider) MutateRole(workspace, role string, options ...rubix.MutateRoleOption) error {

	payload := rubix.MutateRolePayload{}
	for _, opt := range options {
		opt(&payload)
	}

	g := errgroup.Group{}

	g.Go(func() error {

		if payload.Title != nil || payload.Description != nil {

			var fields []string
			if payload.Title != nil {
				fields = append(fields, "name = ?")
			}
			if payload.Description != nil {
				fields = append(fields, "description = ?")
			}

			var vals []any
			if payload.Title != nil {
				vals = append(vals, *payload.Title)
			}
			if payload.Description != nil {
				vals = append(vals, *payload.Description)
			}

			vals = append(vals, workspace, role)

			result, err := p.primaryConnection.Exec(fmt.Sprintf("UPDATE roles SET %s WHERE workspace = ? AND role = ?", strings.Join(fields, ", ")), vals...)
			if err != nil {
				return err
			}

			rows, err := result.RowsAffected()
			if err != nil {
				return err
			}

			if rows == 0 {
				return rubix.ErrNoResultFound
			}
		}

		return nil
	})

	g.Go(func() error {

		for _, user := range payload.UsersToAdd {
			_, err := p.primaryConnection.Exec("INSERT INTO user_roles (workspace, user, role) VALUES (?, ?, ?)", workspace, user, role)
			if err != nil {
				return err
			}
		}

		return nil
	})

	g.Go(func() error {

		for _, user := range payload.UsersToRem {
			_, err := p.primaryConnection.Exec("DELETE FROM user_roles WHERE workspace = ? AND user = ? AND role = ?", workspace, user, role)
			if err != nil {
				return err
			}
		}

		return nil
	})

	g.Go(func() error {

		for _, perm := range payload.PermsToAdd {
			_, err := p.primaryConnection.Exec("INSERT INTO role_permissions (workspace, role, permission, resource, allow) VALUES (?, ?, ?, '', 1)", workspace, role, perm)
			if err != nil {
				return err
			}
		}

		return nil
	})

	g.Go(func() error {

		for _, perm := range payload.PermsToRem {
			_, err := p.primaryConnection.Exec("DELETE FROM role_permissions WHERE workspace = ? AND role = ? AND permission = ? AND resource = ''", workspace, role, perm)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return g.Wait()
}
