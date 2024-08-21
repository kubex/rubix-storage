package sql

import (
	"time"
)

func timeFromString(in string) time.Time {
	for _, format := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		time.DateTime,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02T15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02T15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04",
		"2006-01-02",
	} {

		if t, err := time.ParseInLocation(format, in, time.UTC); err == nil {
			return t
		}
	}
	return time.Time{}
}
