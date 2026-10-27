package sql

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kubex/rubix-storage/rubix"
)

func (p *Provider) DeleteSetting(workspace, vendor, app, key string) error {
	_, err := p.primaryConnection.Exec("DELETE FROM settings WHERE workspace = ? AND vendor = ? AND app = ? AND `key` = ?", workspace, vendor, app, key)
	return err
}

// StoreTaskRun serializes writers per task, including writers on different pods.
func (p *Provider) StoreTaskRun(run rubix.TaskRun, maxRuns int, notBefore time.Time) error {
	if run.TaskKey == "" || run.RunID == "" || run.FinishedAt.IsZero() || maxRuns < 1 {
		return errors.New("invalid task run")
	}
	tx, err := p.primaryConnection.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	lock := "INSERT INTO scheduled_task_run_groups (task_key) VALUES (?)"
	if p.SqlLite {
		lock += " ON CONFLICT(task_key) DO UPDATE SET task_key = excluded.task_key"
	} else {
		lock += " ON DUPLICATE KEY UPDATE task_key = VALUES(task_key)"
	}
	// The upsert acquires an exclusive row lock until commit (SQLite serializes writes).
	if _, err := tx.Exec(lock, run.TaskKey); err != nil {
		return fmt.Errorf("lock task history: %w", err)
	}
	insert := "INSERT INTO scheduled_task_runs (task_key,run_id,finished_at,`value`) VALUES (?,?,?,?)"
	if p.SqlLite {
		insert += " ON CONFLICT(task_key,run_id) DO UPDATE SET finished_at = excluded.finished_at, `value` = excluded.`value`"
	} else {
		insert += " ON DUPLICATE KEY UPDATE finished_at = VALUES(finished_at), `value` = VALUES(`value`)"
	}
	if _, err := tx.Exec(insert, run.TaskKey, run.RunID, run.FinishedAt.UnixNano(), run.Value); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM scheduled_task_runs WHERE task_key = ? AND finished_at < ?", run.TaskKey, notBefore.UnixNano()); err != nil {
		return err
	}
	rows, err := tx.Query("SELECT run_id FROM scheduled_task_runs WHERE task_key = ? ORDER BY finished_at DESC, run_id DESC LIMIT ?", run.TaskKey, maxRuns)
	if err != nil {
		return err
	}
	var keep []any
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		keep = append(keep, id)
	}
	scanErr := rows.Err()
	rows.Close()
	if scanErr != nil {
		return scanErr
	}
	if len(keep) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(keep)), ",")
		args := append([]any{run.TaskKey}, keep...)
		if _, err := tx.Exec("DELETE FROM scheduled_task_runs WHERE task_key = ? AND run_id NOT IN ("+placeholders+")", args...); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (p *Provider) GetTaskRuns(taskKeys []string, notBefore time.Time) ([]rubix.TaskRun, error) {
	if len(taskKeys) == 0 {
		return nil, nil
	}
	args := []any{notBefore.UnixNano()}
	for _, key := range taskKeys {
		args = append(args, key)
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(taskKeys)), ",")
	rows, err := p.primaryConnection.Query("SELECT task_key,run_id,finished_at,`value` FROM scheduled_task_runs WHERE finished_at >= ? AND task_key IN ("+placeholders+") ORDER BY finished_at DESC,run_id DESC", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var runs []rubix.TaskRun
	for rows.Next() {
		var run rubix.TaskRun
		var finished int64
		if err := rows.Scan(&run.TaskKey, &run.RunID, &finished, &run.Value); err != nil {
			return nil, err
		}
		run.FinishedAt = time.Unix(0, finished).UTC()
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func (p *Provider) PruneTaskRuns(notBefore time.Time) error {
	_, err := p.primaryConnection.Exec("DELETE FROM scheduled_task_runs WHERE finished_at < ?", notBefore.UnixNano())
	return err
}
