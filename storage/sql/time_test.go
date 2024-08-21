package sql

import (
	"testing"
	"time"
)

func TestTimeFromString(t *testing.T) {
	testCases := []struct {
		input    string
		expected time.Time
	}{
		{"2024-07-25T14:37:52.42853218Z", time.Date(2024, 7, 25, 14, 37, 52, 428532180, time.UTC)},
		{"2006-01-02 15:04:05.999999999-07:00", time.Date(2006, 1, 2, 22, 4, 5, 999999999, time.UTC)},
		{"2006-01-02T15:04:05.999999999-07:00", time.Date(2006, 1, 2, 22, 4, 5, 999999999, time.UTC)},
		{"2006-01-02 15:04:05.999999999", time.Date(2006, 1, 2, 15, 4, 5, 999999999, time.UTC)},
		{"2006-01-02T15:04:05.999999999", time.Date(2006, 1, 2, 15, 4, 5, 999999999, time.UTC)},
		{"2006-01-02 15:04:05", time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)},
		{"2006-01-02T15:04:05", time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)},
		{"2006-01-02 15:04", time.Date(2006, 1, 2, 15, 4, 0, 0, time.UTC)},
		{"2006-01-02T15:04", time.Date(2006, 1, 2, 15, 4, 0, 0, time.UTC)},
		{"2006-01-02", time.Date(2006, 1, 2, 0, 0, 0, 0, time.UTC)},
		{"invalid", time.Time{}},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := timeFromString(tc.input)
			if !result.Equal(tc.expected) {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}
