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
	retryTasksCounter   = expvar.NewInt("PixurRetryTasks")
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
	if r != nil && r.run != nil {
		return r.run(ctx, task)
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

	runOnce := func(nextBackoff time.Duration) (bool, bool, []status.S) {
		return runTaskOnce(ctx, task, nextBackoff, now)
	}
	retry := &taskRetryAdapter{
		maxTries:     taskMaxRetries,
		initialDelay: taskInitialBackoffDelay,
		rng:          rng,
		multiplier:   taskBackoffMultiplier,
		jitter:       taskBackoffJitter,
	}
	sts := runTask(runOnce, retry, logger)
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

type onceRunner func(nextBackoff time.Duration) (success, retry bool, stss []status.S)

// TODO: test
func runTask(runOnce onceRunner, retry *taskRetryAdapter, logger *log.Logger) (stscap status.S) {
	var stss []status.S
	defer func() {
		if stscap == nil && len(stss) != 0 && logger != nil {
			warnsts := status.Unknownf(nil, "Encountered %d errors runing task", len(stss))
			warnsts = status.WithSuppressed(warnsts, stsSliceToErrSlice(stss)...)
			logger.Println(warnsts)
		}
	}()

	for {
		backoff, proceed := retry.Next()
		if !proceed {
			break
		}
		success, retry, failures := runOnce(backoff)
		if success {
			return nil
		}
		stss = append(stss, failures...)
		if !retry {
			break
		}
		retryTasksCounter.Add(1)
	}
	return status.WithSuppressed(stss[0], stsSliceToErrSlice(stss[1:])...)
}

type taskRetryAdapter struct {
	currentTry, maxTries         int
	currentBackoff, initialDelay time.Duration
	rng                          func() *rand.Rand
	rn                           *rand.Rand
	multiplier, jitter           float64
}

func (r *taskRetryAdapter) Next() (time.Duration, bool) {
	if r.currentTry >= r.maxTries {
		return 0, false
	}
	const maxInt = int((^uint(0)) >> 1)
	if r.maxTries != maxInt {
		r.currentTry++
	}
	if r.currentBackoff == 0 {
		r.currentBackoff = r.initialDelay
		return r.currentBackoff, true
	}
	back := float64(r.currentBackoff) * r.multiplier
	var uniform float64
	if r.rng != nil && r.rn == nil {
		r.rn = r.rng()
	}
	if r.rn != nil {
		uniform = r.rn.Float64()
	} else {
		uniform = 0.5
	}
	back += (uniform*2 - 1) * r.jitter * back
	if back > float64(math.MaxInt64) {
		r.currentBackoff = time.Duration(math.MaxInt64)
	} else {
		r.currentBackoff = time.Duration(back)
	}
	return r.currentBackoff, true
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
