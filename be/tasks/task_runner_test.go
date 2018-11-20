package tasks

import (
	"context"
	"io/ioutil"
	"log"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"

	"pixur.org/pixur/be/status"
)

var silentLogger = log.New(ioutil.Discard, "", 0)

type fakeTask struct {
	runCount int
	run      func(context.Context) status.S
}

func (t *fakeTask) Run(ctx context.Context) status.S {
	t.runCount++
	if t.run != nil {
		return t.run(ctx)
	}
	return nil
}

func TestTaskRunnerNil(t *testing.T) {
	var tr *TaskRunner
	sts := tr.Run(context.Background(), new(fakeTask))
	if sts != nil {
		t.Error("error running task", sts)
	}
}

type fakeError bool

func (e *fakeError) Error() string {
	return "bad"
}

func (e *fakeError) CanRetry() bool {
	return bool(*e)
}

func TestTaskRetriesOnRetryableError(t *testing.T) {
	runner := new(TaskRunner)
	runner.logger = silentLogger
	task := new(fakeTask)

	task.run = func(_ context.Context) status.S {
		err := fakeError(true)
		return status.Internal(&err, "err")
	}

	oldtaskInitialBackoffDelay := taskInitialBackoffDelay
	taskInitialBackoffDelay = 1
	sts := runner.Run(context.Background(), task)
	taskInitialBackoffDelay = oldtaskInitialBackoffDelay

	if sts == nil {
		t.Fatal("Expected error, but was nil")
	}

	if task.runCount != taskMaxRetries {
		t.Fatalf("Expected task to be run %d times !%d", taskMaxRetries, task.runCount)
	}
}

func TestTaskFailsOnOtherError(t *testing.T) {
	runner := new(TaskRunner)
	task := new(fakeTask)

	task.run = func(_ context.Context) status.S {
		err := fakeError(false)
		return status.Internal(&err, "")
	}
	sts := runner.Run(context.Background(), task)

	if sts == nil {
		t.Fatal("Expected error, but was nil")
	}

	if task.runCount != 1 {
		t.Fatal("Expected task to be run 1 time")
	}
}

func TestTaskRunnerRunTaskOnce_failsOnAlreadyComplete(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	task := new(fakeTask)

	success, retry, failures := runTaskOnce(ctx, task, time.Second, time.Now)

	if success || retry {
		t.Error("expected failures", success, retry)
	}
	if len(failures) != 1 {
		t.Error("bad number of failures", failures)
	}
	if have, want := failures[0].Code(), codes.Canceled; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := failures[0].Message(), "canceled"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
	if task.runCount != 0 {
		t.Error("ran task", task.runCount)
	}
}

func TestTaskRunnerRunTaskOnce_succeeds(t *testing.T) {
	ctx := context.Background()

	task := new(fakeTask)

	success, retry, failures := runTaskOnce(ctx, task, time.Second, time.Now)

	if !success {
		t.Error("expected success")
	}
	if retry {
		t.Error("expected no retry")
	}
	if len(failures) != 0 {
		t.Error(failures)
	}
	if task.runCount != 1 {
		t.Error("did not run task", task.runCount)
	}
}

func TestTaskRunnerRunTaskOnce_failsRetryable(t *testing.T) {
	ctx := context.Background()
	task := new(fakeTask)
	task.run = func(_ context.Context) status.S {
		err := fakeError(true)
		return status.Internal(&err, "err")
	}

	success, retry, failures := runTaskOnce(ctx, task, 0, time.Now)

	if success {
		t.Error("expected failures")
	}
	if !retry {
		t.Error("expected retry")
	}
	if len(failures) != 1 {
		t.Error("bad number of failures", failures)
	}
	if have, want := failures[0].Code(), codes.Internal; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := failures[0].Message(), "err"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
	if task.runCount != 1 {
		t.Error("ran task", task.runCount)
	}
}

func TestTaskRunnerRunTaskOnce_failsNonRetryable(t *testing.T) {
	ctx := context.Background()
	task := new(fakeTask)
	task.run = func(_ context.Context) status.S {
		err := fakeError(false)
		return status.Internal(&err, "err")
	}

	// use a long timeout incase this keeps running.  If this goes too long, there
	// is a bug.
	success, retry, failures := runTaskOnce(ctx, task, 5*time.Minute, time.Now)

	if success {
		t.Error("expected failures")
	}
	if retry {
		t.Error("expected no retry")
	}
	if len(failures) != 1 {
		t.Error("bad number of failures", failures)
	}
	if have, want := failures[0].Code(), codes.Internal; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := failures[0].Message(), "err"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
	if task.runCount != 1 {
		t.Error("ran task", task.runCount)
	}
}

func TestTaskRunnerRunTaskOnce_failsRetryable_longDeadline(t *testing.T) {
	nowt := time.Now()
	now := func() time.Time {
		return nowt
	}
	ctx, cancel := context.WithDeadline(context.Background(), now().Add(time.Minute))
	defer cancel()
	task := new(fakeTask)
	task.run = func(_ context.Context) status.S {
		err := fakeError(true)
		return status.Internal(&err, "err")
	}

	success, retry, failures := runTaskOnce(ctx, task, 0, now)

	if success {
		t.Error("expected failures")
	}
	if !retry {
		t.Error("expected retry")
	}
	if len(failures) != 1 {
		t.Error("bad number of failures", failures)
	}
	if have, want := failures[0].Code(), codes.Internal; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := failures[0].Message(), "err"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
	if task.runCount != 1 {
		t.Error("ran task", task.runCount)
	}
}

func TestTaskRunnerRunTaskOnce_failsRetryable_shortDeadline(t *testing.T) {
	nowt := time.Now()
	now := func() time.Time {
		return nowt
	}
	ctx, cancel := context.WithDeadline(context.Background(), now().Add(time.Second))
	defer cancel()
	task := new(fakeTask)
	task.run = func(_ context.Context) status.S {
		err := fakeError(true)
		return status.Internal(&err, "err")
	}

	success, retry, failures := runTaskOnce(ctx, task, time.Minute, now)

	if success {
		t.Error("expected failures")
	}
	if retry {
		t.Error("expected no retry")
	}
	if len(failures) != 2 {
		t.Error("bad number of failures", failures)
	}
	if have, want := failures[0].Code(), codes.Internal; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := failures[0].Message(), "err"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
	if have, want := failures[1].Code(), codes.DeadlineExceeded; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := failures[1].Message(), "not enough time"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
	if task.runCount != 1 {
		t.Error("ran task", task.runCount)
	}
}

func TestTaskRunnerRunTaskOnce_failsRetryable_earlyCancel(t *testing.T) {
	nowt := time.Now()
	now := func() time.Time {
		return nowt
	}
	ctx, cancel := context.WithDeadline(context.Background(), now().Add(time.Minute))
	defer cancel()
	task := new(fakeTask)
	task.run = func(_ context.Context) status.S {
		err := fakeError(true)
		cancel()
		return status.Internal(&err, "err")
	}

	success, retry, failures := runTaskOnce(ctx, task, time.Second, now)

	if success {
		t.Error("expected failures")
	}
	if retry {
		t.Error("expected no retry")
	}
	if len(failures) != 2 {
		t.Error("bad number of failures", failures)
	}
	if have, want := failures[0].Code(), codes.Internal; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := failures[0].Message(), "err"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
	if have, want := failures[1].Code(), codes.Canceled; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := failures[1].Message(), "cancel"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
	if task.runCount != 1 {
		t.Error("ran task", task.runCount)
	}
}
