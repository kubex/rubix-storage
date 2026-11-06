package sql

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"syscall"
	"testing"
)

func TestIsStaleConnErr(t *testing.T) {
	libsqlWrap := func(err error) error {
		return fmt.Errorf("failed to execute SQL:\n%w", &url.Error{Op: "Post", URL: "https://db.turso.io/v2/pipeline", Err: err})
	}
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"eof via libsql", libsqlWrap(io.EOF), true},
		{"unexpected eof via libsql", libsqlWrap(io.ErrUnexpectedEOF), true},
		{"connection reset via libsql", libsqlWrap(syscall.ECONNRESET), true},
		{"server closed idle via libsql", libsqlWrap(errors.New("http: server closed idle connection")), true},
		{"connection refused via libsql", libsqlWrap(syscall.ECONNREFUSED), false},
		{"sql error", errors.New("no such table: workspaces"), false},
	}
	for _, c := range cases {
		if got := isStaleConnErr(c.err); got != c.want {
			t.Errorf("%s: isStaleConnErr = %v, want %v", c.name, got, c.want)
		}
	}
}
