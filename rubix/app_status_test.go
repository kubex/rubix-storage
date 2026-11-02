package rubix

import "testing"

func TestParseAppStatus(t *testing.T) {
	cases := []struct {
		in      string
		want    AppStatus
		ok      bool
		blocked bool
	}{
		{"", AppStatusActive, true, false},
		{"active", AppStatusActive, true, false},
		{" Pre-Release ", AppStatusPreRelease, true, false},
		{"suspended", AppStatusSuspended, true, true},
		{"archived", AppStatusArchived, true, true},
		{"deleted", AppStatusActive, false, false},
	}
	for _, c := range cases {
		got, ok := ParseAppStatus(c.in)
		if got != c.want || ok != c.ok || got.Blocked() != c.blocked {
			t.Fatalf("ParseAppStatus(%q) = %q, %v (blocked %v); want %q, %v (blocked %v)", c.in, got, ok, got.Blocked(), c.want, c.ok, c.blocked)
		}
	}
	if AppStatus("").Blocked() {
		t.Fatal("empty status must not be blocked")
	}
}
