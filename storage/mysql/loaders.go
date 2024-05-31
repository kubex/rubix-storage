package mysql

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/go-sql-driver/mysql"
	"github.com/kubex/definitions-go/app"
	"github.com/kubex/rubix-storage/rubix"
	"golang.org/x/sync/errgroup"
)

const (
	mySQLDuplicateEntry = 1062
)

func (p *Provider) GetWorkspaceUUIDByAlias(alias string) (string, error) {
	q := p.primaryConnection.QueryRow("SELECT uuid FROM workspaces WHERE alias = ?", alias)
	located := ""
	err := q.Scan(&located)
	return located, err
}

func (p *Provider) AddMemberToWorkspace(workspaceID, userID string) error {

	_, err := p.primaryConnection.Exec("INSERT INTO workspace_memberships (user, workspace, type, since, state_since, state) VALUES (?, ?, ?, NOW(), NOW(), ?)", userID, workspaceID, rubix.MembershipTypeMember, rubix.MembershipStateActive)

	var me2 *mysql.MySQLError
	if errors.As(err, &me2) && me2.Number == mySQLDuplicateEntry {
		return nil
	}
	return err
}

func (p *Provider) CreateUser(userID, name string) error {

	_, err := p.primaryConnection.Exec("INSERT INTO users (user, name) VALUES (?, ?)", userID, name)

	var me1 *mysql.MySQLError
	if errors.As(err, &me1) && me1.Number == mySQLDuplicateEntry {
		return nil
	}
	return err
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

// GetWorkspaceMembers - userID is optional
func (p *Provider) GetWorkspaceMembers(workspaceUuid, userID string) ([]rubix.Membership, error) {

	var fields = []string{"workspace = ?"}
	var values = []any{workspaceUuid}

	if userID != "" {
		fields = append(fields, "user = ?")
		values = append(values, userID)
	}

	q := fmt.Sprintf("SELECT user, type, partner_id, since, state, state_since  FROM workspace_memberships WHERE %s", strings.Join(fields, " AND "))

	rows, err := p.primaryConnection.Query(q, values...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []rubix.Membership
	for rows.Next() {
		var member = rubix.Membership{Workspace: workspaceUuid}
		if err := rows.Scan(&member.UserID, &member.Type, &member.PartnerID, &member.Since, &member.State, &member.StateSince); err != nil {
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
	icon := sql.NullString{}
	err := q.Scan(&located.Uuid, &located.Alias, &located.Domain, &located.Name, &icon, &installedApplicationsJson)
	json.Unmarshal([]byte(installedApplicationsJson), &located.InstalledApplications)
	located.Icon = icon.String
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

func (p *Provider) MutateUser(workspace, user string, options ...rubix.MutateUserOption) error {

	payload := rubix.MutateUserPayload{}
	for _, opt := range options {
		opt(&payload)
	}

	g := errgroup.Group{}
	g.Go(func() error {

		for _, role := range payload.RolesToAdd {
			_, err := p.primaryConnection.Exec("INSERT INTO user_roles (workspace, user, role) VALUES (?, ?, ?)", workspace, user, role)

			var me *mysql.MySQLError
			if errors.As(err, &me) && me.Number == mySQLDuplicateEntry {
				continue
			}
			if err != nil {
				return err
			}
		}

		return nil
	})
	g.Go(func() error {

		for _, role := range payload.RolesToRemove {
			_, err := p.primaryConnection.Exec("DELETE FROM user_roles WHERE workspace = ? AND user = ? AND role = ?", workspace, user, role)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return g.Wait()
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

func (p *Provider) SetMembershipType(workspace, user string, MembershipType rubix.MembershipType) error {

	switch MembershipType {
	case rubix.MembershipTypeOwner, rubix.MembershipTypeMember, rubix.MembershipTypeSupport:
	default:
		return errors.New("invalid user type")
	}

	_, err := p.primaryConnection.Exec("UPDATE workspace_memberships SET type = ? WHERE workspace = ? AND user = ?", MembershipType, workspace, user)
	return err
}

func (p *Provider) SetMembershipState(workspace, user string, userState rubix.MembershipState) error {

	switch userState {
	case rubix.MembershipStatePending, rubix.MembershipStateActive, rubix.MembershipStateSuspended, rubix.MembershipStateArchived:
	case rubix.MembershipStateRemoved:
		return errors.New("use RemoveUserFromWorkspace()")
	default:
		return errors.New("invalid user state")
	}

	_, err := p.primaryConnection.Exec("UPDATE workspace_memberships SET state = ? WHERE workspace = ? AND user = ?", userState, workspace, user)
	return err
}

func (p *Provider) RemoveUserFromWorkspace(workspace, user string) error {

	_, err := p.primaryConnection.Exec("UPDATE workspace_memberships SET state = ? WHERE workspace = ? AND user = ?", rubix.MembershipStateRemoved, workspace, user)
	return err
}

func (p *Provider) GetRole(workspace, role string) (*rubix.Role, error) {

	var ret = rubix.Role{
		Workspace: workspace,
		Role:      role,
	}

	g := errgroup.Group{}
	g.Go(func() error {

		row := p.primaryConnection.QueryRow("SELECT name, description FROM roles WHERE workspace = ? AND role = ?", workspace, role)

		err := row.Scan(&ret.Name, &ret.Description)
		if errors.Is(err, sql.ErrNoRows) {
			return rubix.ErrNoResultFound
		}

		return err
	})
	g.Go(func() error {

		rows, err := p.primaryConnection.Query("SELECT user FROM user_roles WHERE workspace = ? AND role = ?", workspace, role)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {

			var user string
			err = rows.Scan(&user)
			if err != nil {
				return err
			}

			ret.Users = append(ret.Users, user)
		}

		return nil
	})
	g.Go(func() error {

		rows, err := p.primaryConnection.Query("SELECT permission, resource, allow, meta FROM role_permissions WHERE workspace = ? AND role = ?", workspace, role)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {

			var permission = rubix.RolePermission{Workspace: workspace, Role: role}
			err = rows.Scan(&permission.Permission, &permission.Resource, &permission.Allow, &permission.Meta)
			if err != nil {
				return err
			}

			ret.Permissions = append(ret.Permissions, permission)
		}

		return nil
	})

	return &ret, g.Wait()
}

func (p *Provider) GetRoles(workspace string) ([]rubix.Role, error) {

	rows, err := p.primaryConnection.Query("SELECT role, name, description FROM roles WHERE workspace = ?", workspace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []rubix.Role
	for rows.Next() {

		var role = rubix.Role{Workspace: workspace}
		err = rows.Scan(&role.Role, &role.Name, &role.Description)
		if err != nil {
			return nil, err
		}

		roles = append(roles, role)
	}

	return roles, nil
}

func (p *Provider) GetUserRoles(workspace, user string) ([]rubix.UserRole, error) {

	rows, err := p.primaryConnection.Query("SELECT role FROM user_roles WHERE workspace = ? AND user = ?", workspace, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []rubix.UserRole
	for rows.Next() {

		var role = rubix.UserRole{Workspace: workspace, User: user}
		err = rows.Scan(&role.Role)
		if err != nil {
			return nil, err
		}

		roles = append(roles, role)
	}

	return roles, nil
}

func (p *Provider) DeleteRole(workspace, role string) error {

	_, err := p.primaryConnection.Exec("DELETE FROM roles  WHERE workspace = ? AND role = ?", workspace, role)
	return err
}

func (p *Provider) CreateRole(workspace, role, name, description string, permissions, users []string) error {

	_, err := p.primaryConnection.Exec("INSERT INTO roles (workspace, role, name, description) VALUES (?, ?, ?, ?)", workspace, role, name, description)

	var me *mysql.MySQLError
	if errors.As(err, &me) && me.Number == mySQLDuplicateEntry {
		return errors.New("role already exists")
	}
	if err != nil {
		return err
	}

	return p.MutateRole(workspace, role, rubix.WithUsersToAdd(users...), rubix.WithPermsToAdd(permissions...))
}

func (p *Provider) MutateRole(workspace, role string, options ...rubix.MutateRoleOption) error {

	payload := rubix.MutateRolePayload{}
	for _, opt := range options {
		opt(&payload)
	}

	g := errgroup.Group{}
	g.Go(func() error {

		if payload.Title != nil || payload.Description != nil {

			var fields []string
			var vals []any

			if payload.Title != nil {
				fields = append(fields, "name = ?")
				vals = append(vals, *payload.Title)
			}
			if payload.Description != nil {
				fields = append(fields, "description = ?")
				vals = append(vals, *payload.Description)
			}

			vals = append(vals, workspace, role)

			q := fmt.Sprintf("UPDATE roles SET %s WHERE workspace = ? AND role = ?", strings.Join(fields, ", "))
			result, err := p.primaryConnection.Exec(q, vals...)
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

			var me *mysql.MySQLError
			if errors.As(err, &me) && me.Number == mySQLDuplicateEntry {
				continue
			}
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
			_, err := p.primaryConnection.Exec("INSERT INTO role_permissions (workspace, role, permission) VALUES (?, ?, ?)", workspace, role, perm)

			var me *mysql.MySQLError
			if errors.As(err, &me) && me.Number == mySQLDuplicateEntry {
				continue
			}
			if err != nil {
				return err
			}
		}

		return nil
	})
	g.Go(func() error {

		for _, perm := range payload.PermsToRem {
			_, err := p.primaryConnection.Exec("DELETE FROM role_permissions WHERE workspace = ? AND role = ? AND permission = ?", workspace, role, perm)
			if err != nil {
				return err
			}
		}

		return nil
	})

	return g.Wait()
}
