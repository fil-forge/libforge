package taskrunner_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fil-forge/libforge/taskrunner"
)

// WaitAll returns once every task has finished, however each ended, with the
// result of one that succeeded and the error Wait gives for the others.
func TestWaitAllReportsHowEachTaskEnded(t *testing.T) {
	f := newFixture(t, 2)
	release := make(chan struct{})
	f.handle(t, qTop, "step", func(_ context.Context, _ *taskrunner.Exec, task taskrunner.Task) ([]byte, error) {
		switch string(task.Args) {
		case "refused":
			return nil, errors.New("refused")
		case "slow":
			<-release
		}
		return append([]byte("done:"), task.Args...), nil
	})
	f.start(t)
	ctx := waitCtx(t)
	submit := func(subject, args string, at *time.Time) taskrunner.TaskID {
		t.Helper()
		id, err := f.r.Submit(ctx, taskrunner.Spec{Queue: qTop, Subject: subject, Action: "step", Args: []byte(args), NotBefore: at})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	later := time.Now().Add(time.Hour)
	ok := submit("a", "ok", nil)
	refused := submit("b", "refused", nil)
	slow := submit("c", "slow", nil)
	cancelled := submit("d", "never", &later)
	if err := f.r.Cancel(ctx, qTop, cancelled); err != nil {
		t.Fatal(err)
	}
	unknown := uuid.New()

	// The slow task is still running, so the wait must not return yet.
	short, stop := context.WithTimeout(ctx, 200*time.Millisecond)
	got, err := f.r.WaitAll(short, qTop, ok, slow)
	stop()
	if !errors.Is(err, context.DeadlineExceeded) || got != nil {
		t.Fatalf("WaitAll with a task still running = %v, %v, want nothing and the deadline", got, err)
	}

	close(release)
	got, err = f.r.WaitAll(ctx, qTop, ok, refused, slow, cancelled, unknown, ok)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Fatalf("WaitAll returned %d completions, want one for each of the 5 tasks", len(got))
	}
	for id, want := range map[taskrunner.TaskID]string{ok: "done:ok", slow: "done:slow"} {
		if o := got[id]; o.Err != nil || string(o.Result) != want {
			t.Errorf("task %s: %q, %v, want %q", id, o.Result, o.Err, want)
		}
	}
	var failed *taskrunner.HandlerError
	if o := got[refused]; !errors.As(o.Err, &failed) || failed.Message != "refused" || o.Result != nil {
		t.Errorf("the refused task: %q, %v, want a HandlerError", o.Result, o.Err)
	}
	if o := got[cancelled]; !errors.Is(o.Err, taskrunner.ErrCancelled) {
		t.Errorf("the cancelled task: %v, want ErrCancelled", o.Err)
	}
	if o := got[unknown]; !errors.Is(o.Err, taskrunner.ErrNotFound) {
		t.Errorf("the unknown task: %v, want ErrNotFound", o.Err)
	}
}

