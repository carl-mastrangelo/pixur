package tasks

import (
	"context"
	"io/ioutil"
	"log"
	"math/rand"
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

func TestTaskRunnerRetryAdapter_emptyNotRetryable(t *testing.T) {
	retry := &taskRetryAdapter{}

	if _, ok := retry.Next(); ok {
		t.Error("should not be retryable")
	}
}

func TestTaskRunnerRetryAdapter_maxTries(t *testing.T) {
	retry := &taskRetryAdapter{
		maxTries: 5,
	}

	tries := 0
	for i := 0; i < 10; i++ {
		if _, ok := retry.Next(); ok {
			tries++
		}
	}

	if tries != 5 {
		t.Error("wrong tries", tries, 5)
	}
}

func TestTaskRunnerRetryAdapter_noJitterForFirst(t *testing.T) {
	retry := &taskRetryAdapter{
		maxTries:     5,
		initialDelay: time.Second,
		rng: func() *rand.Rand {
			t.Fatal("should not be called!")
			return nil
		},
	}

	retry.Next()
}

func TestTaskRunnerRetryAdapter_JitterForSecond(t *testing.T) {
	rngCall := 0
	retry := &taskRetryAdapter{
		maxTries:     5,
		initialDelay: time.Second,
		rng: func() *rand.Rand {
			rngCall++
			return nil
		},
	}

	retry.Next()
	retry.Next()

	if rngCall != 1 {
		t.Error("rng not used")
	}
}

func TestTaskRunnerRetryAdapter_nilRngResult(t *testing.T) {
	rngCall := 0
	retry := &taskRetryAdapter{
		maxTries:     5,
		multiplier:   1,
		initialDelay: time.Second,
		rng: func() *rand.Rand {
			rngCall++
			return nil
		},
	}

	retry.Next()
	retry.Next()
	retry.Next()

	if rngCall != 2 {
		t.Error("rng used wrong", rngCall)
	}
}

func TestTaskRunnerRetryAdapter_nilRngResultUnusedIfRnPresent(t *testing.T) {
	rn := rand.New(rand.NewSource(0))

	retry := &taskRetryAdapter{
		maxTries:     5,
		initialDelay: time.Second,
		rng: func() *rand.Rand {
			t.Fatal("should not be called!")
			return nil
		},
		rn: rn,
	}

	retry.Next()
	retry.Next()
	retry.Next()
}

func TestTaskRunnerRetryAdapter_multiply(t *testing.T) {
	retry := &taskRetryAdapter{
		maxTries:     5,
		multiplier:   2,
		initialDelay: time.Second,
		rng: func() *rand.Rand {
			return nil
		},
	}

	delay1, ok1 := retry.Next()
	delay2, ok2 := retry.Next()
	delay3, ok3 := retry.Next()
	if !ok1 || !ok2 || !ok3 {
		t.Fatal(ok1, ok2, ok3)
	}
	if have, want := delay1, time.Second; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := delay2, 2*time.Second; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := delay3, 4*time.Second; have != want {
		t.Error("have", have, "want", want)
	}
}

func TestTaskRunnerRetryAdapter_jitter(t *testing.T) {
	rn := rand.New(rand.NewSource(0))
	rn1 := rn.Float64()
	rn2 := rn.Float64()
	rn = rand.New(rand.NewSource(0))
	retry := &taskRetryAdapter{
		maxTries:     5,
		multiplier:   3,
		jitter:       0.5,
		initialDelay: time.Second,
		rn:           rn,
	}

	delay1, ok1 := retry.Next()
	delay2, ok2 := retry.Next()
	delay3, ok3 := retry.Next()
	if !ok1 || !ok2 || !ok3 {
		t.Fatal(ok1, ok2, ok3)
	}
	if have, want := delay1, time.Second; have != want {
		t.Error("have", have, "want", want)
	}

	if have, want := delay2, time.Duration((3*(1+(rn1*2-1)*.5))*float64(delay1)); have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := delay3, time.Duration((3*(1+(rn2*2-1)*.5))*float64(delay2)); have != want {
		t.Error("have", have, "want", want)
	}
}
