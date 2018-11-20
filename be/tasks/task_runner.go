package tasks

import (
	"context"
	"expvar"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"runtime/trace"
	"time"

	"pixur.org/pixur/be/schema/db"
	"pixur.org/pixur/be/status"
)

var defaulttasklogger *log.Logger = log.New(os.Stderr, "", log.LstdFlags)

var (
	totalTasksCounter   = expvar.NewInt("PixurTasks")
	successTasksCounter = expvar.NewInt("PixurSuccessTasks")
	failureTasksCounter = expvar.NewInt("PixurFailureTasks")
)

var (
	taskMaxRetries          = 5
	taskInitialBackoffDelay = time.Second
	taskBackoffMultiplier   = math.Phi
	taskBackoffJitter       = 0.2
)

type TaskRunner struct {
	run    func(context.Context, Task) status.S
	logger *log.Logger
	now    func() time.Time
	rng    func() *rand.Rand
}

func TestTaskRunner(run func(context.Context, Task) status.S) *TaskRunner {
	return &TaskRunner{
		run: run,
	}
}

func (r *TaskRunner) Run(ctx context.Context, task Task) status.S {
	if trace.IsEnabled() {
		var tracetask *trace.Task
		ctx, tracetask = trace.NewTask(ctx, fmt.Sprintf("%T", task))
		defer tracetask.End()
	}
	totalTasksCounter.Add(1)
	var logger *log.Logger
	if r != nil && r.logger != nil {
		logger = r.logger
	} else {
		logger = defaulttasklogger
	}

	var now func() time.Time
	if r != nil && r.now != nil {
		now = r.now
	} else {
		now = time.Now
	}
	var rng func() *rand.Rand
	if r != nil && r.rng != nil {
		rng = r.rng
	} else {
		rng = func() *rand.Rand {
			return rand.New(rand.NewSource(now().UnixNano()))
		}
	}

	var sts status.S
	if r != nil && r.run != nil {
		sts = r.run(ctx, task)
	}
	sts = runTask(ctx, task, now, rng, logger)
	if sts != nil {
		failureTasksCounter.Add(1)
	} else {
		successTasksCounter.Add(1)
	}
	return sts
}

func stsSliceToErrSlice(ss []status.S) []error {
	es := make([]error, 0, len(ss))
	for _, s := range ss {
		es = append(es, s)
	}
	return es
}

// TODO: test
func runTask(
	ctx context.Context, task Task, now func() time.Time, rng func() *rand.Rand, logger *log.Logger) (
	stscap status.S) {
	var stss []status.S
	defer func() {
		if stscap == nil && len(stss) != 0 && logger != nil {
			warnsts := status.Unknownf(nil, "Encountered %d errors while running %T", len(stss), task)
			warnsts = status.WithSuppressed(warnsts, stsSliceToErrSlice(stss)...)
			logger.Println(warnsts)
		}
	}()

	backoff := taskInitialBackoffDelay
	var rn *rand.Rand
	for i := 0; i < taskMaxRetries; i++ {
		success, retry, failures := runTaskOnce(ctx, task, backoff, now)
		if success {
			return nil
		}
		stss = append(stss, failures...)
		if !retry {
			break
		}
		backoff = time.Duration(float64(backoff) * taskBackoffMultiplier)
		if rn == nil {
			rn = rng()
		}
		backoff +=
			time.Duration((rn.Float64()*taskBackoffJitter*2 - taskBackoffJitter) * float64(backoff))
	}
	return status.WithSuppressed(stss[0], stsSliceToErrSlice(stss[1:])...)
}

func runTaskOnce(ctx context.Context, task Task, nextBackoff time.Duration, now func() time.Time) (
	success bool, retry bool, _ []status.S) {
	select {
	case <-ctx.Done():
		return false, false, []status.S{status.From(ctx.Err())}
	default:
	}
	sts := task.Run(ctx)
	if sts == nil {
		return true, false, nil
	}
	retryable, ok := unwrapTaskStatus(sts).(db.Retryable)
	if !(ok && retryable.CanRetry()) {
		return false, false, []status.S{sts}
	}
	sleep := nextBackoff
	if deadline, deadlineok := ctx.Deadline(); deadlineok {
		if wakeup := now().Add(sleep); wakeup.After(deadline) {
			deadlinests := status.DeadlineExceededf(
				nil, "not enough time to complete %T (%v < %v)", task, deadline, wakeup)
			return false, false, []status.S{sts, deadlinests}
		}
	}
	select {
	case <-time.After(sleep):
	case <-ctx.Done():
		return false, false, []status.S{sts, status.From(ctx.Err())}
	}
	return false, true, []status.S{sts}
}

func unwrapTaskStatus(sts status.S) error {
	var cause error = sts
	for {
		if s, ok := cause.(status.S); ok {
			cause = s.Cause()
		} else {
			break
		}
	}
	return cause
}
