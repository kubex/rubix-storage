package mysql

import (
	"database/sql"
	"errors"
	"time"

	"github.com/kubex/rubix-storage/rubix"
)

func (p *Provider) SetUserStatus(workspaceUuid, userUuid string, status rubix.UserStatus) (bool, error) {
	var expiry *time.Time
	duration := status.ClearAfterSeconds
	if !status.ExpiryTime.IsZero() {
		expiry = &status.ExpiryTime
		if duration == 0 {
			duration = int32(time.Until(status.ExpiryTime).Seconds())
		}
	}

	var afterId *string
	if status.AfterID != "" {

		if status.AfterID == "latest" {
			latest := p.primaryConnection.QueryRow("SELECT id,expiry FROM user_status WHERE workspace = ? AND user = ? AND id != '' AND expiry IS NOT NULL ORDER BY expiry DESC LIMIT 1", workspaceUuid, userUuid)
			exp := time.Time{}
			latestErr := latest.Scan(&status.AfterID, &exp)
			if latestErr != nil && !errors.Is(latestErr, sql.ErrNoRows) {
				return false, latestErr
			}

			if exp.IsZero() {
				status.ExpiryTime = time.Now().Add(time.Second * time.Duration(duration))
			} else {
				status.ExpiryTime = exp.Add(time.Second * time.Duration(duration))
			}
			expiry = &status.ExpiryTime
		}

		afterId = &status.AfterID
	}

	if status.ExpiryTime.IsZero() || status.AfterID == rubix.OverlayAfterID {
		expiry = nil
	}

	res, err := p.primaryConnection.Exec("INSERT INTO user_status (workspace, user, state, extendedState, expiry, applied, id, afterId, duration, clearOnLogout) "+
		"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) "+
		"ON DUPLICATE KEY UPDATE "+
		"state = ?, extendedState = ?, expiry = ?, applied = ?, afterId = ?, duration = ?, clearOnLogout = ?",
		workspaceUuid, userUuid, status.State, status.ExtendedState, expiry, time.Now(), status.ID, afterId, duration, status.ClearOnLogout,
		status.State, status.ExtendedState, expiry, time.Now(), afterId, duration, status.ClearOnLogout)
	if err != nil {
		return false, err
	}
	impact, err := res.RowsAffected()

	// Always clear previous states when applying
	go p.primaryConnection.Exec("DELETE FROM user_status  WHERE workspace = ? AND user = ? AND expiry < ? AND expiry IS NOT NULL", workspaceUuid, userUuid, time.Now())

	return impact > 0, err
}

func (p *Provider) ClearUserStatusLogout(workspaceUuid, userUuid string) error {
	_, err := p.primaryConnection.Exec("DELETE FROM user_status  WHERE workspace = ? AND user = ? AND clearOnLogout = 1", workspaceUuid, userUuid)
	return err
}

func (p *Provider) setExpiry(workspaceUuid, userUuid, statusID string, expiry time.Time) error {
	_, err := p.primaryConnection.Exec("UPDATE user_status SET expiry = ? WHERE workspace = ? AND user = ? AND id = ?", expiry, workspaceUuid, userUuid, statusID)
	return err
}

func (p *Provider) ClearUserStatusID(workspaceUuid, userUuid, statusID string) error {

	if statusID == "" {
		return errors.New("statusID is required")
	}

	_, deleteErr := p.primaryConnection.Exec("DELETE FROM user_status  WHERE workspace = ? AND user = ? AND (id = ? OR (expiry < ? AND expiry IS NOT NULL))", workspaceUuid, userUuid, statusID, time.Now())
	return deleteErr
}

func (p *Provider) GetUserStatus(workspaceUuid, userUuid string) (rubix.UserStatus, error) {
	status := rubix.UserStatus{}
	var expiry *time.Time
	rows, err := p.primaryConnection.Query("SELECT state, extendedState, applied, expiry, id, afterId, duration, clearOnLogout FROM user_status WHERE workspace = ? AND user = ? AND (expiry IS NULL OR expiry > ?)", workspaceUuid, userUuid, time.Now())
	if err != nil {
		return status, err
	}
	defer rows.Close()

	for rows.Next() {
		newResult := rubix.UserStatus{}
		afterId := sql.NullString{}
		if scanErr := rows.Scan(&newResult.State, &newResult.ExtendedState, &newResult.AppliedTime, &expiry, &newResult.ID, &afterId, &newResult.ClearAfterSeconds, &newResult.ClearOnLogout); scanErr != nil {
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
			status.AfterID = newResult.AfterID
			status.ClearAfterSeconds = newResult.ClearAfterSeconds
			status.ClearOnLogout = newResult.ClearOnLogout
		} else {
			status.Overlays = append(status.Overlays, newResult)
		}
	}

	writeExpiry := status.AfterID == rubix.OverlayAfterID && len(status.Overlays) == 0 && status.ExpiryTime.IsZero()

	status.Repair()

	if writeExpiry {
		status.ExpiryTime = time.Now().Add(time.Second * time.Duration(status.ClearAfterSeconds))
		_ = p.setExpiry(workspaceUuid, userUuid, status.ID, status.ExpiryTime)
	}

	return status, nil
}
