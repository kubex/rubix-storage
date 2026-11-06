package sql

import (
	"database/sql"
	"errors"
	"io"
	"strings"
	"syscall"
)

// errServerClosedIdleMsg is the text of net/http's unexported errServerClosedIdle,
// returned when the server's close arrives while the request is being written.
const errServerClosedIdleMsg = "http: server closed idle connection"

// isStaleConnErr reports whether err is the failure seen when a pooled
// keep-alive connection was closed by the server between requests. net/http
// only replays idempotent requests, and libsql sends each statement as a POST,
// so the first request on a stale connection surfaces one of these errors.
func isStaleConnErr(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, syscall.ECONNRESET) ||
		strings.Contains(err.Error(), errServerClosedIdleMsg)
}

// query runs a read-only statement, retrying once on a stale connection.
// Only pass SELECT statements: the retry re-executes the statement.
func (p *Provider) query(query string, args ...any) (*sql.Rows, error) {
	rows, err := p.primaryConnection.Query(query, args...)
	if err != nil && isStaleConnErr(err) {
		rows, err = p.primaryConnection.Query(query, args...)
	}
	return rows, err
}

// queryRow runs a read-only single-row statement, retrying once on a stale
// connection. Only pass SELECT statements: the retry re-executes the statement.
func (p *Provider) queryRow(query string, args ...any) *sql.Row {
	row := p.primaryConnection.QueryRow(query, args...)
	if isStaleConnErr(row.Err()) {
		row = p.primaryConnection.QueryRow(query, args...)
	}
	return row
}