// A task still running when the wait begins is waited for, among many that
// have already ended, and every one of them is reported.
func TestWaitAllWaitsForTheTasksStillPending(t *testing.T) {
	f := newFixture(t, 4)
	started, release := make(chan struct{}), make(chan struct{})
	f.handle(t, qTop, "step", func(_ context.Context, _ *taskrunner.Exec, task taskrunner.Task) ([]byte, error) {
		if string(task.Args) == "slow" {
			close(started)
			<-release
		}
		return task.Args, nil
	})
	f.start(t)
	ctx := waitCtx(t)
	var ids []taskrunner.TaskID
	for i := range 40 {
		id, err := f.r.Submit(ctx, taskrunner.Spec{Queue: qTop, Subject: "s" + strconv.Itoa(i), Action: "step", Args: []byte("quick")})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if _, err := f.r.WaitAll(ctx, qTop, ids...); err != nil {
		t.Fatal(err)
	}
	slow, err := f.r.Submit(ctx, taskrunner.Spec{Queue: qTop, Subject: "slow", Action: "step", Args: []byte("slow")})
	if err != nil {
		t.Fatal(err)
	}
	<-started
	go func() {
		time.Sleep(100 * time.Millisecond)
		close(release)
	}()

	got, err := f.r.WaitAll(ctx, qTop, append([]taskrunner.TaskID{slow}, ids...)...)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 41 {
		t.Fatalf("WaitAll returned %d completions, want 41", len(got))
	}
	if c := got[slow]; c.Err != nil || string(c.Result) != "slow" {
		t.Errorf("the slow task: %q, %v", c.Result, c.Err)
	}
	for _, id := range ids {
		if c := got[id]; c.Err != nil || string(c.Result) != "quick" {
			t.Fatalf("task %s: %q, %v", id, c.Result, c.Err)
		}
	}
}

// A task that ends while WaitAll is held up by a slower one is recorded when
// it ends, so its row being deleted before the slow one finishes loses nothing.
func TestWaitAllRecordsATaskThatEndsWhileAnotherIsSlow(t *testing.T) {
	f := newFixture(t, 2)
	started, release := make(chan struct{}), make(chan struct{})
	f.handle(t, qTop, "step", func(_ context.Context, _ *taskrunner.Exec, task taskrunner.Task) ([]byte, error) {
		if string(task.Args) == "slow" {
			close(started)
			<-release
		}
		return task.Args, nil
	})
	f.start(t)
	ctx := waitCtx(t)
	submit := func(subject, args string) taskrunner.TaskID {
		t.Helper()
		id, err := f.r.Submit(ctx, taskrunner.Spec{Queue: qTop, Subject: subject, Action: "step", Args: []byte(args)})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	slow := submit("slow", "slow")
	<-started

	type waited struct {
		got map[taskrunner.TaskID]taskrunner.Completion
		err error
	}
	done := make(chan waited, 1)
	quick := submit("quick", "quick")
	go func() {
		// The slow task first: a wait that took the IDs in turn would not
		// look at the quick one until the slow one ended.
		got, err := f.r.WaitAll(ctx, qTop, slow, quick)
		done <- waited{got, err}
	}()

	// Once the quick task has ended and the wait has had time to see it,
	// its row goes, as retention would take it.
	if _, err := f.r.Wait(ctx, qTop, quick); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	tasks := f.env.Table(queueTable(qTop) + "_tasks")
	if _, err := f.env.DB.Exec(ctx, `DELETE FROM `+tasks+` WHERE id = $1`, quick); err != nil {
		t.Fatal(err)
	}
	close(release)

	w := <-done
	if w.err != nil {
		t.Fatal(w.err)
	}
	if c := w.got[quick]; c.Err != nil || string(c.Result) != "quick" {
		t.Fatalf("the quick task: %q, %v, want its result", c.Result, c.Err)
	}
	if c := w.got[slow]; c.Err != nil || string(c.Result) != "slow" {
		t.Fatalf("the slow task: %q, %v, want its result", c.Result, c.Err)
	}
}

// With nothing to wait for WaitAll returns at once, and an unknown queue is
// its own error.
func TestWaitAllWithNothingToWaitFor(t *testing.T) {
	f := newFixture(t, 0)
	f.handle(t, qTop, "step", func(context.Context, *taskrunner.Exec, taskrunner.Task) ([]byte, error) { return nil, nil })
	f.migrate(t)
	ctx := waitCtx(t)

	got, err := f.r.WaitAll(ctx, qTop)
	if err != nil || len(got) != 0 {
		t.Fatalf("WaitAll of no tasks = %v, %v, want none", got, err)
	}
	if _, err := f.r.WaitAll(ctx, "no-such-queue", uuid.New()); err == nil {
		t.Fatal("WaitAll accepted an unknown queue")
	}
}
