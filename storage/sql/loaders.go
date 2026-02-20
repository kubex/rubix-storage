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
	rows, err := p.primaryConnection.Query("SELECT uuid, alias, domain, name, icon, installedApplications,defaultApp,systemVendors,footerParts,accessCondition,emailDomainWhitelist FROM workspaces WHERE "+where, args...)
	if err != nil {
		return resp, err
	}

	defer rows.Close()
	for rows.Next() {
		located := rubix.Workspace{}
		installedApplicationsJson := sql.NullString{}
		footerPartsJson := sql.NullString{}
		accessConditionJson := sql.NullString{}
		emailDomainWhitelistJson := sql.NullString{}
		sysVendors := sql.NullString{}
		icon := sql.NullString{}
		defaultApp := sql.NullString{}
		scanErr := rows.Scan(&located.Uuid, &located.Alias, &located.Domain, &located.Name, &icon, &installedApplicationsJson, &defaultApp, &sysVendors, &footerPartsJson, &accessConditionJson, &emailDomainWhitelistJson)
		if scanErr != nil {
			continue
		}
		located.SystemVendors = strings.Split(sysVendors.String, ",")
		located.Icon = icon.String
		located.DefaultApp = app.IDFromString(defaultApp.String)
		json.Unmarshal([]byte(installedApplicationsJson.String), &located.InstalledApplications)
		json.Unmarshal([]byte(footerPartsJson.String), &located.FooterParts)
		json.Unmarshal([]byte(accessConditionJson.String), &located.AccessCondition)
		json.Unmarshal([]byte(emailDomainWhitelistJson.String), &located.EmailDomainWhitelist)
		resp[located.Uuid] = &located
	}

	return resp, err
}

func (p *Provider) SetWorkspaceAccessCondition(workspaceUuid string, condition rubix.Condition) error {
	conditionBytes, err := json.Marshal(condition)
	if err != nil {
		return err
	}
	_, err = p.primaryConnection.Exec("UPDATE workspaces SET accessCondition = ? WHERE uuid = ?", string(conditionBytes), workspaceUuid)
	if err != nil {
		return err
	}
	p.update()
	return nil
}

func (p *Provider) SetWorkspaceEmailDomainWhitelist(workspaceUuid string, domains []string) error {
	domainsBytes, err := json.Marshal(domains)
	if err != nil {
		return err
	}
	_, err = p.primaryConnection.Exec("UPDATE workspaces SET emailDomainWhitelist = ? WHERE uuid = ?", string(domainsBytes), workspaceUuid)
	if err != nil {
		return err
	}
	p.update()
	return nil
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

	// Update user name/email in users table
	if payload.Name != nil || payload.Email != nil {
		g.Go(func() error {
			var fields []string
			var vals []any
			if payload.Name != nil {
				fields = append(fields, "name = ?")
				vals = append(vals, *payload.Name)
			}
			if payload.Email != nil {
				fields = append(fields, "email = ?")
				vals = append(vals, *payload.Email)
			}
			if len(fields) > 0 {
				vals = append(vals, user)
				_, err := p.primaryConnection.Exec("UPDATE users SET "+strings.Join(fields, ", ")+" WHERE user = ?", vals...)
				return err
			}
			return nil
		})
	}

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

func (p *Provider) GetRolePermissions(workspace, role string) ([]rubix.RolePermission, error) {
	rows, err := p.primaryConnection.Query("SELECT permission, resource, allow, options FROM role_permissions WHERE workspace = ? AND role = ?", workspace, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []rubix.RolePermission
	for rows.Next() {
		var rp = rubix.RolePermission{Workspace: workspace, Role: role}
		var optionsStr sql.NullString
		if err := rows.Scan(&rp.Permission, &rp.Resource, &rp.Allow, &optionsStr); err != nil {
			return nil, err
		}
		if optionsStr.Valid {
			if err := json.Unmarshal([]byte(optionsStr.String), &rp.Options); err != nil {
				return nil, err
			}
		}
		perms = append(perms, rp)
	}
	return perms, nil
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
	row := p.primaryConnection.QueryRow("SELECT department, name, description, maxLevel FROM channels WHERE workspace = ? AND channel = ?", workspace, channel)
	var desc sql.NullString
	if err := row.Scan(&ret.DepartmentID, &ret.Name, &desc, &ret.MaxLevel); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, rubix.ErrNoResultFound
		}
		return nil, err
	}
	ret.Description = desc.String
	return ret, nil
}

