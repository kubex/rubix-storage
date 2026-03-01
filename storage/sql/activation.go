package sql

import (
	"github.com/kubex/rubix-storage/rubix"
)

func (p *Provider) CompleteActivationStep(workspace, user, vendor, app, stepID string) error {
	query := "INSERT INTO app_activation_state (workspace, user, vendor, app, step_id) VALUES (?, ?, ?, ?, ?)"
	if p.SqlLite {
		query += " ON CONFLICT(workspace, user, vendor, app, step_id) DO NOTHING"
	} else {
		query += " ON DUPLICATE KEY UPDATE completed_at = completed_at"
	}
	_, err := p.primaryConnection.Exec(query, workspace, user, vendor, app, stepID)
	if err == nil {
		p.update()
	}
	return err
}

func (p *Provider) ResetActivationSteps(workspace, vendor, app string) error {
	_, err := p.primaryConnection.Exec(
		"DELETE FROM app_activation_state WHERE workspace = ? AND vendor = ? AND app = ?",
		workspace, vendor, app,
	)
	if err == nil {
		p.update()
	}
	return err
}

func (p *Provider) GetActivationState(workspace, user, vendor, app string) ([]rubix.ActivationState, error) {
	rows, err := p.primaryConnection.Query(
		"SELECT workspace, user, vendor, app, step_id, completed_at FROM app_activation_state WHERE workspace = ? AND vendor = ? AND app = ? AND (user = '' OR user = ?)",
		workspace, vendor, app, user,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []rubix.ActivationState
	for rows.Next() {
		var s rubix.ActivationState
		if err := rows.Scan(&s.Workspace, &s.UserID, &s.VendorID, &s.AppID, &s.StepID, &s.CompletedAt); err != nil {
			return nil, err
		}
		states = append(states, s)
	}
	return states, rows.Err()
}
