package sql

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/kubex/definitions-go/app"
	"github.com/kubex/rubix-storage/rubix"
	"golang.org/x/sync/errgroup"
)

const (
	mySQLDuplicateEntry   = 1062
	sqlLiteDuplicateEntry = 1555
)

func (p *Provider) CreateWorkspace(workspaceUuid, name, alias, domain string) error {
	_, err := p.primaryConnection.Exec("INSERT INTO workspaces (uuid,name,alias,domain) VALUES (?, ?, ?, ?)", workspaceUuid, name, alias, domain)

	if p.isDuplicateConflict(err) {
		return nil
	}
	p.update()
	return err
}

func (p *Provider) GetWorkspaceUUIDByAlias(alias string) (string, error) {
	q := p.primaryConnection.QueryRow("SELECT uuid FROM workspaces WHERE alias = ?", alias)
	located := ""
	err := q.Scan(&located)
	return located, err
}

func (p *Provider) isDuplicateConflict(err error) bool {
	var me1 *mysql.MySQLError
	if errors.As(err, &me1) && (me1.Number == mySQLDuplicateEntry || me1.Number == sqlLiteDuplicateEntry) {
		return true
	}
	if err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return true
	}
	return false
}

func (p *Provider) AddUserToWorkspace(workspaceID, userID string, as rubix.MembershipType, partnerId string) error {
	var err error
	_, err = p.primaryConnection.Exec("INSERT INTO workspace_memberships (user, workspace, type, since, state_since, state, partner_id) VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?, ?)", userID, workspaceID, as, rubix.MembershipStatePending, partnerId)

	if p.isDuplicateConflict(err) {
		_, err = p.primaryConnection.Exec("UPDATE workspace_memberships SET state_since = CURRENT_TIMESTAMP, state = ?, type = ?, partner_id = ? WHERE state = ? AND user = ? AND workspace = ?", rubix.MembershipStatePending, as, partnerId, rubix.MembershipStateRemoved, userID, workspaceID)
		return err
	}
	p.update()
	return err
}

