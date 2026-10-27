package rubix

import "time"

// TaskRun holds a run payload under an opaque workspace/app/task key.
type TaskRun struct {
	TaskKey    string
	RunID      string
	FinishedAt time.Time
	Value      string
}