func (p *Provider) GetChannels(workspace string) ([]rubix.Channel, error) {
	rows, err := p.primaryConnection.Query("SELECT channel, department, name, description, maxLevel FROM channels WHERE workspace = ? ORDER BY name ASC", workspace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []rubix.Channel
	for rows.Next() {
		var it rubix.Channel
		var desc sql.NullString
		it.Workspace = workspace
		if err := rows.Scan(&it.ID, &it.DepartmentID, &it.Name, &desc, &it.MaxLevel); err != nil {
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
	if payload.MaxLevel != nil {
		fields = append(fields, "maxLevel = ?")
		vals = append(vals, *payload.MaxLevel)
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

// --- Distributors ---
func (p *Provider) GetDistributor(workspace, distributor string) (*rubix.Distributor, error) {
	ret := &rubix.Distributor{Workspace: workspace, ID: distributor}
	row := p.primaryConnection.QueryRow("SELECT name, description, website_url, logo_url FROM distributors WHERE workspace = ? AND distributor = ?", workspace, distributor)
	if err := row.Scan(&ret.Name, &ret.Description, &ret.WebsiteURL, &ret.LogoURL); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, rubix.ErrNoResultFound
		}
		return nil, err
	}
	return ret, nil
}

func (p *Provider) GetDistributors(workspace string) ([]rubix.Distributor, error) {
	rows, err := p.primaryConnection.Query("SELECT distributor, name, description, website_url, logo_url FROM distributors WHERE workspace = ? ORDER BY name ASC", workspace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []rubix.Distributor
	for rows.Next() {
		var it rubix.Distributor
		it.Workspace = workspace
		if err := rows.Scan(&it.ID, &it.Name, &it.Description, &it.WebsiteURL, &it.LogoURL); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
}

func (p *Provider) CreateDistributor(workspace, distributor, name, description string) error {
	_, err := p.primaryConnection.Exec("INSERT INTO distributors (workspace, distributor, name, description) VALUES (?, ?, ?, ?)", workspace, distributor, name, description)
	p.update()
	if p.isDuplicateConflict(err) {
		return errors.New("distributor already exists")
	}
	return err
}

func (p *Provider) MutateDistributor(workspace, distributor string, options ...rubix.MutateDistributorOption) error {
	if len(options) == 0 {
		return nil
	}
	defer p.update()
	payload := rubix.MutateDistributorPayload{}
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
	if payload.WebsiteURL != nil {
		fields = append(fields, "website_url = ?")
		vals = append(vals, *payload.WebsiteURL)
	}
	if payload.LogoURL != nil {
		fields = append(fields, "logo_url = ?")
		vals = append(vals, *payload.LogoURL)
	}
	if len(fields) == 0 {
		return nil
	}
	vals = append(vals, workspace, distributor)
	q := fmt.Sprintf("UPDATE distributors SET %s WHERE workspace = ? AND distributor = ?", strings.Join(fields, ", "))
	res, err := p.primaryConnection.Exec(q, vals...)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return rubix.ErrNoResultFound
	}
	return nil
}

// --- BPOs ---
func (p *Provider) GetBPO(workspace, bpo string) (*rubix.BPO, error) {
	ret := &rubix.BPO{Workspace: workspace, ID: bpo}
	row := p.primaryConnection.QueryRow("SELECT name, description, website_url, logo_url FROM bpos WHERE workspace = ? AND bpo = ?", workspace, bpo)
	if err := row.Scan(&ret.Name, &ret.Description, &ret.WebsiteURL, &ret.LogoURL); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, rubix.ErrNoResultFound
		}
		return nil, err
	}
	return ret, nil
}

func (p *Provider) GetBPOs(workspace string) ([]rubix.BPO, error) {
	rows, err := p.primaryConnection.Query("SELECT bpo, name, description, website_url, logo_url FROM bpos WHERE workspace = ? ORDER BY name ASC", workspace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []rubix.BPO
	for rows.Next() {
		var it rubix.BPO
		it.Workspace = workspace
		if err := rows.Scan(&it.ID, &it.Name, &it.Description, &it.WebsiteURL, &it.LogoURL); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
}

func (p *Provider) CreateBPO(workspace, bpo, name, description string) error {
	_, err := p.primaryConnection.Exec("INSERT INTO bpos (workspace, bpo, name, description) VALUES (?, ?, ?, ?)", workspace, bpo, name, description)
	p.update()
	if p.isDuplicateConflict(err) {
		return errors.New("bpo already exists")
	}
	return err
}

func (p *Provider) MutateBPO(workspace, bpo string, options ...rubix.MutateBPOOption) error {
	if len(options) == 0 {
		return nil
	}
	defer p.update()
	payload := rubix.MutateBPOPayload{}
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
	if payload.WebsiteURL != nil {
		fields = append(fields, "website_url = ?")
		vals = append(vals, *payload.WebsiteURL)
	}
	if payload.LogoURL != nil {
		fields = append(fields, "logo_url = ?")
		vals = append(vals, *payload.LogoURL)
	}
	if len(fields) == 0 {
		return nil
	}
	vals = append(vals, workspace, bpo)
	q := fmt.Sprintf("UPDATE bpos SET %s WHERE workspace = ? AND bpo = ?", strings.Join(fields, ", "))
	res, err := p.primaryConnection.Exec(q, vals...)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return rubix.ErrNoResultFound
	}
	return nil
}

// --- OIDC Providers ---
func (p *Provider) GetOIDCProviders(workspace string) ([]rubix.OIDCProvider, error) {
	rows, err := p.primaryConnection.Query(
		"SELECT uuid, workspace, providerName, displayName, clientID, clientSecret, clientKeys, issuerURL, bpoID, scimEnabled, scimBearerToken, scimSyncTeams, scimSyncRoles, scimAutoCreate FROM workspace_oidc_providers WHERE workspace = ?",
		workspace,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []rubix.OIDCProvider
	for rows.Next() {
		var it rubix.OIDCProvider
		clientSecret := sql.NullString{}
		clientKeys := sql.NullString{}
		if err := rows.Scan(&it.Uuid, &it.Workspace, &it.ProviderName, &it.DisplayName, &it.ClientID, &clientSecret, &clientKeys, &it.IssuerURL, &it.BpoID, &it.ScimEnabled, &it.ScimBearerToken, &it.ScimSyncTeams, &it.ScimSyncRoles, &it.ScimAutoCreate); err != nil {
			return nil, err
		}
		it.ClientSecret = clientSecret.String
		it.ClientKeys = clientKeys.String
		items = append(items, it)
	}
	return items, nil
}

func (p *Provider) GetOIDCProvider(workspace, uuid string) (*rubix.OIDCProvider, error) {
	row := p.primaryConnection.QueryRow(
		"SELECT uuid, workspace, providerName, displayName, clientID, clientSecret, clientKeys, issuerURL, bpoID, scimEnabled, scimBearerToken, scimSyncTeams, scimSyncRoles, scimAutoCreate FROM workspace_oidc_providers WHERE workspace = ? AND uuid = ?",
		workspace, uuid,
	)
	var it rubix.OIDCProvider
	clientSecret := sql.NullString{}
	clientKeys := sql.NullString{}
	if err := row.Scan(&it.Uuid, &it.Workspace, &it.ProviderName, &it.DisplayName, &it.ClientID, &clientSecret, &clientKeys, &it.IssuerURL, &it.BpoID, &it.ScimEnabled, &it.ScimBearerToken, &it.ScimSyncTeams, &it.ScimSyncRoles, &it.ScimAutoCreate); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, rubix.ErrNoResultFound
		}
		return nil, err
	}
	it.ClientSecret = clientSecret.String
	it.ClientKeys = clientKeys.String
	return &it, nil
}

func (p *Provider) CreateOIDCProvider(workspace string, provider rubix.OIDCProvider) error {
	_, err := p.primaryConnection.Exec(
		"INSERT INTO workspace_oidc_providers (uuid, workspace, providerName, displayName, clientID, clientSecret, clientKeys, issuerURL, bpoID, scimEnabled, scimBearerToken, scimSyncTeams, scimSyncRoles, scimAutoCreate) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		provider.Uuid, workspace, provider.ProviderName, provider.DisplayName, provider.ClientID, provider.ClientSecret, provider.ClientKeys, provider.IssuerURL, provider.BpoID, provider.ScimEnabled, provider.ScimBearerToken, provider.ScimSyncTeams, provider.ScimSyncRoles, provider.ScimAutoCreate,
	)
	if p.isDuplicateConflict(err) {
		return rubix.ErrDuplicate
	}
	if err != nil {
		return err
	}
	p.update()
	return nil
}

func (p *Provider) MutateOIDCProvider(workspace, uuid string, options ...rubix.MutateOIDCProviderOption) error {
	if len(options) == 0 {
		return nil
	}
	defer p.update()
	payload := rubix.MutateOIDCProviderPayload{}
	for _, opt := range options {
		opt(&payload)
	}
	var fields []string
	var vals []any
	if payload.ProviderName != nil {
		fields = append(fields, "providerName = ?")
		vals = append(vals, *payload.ProviderName)
	}
	if payload.DisplayName != nil {
		fields = append(fields, "displayName = ?")
		vals = append(vals, *payload.DisplayName)
	}
	if payload.ClientID != nil {
		fields = append(fields, "clientID = ?")
		vals = append(vals, *payload.ClientID)
	}
	if payload.ClientSecret != nil {
		fields = append(fields, "clientSecret = ?")
		vals = append(vals, *payload.ClientSecret)
	}
	if payload.ClientKeys != nil {
		fields = append(fields, "clientKeys = ?")
		vals = append(vals, *payload.ClientKeys)
	}
	if payload.IssuerURL != nil {
		fields = append(fields, "issuerURL = ?")
		vals = append(vals, *payload.IssuerURL)
	}
	if payload.BpoID != nil {
		fields = append(fields, "bpoID = ?")
		vals = append(vals, *payload.BpoID)
	}
	if payload.ScimEnabled != nil {
		fields = append(fields, "scimEnabled = ?")
		vals = append(vals, *payload.ScimEnabled)
	}
	if payload.ScimBearerToken != nil {
		fields = append(fields, "scimBearerToken = ?")
		vals = append(vals, *payload.ScimBearerToken)
	}
	if payload.ScimSyncTeams != nil {
		fields = append(fields, "scimSyncTeams = ?")
		vals = append(vals, *payload.ScimSyncTeams)
	}
	if payload.ScimSyncRoles != nil {
		fields = append(fields, "scimSyncRoles = ?")
		vals = append(vals, *payload.ScimSyncRoles)
	}
	if payload.ScimAutoCreate != nil {
		fields = append(fields, "scimAutoCreate = ?")
		vals = append(vals, *payload.ScimAutoCreate)
	}
	if len(fields) == 0 {
		return nil
	}
	vals = append(vals, workspace, uuid)
	q := fmt.Sprintf("UPDATE workspace_oidc_providers SET %s WHERE workspace = ? AND uuid = ?", strings.Join(fields, ", "))
	res, err := p.primaryConnection.Exec(q, vals...)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return rubix.ErrNoResultFound
	}
	return nil
}

func (p *Provider) DeleteOIDCProvider(workspace, uuid string) error {
	_, err := p.primaryConnection.Exec("DELETE FROM workspace_oidc_providers WHERE workspace = ? AND uuid = ?", workspace, uuid)
	if err != nil {
		return err
	}
	p.update()
	return nil
}

// --- SCIM Group Mappings ---
func (p *Provider) GetSCIMGroupMappings(workspace, providerUUID string) ([]rubix.SCIMGroupMapping, error) {
	rows, err := p.primaryConnection.Query(
		"SELECT providerUUID, scimGroupID, scimGroupName, rubixTeamID, defaultLevel FROM scim_group_mappings WHERE providerUUID = ?",
		providerUUID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []rubix.SCIMGroupMapping
	for rows.Next() {
		var it rubix.SCIMGroupMapping
		if err := rows.Scan(&it.ProviderUUID, &it.ScimGroupID, &it.ScimGroupName, &it.RubixTeamID, &it.DefaultLevel); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
}

func (p *Provider) SetSCIMGroupMapping(workspace string, mapping rubix.SCIMGroupMapping) error {
	args := []any{mapping.ProviderUUID, mapping.ScimGroupID, mapping.ScimGroupName, mapping.RubixTeamID, mapping.DefaultLevel}
	query := "INSERT INTO scim_group_mappings (providerUUID, scimGroupID, scimGroupName, rubixTeamID, defaultLevel) VALUES (?, ?, ?, ?, ?)"
	if p.SqlLite {
		query += " ON CONFLICT(providerUUID, scimGroupID) DO UPDATE SET scimGroupName = excluded.scimGroupName, rubixTeamID = excluded.rubixTeamID, defaultLevel = excluded.defaultLevel"
	} else {
		query += " ON DUPLICATE KEY UPDATE scimGroupName = ?, rubixTeamID = ?, defaultLevel = ?"
		args = append(args, mapping.ScimGroupName, mapping.RubixTeamID, mapping.DefaultLevel)
	}
	_, err := p.primaryConnection.Exec(query, args...)
	if err != nil {
		return err
	}
	p.update()
	return nil
}

func (p *Provider) DeleteSCIMGroupMapping(workspace, providerUUID, scimGroupID string) error {
	_, err := p.primaryConnection.Exec("DELETE FROM scim_group_mappings WHERE providerUUID = ? AND scimGroupID = ?", providerUUID, scimGroupID)
	if err != nil {
		return err
	}
	p.update()
	return nil
}

// --- SCIM Role Mappings ---
func (p *Provider) GetSCIMRoleMappings(workspace, providerUUID string) ([]rubix.SCIMRoleMapping, error) {
	rows, err := p.primaryConnection.Query(
		"SELECT providerUUID, scimAttribute, rubixRoleID FROM scim_role_mappings WHERE providerUUID = ?",
		providerUUID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []rubix.SCIMRoleMapping
	for rows.Next() {
		var it rubix.SCIMRoleMapping
		if err := rows.Scan(&it.ProviderUUID, &it.ScimAttribute, &it.RubixRoleID); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
}

func (p *Provider) SetSCIMRoleMapping(workspace string, mapping rubix.SCIMRoleMapping) error {
	args := []any{mapping.ProviderUUID, mapping.ScimAttribute, mapping.RubixRoleID}
	query := "INSERT INTO scim_role_mappings (providerUUID, scimAttribute, rubixRoleID) VALUES (?, ?, ?)"
	if p.SqlLite {
		query += " ON CONFLICT(providerUUID, scimAttribute) DO UPDATE SET rubixRoleID = excluded.rubixRoleID"
	} else {
		query += " ON DUPLICATE KEY UPDATE rubixRoleID = ?"
		args = append(args, mapping.RubixRoleID)
	}
	_, err := p.primaryConnection.Exec(query, args...)
	if err != nil {
		return err
	}
	p.update()
	return nil
}

func (p *Provider) DeleteSCIMRoleMapping(workspace, providerUUID, scimAttribute string) error {
	_, err := p.primaryConnection.Exec("DELETE FROM scim_role_mappings WHERE providerUUID = ? AND scimAttribute = ?", providerUUID, scimAttribute)
	if err != nil {
		return err
	}
	p.update()
	return nil
}

// --- SCIM Activity Log ---
func (p *Provider) GetSCIMActivityLog(workspace, providerUUID string, limit int) ([]rubix.SCIMActivityLog, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := p.primaryConnection.Query(
		"SELECT id, providerUUID, workspace, timestamp, operation, resource, resourceID, status, detail FROM scim_activity_log WHERE providerUUID = ? ORDER BY id DESC LIMIT ?",
		providerUUID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []rubix.SCIMActivityLog
	for rows.Next() {
		var it rubix.SCIMActivityLog
		detail := sql.NullString{}
		if err := rows.Scan(&it.ID, &it.ProviderUUID, &it.Workspace, &it.Timestamp, &it.Operation, &it.Resource, &it.ResourceID, &it.Status, &detail); err != nil {
			return nil, err
		}
		it.Detail = detail.String
		items = append(items, it)
	}
	return items, nil
}

func (p *Provider) AddSCIMActivityLog(workspace string, entry rubix.SCIMActivityLog) error {
	detail := sql.NullString{}
	if entry.Detail != "" {
		detail.String = entry.Detail
		detail.Valid = true
	}
	_, err := p.primaryConnection.Exec(
		"INSERT INTO scim_activity_log (id, providerUUID, workspace, operation, resource, resourceID, status, detail) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		entry.ID, entry.ProviderUUID, entry.Workspace, entry.Operation, entry.Resource, entry.ResourceID, entry.Status, detail,
	)
	return err
}

func (p *Provider) GetSettings(workspace, vendor, app string, keys ...string) ([]rubix.Setting, error) {
	var conditions []string
	var args []any

	conditions = append(conditions, "workspace = ?")
	args = append(args, workspace)

	if vendor != "" {
		conditions = append(conditions, "vendor = ?")
		args = append(args, vendor)
	}

	if app != "" {
		conditions = append(conditions, "app = ?")
		args = append(args, app)
	} else if vendor != "" {
		// Only filter by NULL app if vendor is specified (to get vendor-level settings)
		conditions = append(conditions, "app IS NULL")
	}

	if len(keys) > 0 {
		var placeholders []string
		for _, key := range keys {
			placeholders = append(placeholders, "?")
			args = append(args, key)
		}
		conditions = append(conditions, "`key` IN ("+strings.Join(placeholders, ",")+")")
	}

	query := "SELECT workspace, vendor, app, `key`, `value` FROM settings WHERE " + strings.Join(conditions, " AND ") + " ORDER BY vendor ASC, app ASC, `key` ASC"

	rows, err := p.primaryConnection.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []rubix.Setting
	for rows.Next() {
		var setting rubix.Setting
		var appID sql.NullString
		if err := rows.Scan(&setting.Workspace, &setting.Vendor, &appID, &setting.Key, &setting.Value); err != nil {
			return nil, err
		}
		setting.App = appID.String
		settings = append(settings, setting)
	}

	return settings, nil
}

func (p *Provider) SetSetting(workspace, vendor, app, key, value string) error {
	appID := sql.NullString{}
	if app != "" {
		appID.String = app
		appID.Valid = true
	}

	args := []any{workspace, vendor, appID, key, value}
	query := "INSERT INTO settings (workspace, vendor, app, `key`, `value`) VALUES (?, ?, ?, ?, ?)"
	if p.SqlLite {
		query += " ON CONFLICT(workspace, vendor, app, `key`) DO UPDATE SET `value` = excluded.`value`"
	} else {
		query += " ON DUPLICATE KEY UPDATE `value` = ?"
		args = append(args, value)
	}

	_, err := p.primaryConnection.Exec(query, args...)
	if err != nil {
		return err
	}
	p.update()
	return nil
}

// Workspace User CRUD

func (p *Provider) CreateWorkspaceUser(workspace string, user rubix.WorkspaceUser) error {
	_, err := p.primaryConnection.Exec(
		"INSERT INTO workspace_users (user_id, workspace, name, email, oidc_provider, scim_managed, auto_created, last_sync_time, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		user.UserID, workspace, user.Name, user.Email, user.OIDCProvider, user.SCIMManaged, user.AutoCreated, user.LastSyncTime, user.CreatedAt,
	)
	if p.isDuplicateConflict(err) {
		return rubix.ErrDuplicate
	}
	if err != nil {
		return err
	}
	p.update()
	return nil
}

func (p *Provider) GetWorkspaceUser(workspace, userID string) (*rubix.WorkspaceUser, error) {
	row := p.primaryConnection.QueryRow(
		"SELECT user_id, workspace, name, email, oidc_provider, scim_managed, auto_created, last_sync_time, created_at FROM workspace_users WHERE workspace = ? AND user_id = ?",
		workspace, userID,
	)
	var it rubix.WorkspaceUser
	name := sql.NullString{}
	email := sql.NullString{}
	lastSync := sql.NullString{}
	createdAt := sql.NullString{}
	if err := row.Scan(&it.UserID, &it.Workspace, &name, &email, &it.OIDCProvider, &it.SCIMManaged, &it.AutoCreated, &lastSync, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, rubix.ErrNoResultFound
		}
		return nil, err
	}
	it.Name = name.String
	it.Email = email.String
	if lastSync.Valid && lastSync.String != "" {
		it.LastSyncTime, _ = time.Parse(time.RFC3339Nano, lastSync.String)
	}
	if createdAt.Valid && createdAt.String != "" {
		it.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt.String)
	}
	return &it, nil
}

func (p *Provider) GetWorkspaceUsersByProvider(workspace, providerUUID string) ([]rubix.WorkspaceUser, error) {
	rows, err := p.primaryConnection.Query(
		"SELECT user_id, workspace, name, email, oidc_provider, scim_managed, auto_created, last_sync_time, created_at FROM workspace_users WHERE workspace = ? AND oidc_provider = ?",
		workspace, providerUUID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []rubix.WorkspaceUser
	for rows.Next() {
		var it rubix.WorkspaceUser
		name := sql.NullString{}
		email := sql.NullString{}
		lastSync := sql.NullString{}
		createdAt := sql.NullString{}
		if err := rows.Scan(&it.UserID, &it.Workspace, &name, &email, &it.OIDCProvider, &it.SCIMManaged, &it.AutoCreated, &lastSync, &createdAt); err != nil {
			return nil, err
		}
		it.Name = name.String
		it.Email = email.String
		if lastSync.Valid && lastSync.String != "" {
			it.LastSyncTime, _ = time.Parse(time.RFC3339Nano, lastSync.String)
		}
		if createdAt.Valid && createdAt.String != "" {
			it.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt.String)
		}
		items = append(items, it)
	}
	return items, nil
}

func (p *Provider) UpdateWorkspaceUser(workspace, userID string, opts ...rubix.MutateWorkspaceUserOption) error {
	if len(opts) == 0 {
		return nil
	}
	defer p.update()
	payload := rubix.MutateWorkspaceUserPayload{}
	for _, opt := range opts {
		opt(&payload)
	}
	var fields []string
	var vals []any
	if payload.Name != nil {
		fields = append(fields, "name = ?")
		vals = append(vals, *payload.Name)
	}
	if payload.Email != nil {
		fields = append(fields, "email = ?")
		vals = append(vals, *payload.Email)
	}
	if payload.SCIMManaged != nil {
		fields = append(fields, "scim_managed = ?")
		vals = append(vals, *payload.SCIMManaged)
	}
	if payload.AutoCreated != nil {
		fields = append(fields, "auto_created = ?")
		vals = append(vals, *payload.AutoCreated)
	}
	if payload.LastSyncTime != nil {
		fields = append(fields, "last_sync_time = ?")
		vals = append(vals, *payload.LastSyncTime)
	}
	if len(fields) == 0 {
		return nil
	}
	vals = append(vals, workspace, userID)
	q := fmt.Sprintf("UPDATE workspace_users SET %s WHERE workspace = ? AND user_id = ?", strings.Join(fields, ", "))
	res, err := p.primaryConnection.Exec(q, vals...)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return rubix.ErrNoResultFound
	}
	return nil
}

func (p *Provider) DeleteWorkspaceUser(workspace, userID string) error {
	_, err := p.primaryConnection.Exec("DELETE FROM workspace_users WHERE workspace = ? AND user_id = ?", workspace, userID)
	if err != nil {
		return err
	}
	p.update()
	return nil
}

// MigrateOIDCUsersToWorkspaceUsers migrates existing OIDC users from the global users table
// to workspace_users. This is idempotent and safe to run multiple times.
func (p *Provider) MigrateOIDCUsersToWorkspaceUsers() error {
	// Find all users with oidc_ prefix
	rows, err := p.primaryConnection.Query("SELECT user, name, email FROM users WHERE user LIKE 'oidc_%'")
	if err != nil {
		return err
	}
	defer rows.Close()

	type oidcUser struct {
		userID string
		name   string
		email  string
	}

	var users []oidcUser
	for rows.Next() {
		var u oidcUser
		name := sql.NullString{}
		email := sql.NullString{}
		if err := rows.Scan(&u.userID, &name, &email); err != nil {
			return err
		}
		u.name = name.String
		u.email = email.String
		users = append(users, u)
	}

	for _, u := range users {
		// Extract provider UUID from oidc_{providerUUID}_{sub}
		parts := strings.SplitN(strings.TrimPrefix(u.userID, "oidc_"), "_", 2)
		if len(parts) != 2 {
			continue
		}
		providerUUID := parts[0]

		// Find workspace from membership
		wsRows, err := p.primaryConnection.Query("SELECT workspace FROM workspace_memberships WHERE user = ?", u.userID)
		if err != nil {
			continue
		}

		for wsRows.Next() {
			var workspace string
			if err := wsRows.Scan(&workspace); err != nil {
				continue
			}

			// Check if provider exists and if SCIM is enabled
			provider, provErr := p.GetOIDCProvider(workspace, providerUUID)
			scimManaged := false
			if provErr == nil && provider != nil {
				scimManaged = provider.ScimEnabled
			}

			// Insert into workspace_users (idempotent)
			_, err := p.primaryConnection.Exec(
				"INSERT INTO workspace_users (user_id, workspace, name, email, oidc_provider, scim_managed, auto_created, created_at) VALUES (?, ?, ?, ?, ?, ?, 1, CURRENT_TIMESTAMP)",
				u.userID, workspace, u.name, u.email, providerUUID, scimManaged,
			)
			if p.isDuplicateConflict(err) {
				continue // Already migrated
			}
		}
		wsRows.Close()

		// Remove from global users table
		_, _ = p.primaryConnection.Exec("DELETE FROM users WHERE user = ? AND user LIKE 'oidc_%'", u.userID)
	}

	p.update()
	return nil
}

func (p *Provider) GetResolvedMembers(workspace string, filter rubix.MemberFilter) ([]rubix.ResolvedMember, error) {
	// Get all memberships (or filtered by user IDs)
	members, err := p.GetWorkspaceMembers(workspace, filter.UserIDs...)
	if err != nil {
		return nil, err
	}

	// Partition into native and OIDC user IDs
	var oidcIDs []string
	for _, m := range members {
		if strings.HasPrefix(m.UserID, "oidc_") {
			oidcIDs = append(oidcIDs, m.UserID)
		}
	}

	// Build lookup map for OIDC users from workspace_users
	oidcUserMap := make(map[string]rubix.WorkspaceUser)
	if len(oidcIDs) > 0 {
		for _, uid := range oidcIDs {
			wu, wuErr := p.GetWorkspaceUser(workspace, uid)
			if wuErr == nil && wu != nil {
				oidcUserMap[uid] = *wu
			}
		}
	}

	var resolved []rubix.ResolvedMember
	for _, m := range members {
		rm := rubix.ResolvedMember{Membership: m}

		if wu, ok := oidcUserMap[m.UserID]; ok {
			rm.Source = "oidc"
			rm.ProviderID = wu.OIDCProvider
			rm.SCIMManaged = wu.SCIMManaged
			rm.AutoCreated = wu.AutoCreated
			rm.LastSync = wu.LastSyncTime
			if wu.Name != "" {
				rm.Name = wu.Name
			}
			if wu.Email != "" {
				rm.Email = wu.Email
			}
		} else if strings.HasPrefix(m.UserID, "oidc_") {
			rm.Source = "oidc"
		} else {
			rm.Source = "native"
		}

		// Apply filters
		if filter.Source != "" && filter.Source != rm.Source {
			continue
		}
		if filter.ProviderUUID != "" && rm.ProviderID != filter.ProviderUUID {
			continue
		}

		resolved = append(resolved, rm)
	}

	return resolved, nil
}