func (p *Provider) CreateUser(userID, name, email string) error {

	_, err := p.primaryConnection.Exec("INSERT INTO users (user, name, email) VALUES (?, ?, ?)", userID, name, email)

	if p.isDuplicateConflict(err) {
		return nil
	}
	p.update()
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
func (p *Provider) GetWorkspaceMembers(workspaceUuid string, userIDs ...string) ([]rubix.Membership, error) {

	var fields = []string{"workspace = ?", "state != ?"}
	var values = []any{workspaceUuid, rubix.MembershipStateRemoved}

	if len(userIDs) > 0 {
		var placeholders []string
		for _, uid := range userIDs {
			if uid == "" {
				continue
			}
			values = append(values, uid)
			placeholders = append(placeholders, "?")
		}
		if len(placeholders) > 0 {
			fields = append(fields, "m.user IN ("+strings.Join(placeholders, ",")+")")
		}
	}

	q := "SELECT m.user, m.type, m.partner_id, m.since, m.state, m.state_since, u.name, u.email " +
		"FROM workspace_memberships AS m " +
		"LEFT JOIN users AS u ON m.user = u.user " +
		"WHERE " + strings.Join(fields, " AND ")

	rows, err := p.primaryConnection.Query(q, values...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []rubix.Membership
	for rows.Next() {
		var member = rubix.Membership{Workspace: workspaceUuid}
		email := sql.NullString{}
		name := sql.NullString{}
		since := sql.NullString{}
		stateSince := sql.NullString{}
		if scanErr := rows.Scan(&member.UserID, &member.Type, &member.PartnerID, &since, &member.State, &stateSince, &name, &email); scanErr != nil {
			return nil, scanErr
		} else {
			member.Email = email.String
			member.Name = name.String

			if stateSince.Valid && stateSince.String != "" {
				member.StateSince, _ = time.Parse(time.RFC3339Nano, stateSince.String)
			}
			if since.Valid && since.String != "" {
				member.Since, _ = time.Parse(time.RFC3339Nano, since.String)
			}

		}
		members = append(members, member)
	}
	return members, nil
}

func (p *Provider) RetrieveWorkspace(workspaceUuid string) (*rubix.Workspace, error) {
	return p.retrieveWorkspaceBy("uuid", workspaceUuid)
}

func (p *Provider) RetrieveWorkspaceByDomain(domain string) (*rubix.Workspace, error) {
	return p.retrieveWorkspaceBy("domain", domain)
}

func (p *Provider) RetrieveWorkspaces(workspaceUuids ...string) (map[string]*rubix.Workspace, error) {
	if len(workspaceUuids) == 0 {
		return nil, nil
	}

	args := make([]any, len(workspaceUuids))
	inQ := "?"
	args[0] = workspaceUuids[0]
	for i := 1; i < len(workspaceUuids); i++ {
		inQ += ",?"
		args[i] = workspaceUuids[i]
	}

	return p.retrieveWorkspacesByQuery("uuid IN ("+inQ+")", args...)
}

func (p *Provider) retrieveWorkspaceBy(field, match string) (*rubix.Workspace, error) {
	if match == "" {
		return nil, errors.New("invalid match")
	}

	resp, err := p.retrieveWorkspacesByQuery(field+" = ?", match)
	if err != nil {
		return nil, err
	}

	if len(resp) > 0 {
		for _, workspace := range resp {
			return workspace, nil
		}
	}
	return nil, nil
}

func (p *Provider) retrieveWorkspacesByQuery(where string, args ...any) (map[string]*rubix.Workspace, error) {
	resp := make(map[string]*rubix.Workspace)
	rows, err := p.primaryConnection.Query("SELECT uuid, alias, domain, name, icon, installedApplications,defaultApp,systemVendors,footerParts,accessCondition FROM workspaces WHERE "+where, args...)
	if err != nil {
		return resp, err
	}

	defer rows.Close()
	for rows.Next() {
		located := rubix.Workspace{}
		installedApplicationsJson := sql.NullString{}
		footerPartsJson := sql.NullString{}
		accessConditionJson := sql.NullString{}
		sysVendors := sql.NullString{}
		icon := sql.NullString{}
		defaultApp := sql.NullString{}
		scanErr := rows.Scan(&located.Uuid, &located.Alias, &located.Domain, &located.Name, &icon, &installedApplicationsJson, &defaultApp, &sysVendors, &footerPartsJson, &accessConditionJson)
		if scanErr != nil {
			continue
		}
		located.SystemVendors = strings.Split(sysVendors.String, ",")
		located.Icon = icon.String
		located.DefaultApp = app.IDFromString(defaultApp.String)
		json.Unmarshal([]byte(installedApplicationsJson.String), &located.InstalledApplications)
		json.Unmarshal([]byte(footerPartsJson.String), &located.FooterParts)
		json.Unmarshal([]byte(accessConditionJson.String), &located.AccessCondition)
		resp[located.Uuid] = &located
	}

	return resp, err
}

func (p *Provider) SetAuthData(workspaceUuid, userUuid string, value rubix.DataResult, forceUpdate bool) error {
	query := "INSERT INTO auth_data (workspace, user, `vendor`, `app`, `key`, `value`) VALUES (?, ?, ?, ?, ?, ?) "
	uid := sql.NullString{}
	if userUuid != "" {
		uid.String = userUuid
		uid.Valid = true
	}
	aid := sql.NullString{}
	if value.AppID != "" {
		aid.String = value.AppID
		aid.Valid = true
	}
	args := []any{workspaceUuid, uid, value.VendorID, aid, value.Key, value.Value}
	if forceUpdate {
		query += "ON DUPLICATE KEY UPDATE `value` = ?"
		args = append(args, value.Value)
	}
	_, err := p.primaryConnection.Exec(query, args...)
	return err
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

	query := "SELECT rp.permission, rp.resource, rp.allow, r.constraints" +
		" FROM user_roles AS ur" +
		" INNER JOIN roles AS r ON ur.role = r.role AND ur.workspace = r.workspace" +
		" INNER JOIN role_permissions AS rp ON rp.role = r.role AND rp.workspace = r.workspace" +
		" WHERE rp.resource = '' " + // Resource not supported in query
		" AND ur.user = ? AND ur.workspace = ?" +
		" AND rp.permission IN (?" + strings.Repeat(",?", len(permissions)-1) + ")"

	rows, err := p.primaryConnection.Query(query, params...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	result := []permissionResult{}
	for rows.Next() {
		newResult := permissionResult{}
		var roleConstraintsStr sql.NullString

		if err := rows.Scan(&newResult.PermissionKey, &newResult.Resource, &newResult.Allow, &roleConstraintsStr); err != nil {
			return nil, err
		}

		if roleConstraintsStr.Valid {
			if err = json.Unmarshal([]byte(roleConstraintsStr.String), &newResult.RoleConstraints); err != nil {
				return nil, err
			}
		}

		result = append(result, newResult)
	}

	statements := make(map[string]app.PermissionStatement)
	for _, res := range result {
		effect := app.PermissionEffectAllow
		if !res.Allow {
			effect = app.PermissionEffectDeny
		} else if !rubix.CheckRoleConstraints(res.RoleConstraints, lookup) {
			continue
		}

		// only overwrite if the effect is deny
		if _, ok := statements[res.PermissionKey]; !ok || effect == app.PermissionEffectDeny {
			statements[res.PermissionKey] = app.PermissionStatement{
				Effect:     effect,
				Permission: app.ScopedKeyFromString(res.PermissionKey),
				Resource:   "",
			}
		}
	}

	return slices.Collect(maps.Values(statements)), nil
}

func (p *Provider) MutateUser(workspace, user string, options ...rubix.MutateUserOption) error {

	if len(options) == 0 {
		return nil
	}

	payload := rubix.MutateUserPayload{}
	for _, opt := range options {
		opt(&payload)
	}

	g := errgroup.Group{}
	g.Go(func() error {

		for _, role := range payload.RolesToAdd {
			_, err := p.primaryConnection.Exec("INSERT INTO user_roles (workspace, user, role) VALUES (?, ?, ?)", workspace, user, role)

			if p.isDuplicateConflict(err) {
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
	p.update()
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
	p.update()
	return err
}

func (p *Provider) RemoveUserFromWorkspace(workspace, user string) error {

	_, err := p.primaryConnection.Exec("UPDATE workspace_memberships SET state = ? WHERE workspace = ? AND user = ?", rubix.MembershipStateRemoved, workspace, user)
	p.update()
	return err
}

func (p *Provider) GetRole(workspace, role string) (*rubix.Role, error) {

	var ret = rubix.Role{
		Workspace: workspace,
		ID:        role,
	}

	g := errgroup.Group{}
	g.Go(func() error {

		row := p.primaryConnection.QueryRow("SELECT name, description, constraints FROM roles WHERE workspace = ? AND role = ?", workspace, role)

		var constraintsStr sql.NullString
		err := row.Scan(&ret.Name, &ret.Description, &constraintsStr)
		if errors.Is(err, sql.ErrNoRows) {
			return rubix.ErrNoResultFound
		}

		if constraintsStr.Valid {
			err = json.Unmarshal([]byte(constraintsStr.String), &ret.Constraints)
			if err != nil {
				return err
			}
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

		rows, err := p.primaryConnection.Query("SELECT permission, resource, allow, meta, constraints FROM role_permissions WHERE workspace = ? AND role = ?", workspace, role)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {

			var permission = rubix.RolePermission{Workspace: workspace, Role: role}
			var constraintsStr sql.NullString
			err = rows.Scan(&permission.Permission, &permission.Resource, &permission.Allow, &permission.Meta, &constraintsStr)
			if err != nil {
				return err
			}

			if constraintsStr.Valid {
				err = json.Unmarshal([]byte(constraintsStr.String), &permission.Constraints)
				if err != nil {
					return err
				}
			}

			ret.Permissions = append(ret.Permissions, permission)
		}

		return nil
	})

	return &ret, g.Wait()
}

func (p *Provider) GetRoles(workspace string) ([]rubix.Role, error) {

	rows, err := p.primaryConnection.Query("SELECT role, name, description FROM roles WHERE workspace = ? ORDER BY name ASC", workspace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []rubix.Role
	for rows.Next() {

		var role = rubix.Role{Workspace: workspace}
		err = rows.Scan(&role.ID, &role.Name, &role.Description)
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
	p.update()
	return err
}

func (p *Provider) CreateRole(workspace, role, name, description string, permissions, users []string) error {

	_, err := p.primaryConnection.Exec("INSERT INTO roles (workspace, role, name, description) VALUES (?, ?, ?, ?)", workspace, role, name, description)
	p.update()

	if p.isDuplicateConflict(err) {
		return errors.New("role already exists")
	}
	if err != nil {
		return err
	}

	return p.MutateRole(workspace, role, rubix.WithUsersToAdd(users...), rubix.WithPermsToAdd(permissions...))
}

func (p *Provider) MutateRole(workspace, role string, options ...rubix.MutateRoleOption) error {

	if len(options) == 0 {
		return nil
	}
	defer p.update()

	payload := rubix.MutateRolePayload{}
	for _, opt := range options {
		opt(&payload)
	}

	g := errgroup.Group{}
	g.Go(func() error {

		if payload.Title != nil || payload.Description != nil || payload.Constraints != nil {

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
			if payload.Constraints != nil {
				fields = append(fields, "constraints = ?")
				constraintsBytes, err := json.Marshal(*payload.Constraints)
				if err != nil {
					return err
				}

				vals = append(vals, string(constraintsBytes))
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

			if p.isDuplicateConflict(err) {
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

			if p.isDuplicateConflict(err) {
				continue
			}
			if err != nil {
				return err
			}
		}

		return nil
	})
	g.Go(func() error {
		for perm, constraints := range payload.PermConstraintsToAdd {
			constraintsStr, err := json.Marshal(constraints)
			if err != nil {
				return err
			}

			_, err = p.primaryConnection.Exec("UPDATE role_permissions SET constraints = ? WHERE workspace = ? AND role = ? AND permission = ?", string(constraintsStr), workspace, role, perm)
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
