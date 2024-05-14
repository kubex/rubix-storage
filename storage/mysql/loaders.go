package mysql

import (
	"database/sql"
	"encoding/json"
	"github.com/kubex/definitions-go/app"
	"github.com/kubex/rubix-storage/rubix"
	"log"
	"strings"
	"time"
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

func (p *Provider) GetWorkspaceUserIDs(workspaceUuid string) ([]string, error) {
	rows, err := p.primaryConnection.Query("SELECT user FROM workspace_memberships WHERE workspace = ?", workspaceUuid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := []string{}
	for rows.Next() {
		var user string
		if err := rows.Scan(&user); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (p *Provider) RetrieveWorkspace(workspaceUuid string) (*rubix.Workspace, error) {
	q := p.primaryConnection.QueryRow("SELECT uuid, alias, domain, name, installedApplications FROM workspaces WHERE uuid = ?", workspaceUuid)
	located := rubix.Workspace{}
	installedApplicationsJson := ""
	err := q.Scan(&located.Uuid, &located.Alias, &located.Domain, &located.Name, &installedApplicationsJson)
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

func (p *Provider) SetUserStatus(workspaceUuid, userUuid string, status rubix.UserStatus) (rubix.UserStatus, bool, error) {
	var expiry *time.Time
	duration := status.ClearAfterSeconds
	if !status.ExpiryTime.IsZero() {
		expiry = &status.ExpiryTime
		if duration == 0 {
			duration = int32(status.ExpiryTime.Sub(time.Now()).Seconds())
		}
	}

	if status.AfterID == status.ID {
		status.AfterID = ""
	}

	var afterId *string
	if status.AfterID != "" {
		afterId = &status.AfterID

		args := []interface{}{workspaceUuid, userUuid}
		queryAppend := ""
		var parentExpiry *time.Time
		if status.AfterID == "latest" {
			queryAppend += "AND id != \"\" ORDER BY expiry DESC"
		} else {
			queryAppend += "AND id = ?"
			args = append(args, status.AfterID)
		}
		qu := p.primaryConnection.QueryRow("SELECT expiry, id FROM user_status WHERE workspace = ? AND user = ? "+queryAppend+" LIMIT 1", args...)
		err := qu.Scan(&parentExpiry, &status.AfterID)
		if err == nil && parentExpiry != nil && parentExpiry.After(time.Now()) && duration > 0 {
			newExp := parentExpiry.Add(time.Duration(duration) * time.Second)
			expiry = &newExp
			status.ExpiryTime = newExp
		}
	}

	res, err := p.primaryConnection.Exec("INSERT INTO user_status (workspace, user, state, extendedState, expiry, applied, id, afterId, duration, clearOnLogout) "+
		"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) "+
		"ON DUPLICATE KEY UPDATE "+
		"state = ?, extendedState = ?, expiry = ?, applied = ?, afterId = ?, duration = ?, clearOnLogout = ?",
		workspaceUuid, userUuid, status.State, status.ExtendedState, expiry, time.Now(), status.ID, afterId, duration, status.ClearOnLogout,
		status.State, status.ExtendedState, expiry, time.Now(), afterId, duration, status.ClearOnLogout)
	if err != nil {
		return status, false, err
	}
	impact, err := res.RowsAffected()
	return status, impact > 0, err
}

func (p *Provider) ClearUserStatusID(workspaceUuid, userUuid, statusID string) error {
	_, updateErr := p.primaryConnection.Exec("UPDATE user_status SET expiry = DATE_ADD(expiry,INTERVAL duration SECOND) WHERE workspace = ? AND user = ? AND expiry > ? AND afterId = ? AND duration > 0", workspaceUuid, userUuid, time.Now(), statusID)
	if updateErr != nil {
		return updateErr
	}

	_, deleteErr := p.primaryConnection.Exec("DELETE FROM user_status  WHERE workspace = ? AND user = ? AND id = ?", workspaceUuid, userUuid, statusID)
	return deleteErr
}

func (p *Provider) GetUserStatus(workspaceUuid, userUuid string) (rubix.UserStatus, error) {
	status := rubix.UserStatus{}
	var expiry *time.Time
	rows, err := p.primaryConnection.Query("SELECT state, extendedState, applied, expiry, id, afterId FROM user_status WHERE workspace = ? AND user = ? AND expiry > ?", workspaceUuid, userUuid, time.Now())
	if err != nil {
		return status, err
	}
	defer rows.Close()

	for rows.Next() {
		newResult := rubix.UserStatus{}
		afterId := sql.NullString{}
		if scanErr := rows.Scan(&newResult.State, &newResult.ExtendedState, &newResult.AppliedTime, &expiry, &newResult.ID, &afterId); scanErr != nil {
			return status, scanErr
		}
		if afterId.Valid {
			newResult.AfterID = afterId.String
		}

		if expiry != nil {
			newResult.ExpiryTime = *expiry
		}

		if newResult.ID == "" {
			status.AppliedTime = newResult.AppliedTime
			status.ExpiryTime = newResult.ExpiryTime
			status.State = newResult.State
			status.ExtendedState = newResult.ExtendedState
		} else {
			status.Overlays = append(status.Overlays, newResult)
		}
	}

	return status, nil
}
