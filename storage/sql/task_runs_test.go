package sql

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/kubex/rubix-storage/rubix"
)

func TestTaskRunHistoryBoundsAgeAndConcurrentWriters(t *testing.T) {
	first := newTestProvider(t)
	defer first.Close()
	second := &Provider{SqlLite: true, PrimaryDSN: first.PrimaryDSN}
	if err := second.Connect(); err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	now := time.Now().UTC()
	cutoff := now.Add(-30 * 24 * time.Hour)
	var wg sync.WaitGroup
	failures := make(chan error, 30)
	for i := range 25 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			provider := first
			if i%2 == 1 {
				provider = second
			}
			failures <- provider.StoreTaskRun(rubix.TaskRun{TaskKey: "task-one", RunID: fmt.Sprintf("run-%02d", i), FinishedAt: now.Add(time.Duration(i) * time.Second), Value: strconv.Itoa(i)}, 10, cutoff)
		}(i)
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	runs, err := first.GetTaskRuns([]string{"task-one"}, cutoff)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 10 {
		t.Fatalf("stored %d runs, want 10", len(runs))
	}
	for i, run := range runs {
		want := fmt.Sprintf("run-%02d", 24-i)
		if run.RunID != want {
			t.Fatalf("run %d=%s want %s", i, run.RunID, want)
		}
	}
	// Repeated completion updates the same occurrence rather than duplicating it.
	if err := first.StoreTaskRun(runs[0], 10, cutoff); err != nil {
		t.Fatal(err)
	}
	again, err := first.GetTaskRuns([]string{"task-one"}, cutoff)
	if err != nil || len(again) != 10 {
		t.Fatalf("duplicate run: %d, %v", len(again), err)
	}
	if err := first.StoreTaskRun(rubix.TaskRun{TaskKey: "expired", RunID: "old", FinishedAt: cutoff.Add(-time.Second), Value: "old"}, 10, time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	if err := first.StoreTaskRun(rubix.TaskRun{TaskKey: "boundary", RunID: "boundary", FinishedAt: cutoff, Value: "boundary"}, 10, cutoff); err != nil {
		t.Fatal(err)
	}
	if err := first.PruneTaskRuns(cutoff); err != nil {
		t.Fatal(err)
	}
	expired, err := first.GetTaskRuns([]string{"expired"}, time.Unix(0, 0))
	if err != nil || len(expired) != 0 {
		t.Fatalf("expired records retained: %+v %v", expired, err)
	}
	boundary, err := first.GetTaskRuns([]string{"boundary"}, cutoff)
	if err != nil || len(boundary) != 1 {
		t.Fatalf("boundary removed: %+v %v", boundary, err)
	}
	if err := first.PruneTaskRuns(cutoff.Add(time.Nanosecond)); err != nil {
		t.Fatal(err)
	}
	boundary, err = first.GetTaskRuns([]string{"boundary"}, time.Unix(0, 0))
	if err != nil || len(boundary) != 0 {
		t.Fatal("idle task history did not expire")
	}
}
