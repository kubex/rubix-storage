package sql

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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
	query := "INSERT INTO auth_data (workspace, user, `vendor`, `app`, `key`, `value`) VALUES (?, ?, ?, ?, ?, ?)"
	if forceUpdate {
		if p.SqlLite {
			query += " ON CONFLICT(workspace, user, `vendor`, `app`, `key`) DO UPDATE SET `value` = excluded.`value`"
		} else {
			query += " ON DUPLICATE KEY UPDATE `value` = ?"
			args = append(args, value.Value)
		}
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

	query := "SELECT rp.permission,rp.resource, rp.allow, r.conditions, rp.options" +
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
	result := make(map[string]permissionResult)
	for rows.Next() {
		newResult := permissionResult{}
		var roleConditionsStr sql.NullString
		var optionsStr sql.NullString
		if err = rows.Scan(&newResult.PermissionKey, &newResult.Resource, &newResult.Allow, &roleConditionsStr, &optionsStr); err != nil {
			return nil, err
		}

		if roleConditionsStr.Valid {
			if err = json.Unmarshal([]byte(roleConditionsStr.String), &newResult.RoleConditions); err != nil {
				return nil, err
			}
		}

		if optionsStr.Valid {
			if err = json.Unmarshal([]byte(optionsStr.String), &newResult.Options); err != nil {
				return nil, err
			}
		}

		if _, ok := result[newResult.PermissionKey]; !ok || !newResult.Allow {
			result[newResult.PermissionKey] = newResult
		} else if newResult.Options != nil && len(newResult.Options) > 0 {
			for key, opt := range newResult.Options {
				if _, ok = result[newResult.PermissionKey].Options[key]; !ok {
					result[newResult.PermissionKey].Options[key] = opt
				} else {
					result[newResult.PermissionKey].Options[key] = append(result[newResult.PermissionKey].Options[key], newResult.Options[key]...)
				}
			}
		}
	}

	var statements []app.PermissionStatement
	for _, res := range result {
		effect := app.PermissionEffectAllow
		if !res.Allow {
			effect = app.PermissionEffectDeny
		} else if !rubix.CheckCondition(res.RoleConditions, lookup) {
			continue
		}

		statements = append(statements, app.PermissionStatement{
			Effect:     effect,
			Permission: app.ScopedKeyFromString(res.PermissionKey),
			Resource:   "",
			Meta:       res.Options,
		})
	}

	return statements, nil
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
				// No change occurred; skip timestamp update
				continue
			}
			if err != nil {
				return err
			}
			// Update membership lastUpdate when user is added to a role
			if _, err := p.primaryConnection.Exec("UPDATE workspace_memberships SET lastUpdate = CURRENT_TIMESTAMP WHERE workspace = ? AND user = ?", workspace, user); err != nil {
				return err
			}
		}

		return nil
	})
	g.Go(func() error {

		for _, role := range payload.RolesToRemove {
			res, err := p.primaryConnection.Exec("DELETE FROM user_roles WHERE workspace = ? AND user = ? AND role = ?", workspace, user, role)
			if err != nil {
				return err
			}
			if rows, _ := res.RowsAffected(); rows > 0 {
				// Update membership lastUpdate when user is removed from a role
				if _, err := p.primaryConnection.Exec("UPDATE workspace_memberships SET lastUpdate = CURRENT_TIMESTAMP WHERE workspace = ? AND user = ?", workspace, user); err != nil {
					return err
				}
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

		row := p.primaryConnection.QueryRow("SELECT name, description, conditions FROM roles WHERE workspace = ? AND role = ?", workspace, role)

		var conditionsStr sql.NullString
		err := row.Scan(&ret.Name, &ret.Description, &conditionsStr)
		if errors.Is(err, sql.ErrNoRows) {
			return rubix.ErrNoResultFound
		}
		if conditionsStr.Valid {
			err = json.Unmarshal([]byte(conditionsStr.String), &ret.Conditions)
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

		rows, err := p.primaryConnection.Query("SELECT permission, resource, allow, options FROM role_permissions WHERE workspace = ? AND role = ?", workspace, role)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var permission = rubix.RolePermission{Workspace: workspace, Role: role}
			var optionsStr sql.NullString
			err = rows.Scan(&permission.Permission, &permission.Resource, &permission.Allow, &optionsStr)
			if err != nil {
				return err
			}

			if optionsStr.Valid {
				err = json.Unmarshal([]byte(optionsStr.String), &permission.Options)
				if err != nil {
					return err
				}
			}

			ret.Permissions = append(ret.Permissions, permission)
		}

		return nil
	})
	g.Go(func() error {
		rows, err := p.primaryConnection.Query("SELECT resource, resource_type FROM role_resources WHERE workspace = ? AND role = ?", workspace, role)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var rr rubix.RoleResource
			var rt string
			if err := rows.Scan(&rr.Resource, &rt); err != nil {
				return err
			}
			rr.Workspace = workspace
			rr.Role = role
			rr.ResourceType = rubix.ResourceType(rt)
			ret.Resources = append(ret.Resources, rr)
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

	roleIDs, err := p.GetUserRoleIDs(workspace, user)
	if err != nil {
		return nil, err
	}

	var roles []rubix.UserRole
	for _, roleID := range roleIDs {
		roles = append(roles, rubix.UserRole{Workspace: workspace, User: user, Role: roleID})
	}

	return roles, nil
}

func (p *Provider) GetUserRoleIDs(workspace, user string) ([]string, error) {

	rows, err := p.primaryConnection.Query("SELECT role FROM user_roles WHERE workspace = ? AND user = ?", workspace, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {

		var role = ""
		err = rows.Scan(&role)
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

func (p *Provider) CreateRole(workspace, role, name, description string, permissions, users []string, conditions rubix.Condition) error {

	_, err := p.primaryConnection.Exec("INSERT INTO roles (workspace, role, name, description) VALUES (?, ?, ?, ?)", workspace, role, name, description)
	p.update()

	if p.isDuplicateConflict(err) {
		return errors.New("role already exists")
	}
	if err != nil {
		return err
	}

	return p.MutateRole(workspace, role, rubix.WithUsersToAdd(users...), rubix.WithPermsToAdd(permissions...), rubix.WithConditions(conditions))
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

		if payload.Title != nil || payload.Description != nil || payload.Conditions != nil {

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
			if payload.Conditions != nil {
				fields = append(fields, "conditions = ?")
				conditionsBytes, err := json.Marshal(*payload.Conditions)
				if err != nil {
					return err
				}

				vals = append(vals, string(conditionsBytes))
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
				// No change
				continue
			}
			if err != nil {
				return err
			}
			// Update membership lastUpdate for the affected user
			if _, err := p.primaryConnection.Exec("UPDATE workspace_memberships SET lastUpdate = CURRENT_TIMESTAMP WHERE workspace = ? AND user = ?", workspace, user); err != nil {
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
			// Update membership lastUpdate for the affected user
			if _, err := p.primaryConnection.Exec("UPDATE workspace_memberships SET lastUpdate = CURRENT_TIMESTAMP WHERE workspace = ? AND user = ?", workspace, user); err != nil {
				return err
			}
		}

		return nil
	})
	g.Go(func() error {

		for _, perm := range payload.PermsToAdd {
			_, err := p.primaryConnection.Exec("INSERT INTO role_permissions (workspace, role, permission) VALUES (?, ?, ?)", workspace, role, perm)

			if p.isDuplicateConflict(err) {
				// no change
				continue
			}
			if err != nil {
				return err
			}
			// bump role lastUpdate as permissions changed
			if _, err := p.primaryConnection.Exec("UPDATE roles SET lastUpdate = CURRENT_TIMESTAMP WHERE workspace = ? AND role = ?", workspace, role); err != nil {
				return err
			}
		}

		return nil
	})
	g.Go(func() error {

		for _, perm := range payload.PermsToRem {
			res, err := p.primaryConnection.Exec("DELETE FROM role_permissions WHERE workspace = ? AND role = ? AND permission = ?", workspace, role, perm)
			if err != nil {
				return err
			}
			if rows, _ := res.RowsAffected(); rows > 0 {
				if _, err := p.primaryConnection.Exec("UPDATE roles SET lastUpdate = CURRENT_TIMESTAMP WHERE workspace = ? AND role = ?", workspace, role); err != nil {
					return err
				}
			}
		}

		return nil
	})
	g.Go(func() error {
		for perm, option := range payload.PermOptionToAdd {
			optionsStr, err := json.Marshal(option)
			if err != nil {
				return err
			}

			res, err := p.primaryConnection.Exec("UPDATE role_permissions SET options = ? WHERE workspace = ? AND role = ? AND permission = ?", string(optionsStr), workspace, role, perm)
			if err != nil {
				return err
			}
			if rows, _ := res.RowsAffected(); rows > 0 {
				if _, err := p.primaryConnection.Exec("UPDATE roles SET lastUpdate = CURRENT_TIMESTAMP WHERE workspace = ? AND role = ?", workspace, role); err != nil {
					return err
				}
			}
		}

		return nil
	})

	return g.Wait()
}

// --- Role Resources ---
func (p *Provider) GetRoleResources(workspace, role string) ([]rubix.RoleResource, error) {
	rows, err := p.primaryConnection.Query("SELECT resource, resource_type FROM role_resources WHERE workspace = ? AND role = ?", workspace, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []rubix.RoleResource
	for rows.Next() {
		var it rubix.RoleResource
		var rt string
		it.Workspace = workspace
		it.Role = role
		if err := rows.Scan(&it.Resource, &rt); err != nil {
			return nil, err
		}
		it.ResourceType = rubix.ResourceType(rt)
		items = append(items, it)
	}
	return items, nil
}

func (p *Provider) AddRoleResources(workspace, role string, resources ...rubix.RoleResource) error {
	if len(resources) == 0 {
		return nil
	}
	defer p.update()
	anyChange := false
	for _, rr := range resources {
		_, err := p.primaryConnection.Exec("INSERT INTO role_resources (workspace, role, resource, resource_type) VALUES (?, ?, ?, ?)", workspace, role, rr.Resource, string(rr.ResourceType))
		if p.isDuplicateConflict(err) {
			continue
		}
		if err != nil {
			return err
		}
		anyChange = true
	}
	if anyChange {
		_, _ = p.primaryConnection.Exec("UPDATE roles SET lastUpdate = CURRENT_TIMESTAMP WHERE workspace = ? AND role = ?", workspace, role)
	}
	return nil
}

func (p *Provider) RemoveRoleResources(workspace, role string, resources ...rubix.RoleResource) error {
	if len(resources) == 0 {
		return nil
	}
	defer p.update()
	anyChange := false
	for _, rr := range resources {
		res, err := p.primaryConnection.Exec("DELETE FROM role_resources WHERE workspace = ? AND role = ? AND resource = ?", workspace, role, rr.Resource)
		if err != nil {
			return err
		}
		if rows, _ := res.RowsAffected(); rows > 0 {
			anyChange = true
		}
	}
	if anyChange {
		_, _ = p.primaryConnection.Exec("UPDATE roles SET lastUpdate = CURRENT_TIMESTAMP WHERE workspace = ? AND role = ?", workspace, role)
	}
	return nil
}

func (p *Provider) GetTeam(workspace, team string) (*rubix.Team, error) {
	var ret = rubix.Team{
		Workspace: workspace,
		ID:        team,
	}

	g := errgroup.Group{}
	g.Go(func() error {
		row := p.primaryConnection.QueryRow("SELECT name, description FROM `teams` WHERE workspace = ? AND `team` = ?", workspace, team)
		return row.Scan(&ret.Name, &ret.Description)
	})
	g.Go(func() error {
		rows, err := p.primaryConnection.Query("SELECT user, level FROM user_teams WHERE workspace = ? AND `team` = ?", workspace, team)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var ug rubix.UserTeam
			ug.Workspace = workspace
			ug.Team = team
			var level string
			if err := rows.Scan(&ug.User, &level); err != nil {
				return err
			}
			ug.Level = rubix.TeamLevel(level)
			ret.Users = append(ret.Users, ug.User)
			ret.Members = append(ret.Members, ug)
		}
		return nil
	})

	return &ret, g.Wait()
}

func (p *Provider) GetTeams(workspace string) ([]rubix.Team, error) {
	rows, err := p.primaryConnection.Query("SELECT `team`, name, description FROM `teams` WHERE workspace = ? ORDER BY name ASC", workspace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var teams []rubix.Team
	for rows.Next() {
		var g rubix.Team
		g.Workspace = workspace
		if err := rows.Scan(&g.ID, &g.Name, &g.Description); err != nil {
			return nil, err
		}
		teams = append(teams, g)
	}
	return teams, nil
}

func (p *Provider) GetUserTeams(workspace, user string) ([]rubix.UserTeam, error) {
	rows, err := p.primaryConnection.Query("SELECT `team`, level FROM user_teams WHERE workspace = ? AND user = ?", workspace, user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var teams []rubix.UserTeam
	for rows.Next() {
		var ug rubix.UserTeam
		ug.Workspace = workspace
		ug.User = user
		var level string
		if err := rows.Scan(&ug.Team, &level); err != nil {
			return nil, err
		}
		ug.Level = rubix.TeamLevel(level)
		teams = append(teams, ug)
	}
	return teams, nil
}

func (p *Provider) DeleteTeam(workspace, team string) error {
	_, err := p.primaryConnection.Exec("DELETE FROM `teams` WHERE workspace = ? AND `team` = ?", workspace, team)
	_, err = p.primaryConnection.Exec("DELETE FROM `user_teams` WHERE workspace = ? AND `team` = ?", workspace, team)
	p.update()
	return err
}

func (p *Provider) CreateTeam(workspace, team, name, description string, users map[string]rubix.TeamLevel) error {
	_, err := p.primaryConnection.Exec("INSERT INTO `teams` (workspace, `team`, name, description) VALUES (?, ?, ?, ?)", workspace, team, name, description)
	p.update()
	if p.isDuplicateConflict(err) {
		return errors.New("team already exists")
	}
	if err != nil {
		return err
	}
	var opts []rubix.MutateTeamOption
	if len(users) > 0 {
		levelBuckets := map[rubix.TeamLevel][]string{}
		for u, lvl := range users {
			levelBuckets[lvl] = append(levelBuckets[lvl], u)
		}
		for lvl, us := range levelBuckets {
			opts = append(opts, rubix.WithTeamUsersToAdd(lvl, us...))
		}
	}
	return p.MutateTeam(workspace, team, opts...)
}

func (p *Provider) MutateTeam(workspace, team string, options ...rubix.MutateTeamOption) error {
	if len(options) == 0 {
		return nil
	}
	defer p.update()
	payload := rubix.MutateTeamPayload{}
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
			vals = append(vals, workspace, team)
			q := fmt.Sprintf("UPDATE `teams` SET %s WHERE workspace = ? AND `team` = ?", strings.Join(fields, ", "))
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
		for user, level := range payload.UsersToAdd {
			_, err := p.primaryConnection.Exec("INSERT INTO user_teams (workspace, user, `team`, level) VALUES (?, ?, ?, ?)", workspace, user, team, string(level))
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
			_, err := p.primaryConnection.Exec("DELETE FROM user_teams WHERE workspace = ? AND user = ? AND `team` = ?", workspace, user, team)
			if err != nil {
				return err
			}
		}
		return nil
	})
	g.Go(func() error {
		for user, level := range payload.UsersLevel {
			_, err := p.primaryConnection.Exec("UPDATE user_teams SET level = ? WHERE workspace = ? AND user = ? AND `team` = ?", string(level), workspace, user, team)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return g.Wait()
}

// --- Brands ---
func (p *Provider) GetBrand(workspace, brand string) (*rubix.Brand, error) {
	ret := &rubix.Brand{Workspace: workspace, ID: brand}
	row := p.primaryConnection.QueryRow("SELECT name, description FROM brands WHERE workspace = ? AND brand = ?", workspace, brand)
	if err := row.Scan(&ret.Name, &ret.Description); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, rubix.ErrNoResultFound
		}
		return nil, err
	}
	return ret, nil
}

func (p *Provider) GetBrands(workspace string) ([]rubix.Brand, error) {
	rows, err := p.primaryConnection.Query("SELECT brand, name, description FROM brands WHERE workspace = ? ORDER BY name ASC", workspace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []rubix.Brand
	for rows.Next() {
		var it rubix.Brand
		it.Workspace = workspace
		if err := rows.Scan(&it.ID, &it.Name, &it.Description); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
}

func (p *Provider) CreateBrand(workspace, brand, name, description string) error {
	_, err := p.primaryConnection.Exec("INSERT INTO brands (workspace, brand, name, description) VALUES (?, ?, ?, ?)", workspace, brand, name, description)
	p.update()
	if p.isDuplicateConflict(err) {
		return errors.New("brand already exists")
	}
	return err
}

func (p *Provider) MutateBrand(workspace, brand string, options ...rubix.MutateBrandOption) error {
	if len(options) == 0 {
		return nil
	}
	defer p.update()
	payload := rubix.MutateBrandPayload{}
	for _, opt := range options {
		opt(&payload)
	}
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
	if len(fields) == 0 {
		return nil
	}
	vals = append(vals, workspace, brand)
	q := fmt.Sprintf("UPDATE brands SET %s WHERE workspace = ? AND brand = ?", strings.Join(fields, ", "))
	res, err := p.primaryConnection.Exec(q, vals...)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return rubix.ErrNoResultFound
	}
	return nil
}

// --- Departments ---
func (p *Provider) GetDepartment(workspace, department string) (*rubix.Department, error) {
	ret := &rubix.Department{Workspace: workspace, ID: department}
	row := p.primaryConnection.QueryRow("SELECT name, description FROM departments WHERE workspace = ? AND department = ?", workspace, department)
	if err := row.Scan(&ret.Name, &ret.Description); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, rubix.ErrNoResultFound
		}
		return nil, err
	}
	return ret, nil
}

func (p *Provider) GetDepartments(workspace string) ([]rubix.Department, error) {
	rows, err := p.primaryConnection.Query("SELECT department, name, description FROM departments WHERE workspace = ? ORDER BY name ASC", workspace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []rubix.Department
	for rows.Next() {
		var it rubix.Department
		it.Workspace = workspace
		if err := rows.Scan(&it.ID, &it.Name, &it.Description); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
}

func (p *Provider) CreateDepartment(workspace, department, name, description string) error {
	_, err := p.primaryConnection.Exec("INSERT INTO departments (workspace, department, name, description) VALUES (?, ?, ?, ?)", workspace, department, name, description)
	p.update()
	if p.isDuplicateConflict(err) {
		return errors.New("department already exists")
	}
	return err
}

func (p *Provider) MutateDepartment(workspace, department string, options ...rubix.MutateDepartmentOption) error {
	if len(options) == 0 {
		return nil
	}
	defer p.update()
	payload := rubix.MutateDepartmentPayload{}
	for _, opt := range options {
		opt(&payload)
	}
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
	if len(fields) == 0 {
		return nil
	}
	vals = append(vals, workspace, department)
	q := fmt.Sprintf("UPDATE departments SET %s WHERE workspace = ? AND department = ?", strings.Join(fields, ", "))
	res, err := p.primaryConnection.Exec(q, vals...)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return rubix.ErrNoResultFound
	}
	return nil
}

// --- Channels ---
func (p *Provider) GetChannel(workspace, channel string) (*rubix.Channel, error) {
	ret := &rubix.Channel{Workspace: workspace, ID: channel}
	row := p.primaryConnection.QueryRow("SELECT department, name, description FROM channels WHERE workspace = ? AND channel = ?", workspace, channel)
	var desc sql.NullString
	if err := row.Scan(&ret.DepartmentID, &ret.Name, &desc); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, rubix.ErrNoResultFound
		}
		return nil, err
	}
	ret.Description = desc.String
	return ret, nil
}

func (p *Provider) GetChannels(workspace string) ([]rubix.Channel, error) {
	rows, err := p.primaryConnection.Query("SELECT channel, department, name, description FROM channels WHERE workspace = ? ORDER BY name ASC", workspace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []rubix.Channel
	for rows.Next() {
		var it rubix.Channel
		var desc sql.NullString
		it.Workspace = workspace
		if err := rows.Scan(&it.ID, &it.DepartmentID, &it.Name, &desc); err != nil {
			return nil, err
		}
		it.Description = desc.String
		items = append(items, it)
	}
	return items, nil
}

func (p *Provider) CreateChannel(workspace, channel, department, name, description string) error {
	_, err := p.primaryConnection.Exec("INSERT INTO channels (workspace, channel, department, name, description) VALUES (?, ?, ?, ?, ?)", workspace, channel, department, name, description)
	p.update()
	if p.isDuplicateConflict(err) {
		return errors.New("channel already exists")
	}
	return err
}

func (p *Provider) MutateChannel(workspace, channel string, options ...rubix.MutateChannelOption) error {
	if len(options) == 0 {
		return nil
	}
	defer p.update()
	payload := rubix.MutateChannelPayload{}
	for _, opt := range options {
		opt(&payload)
	}
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
	if len(fields) == 0 {
		return nil
	}
	vals = append(vals, workspace, channel)
	q := fmt.Sprintf("UPDATE channels SET %s WHERE workspace = ? AND channel = ?", strings.Join(fields, ", "))
	res, err := p.primaryConnection.Exec(q, vals...)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return rubix.ErrNoResultFound
	}
	return nil
}
