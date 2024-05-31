package rubix

import (
	"errors"
)

var (
	ErrNoResultFound = errors.New("no result found")
	ErrDuplicate     = errors.New("already exists")
)
