package db

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"pixur.org/pixur/be/status"
)

var cockroachReadyMatcher = regexp.MustCompile(`sql:[\s]*postgresql://([^@]+)@([^:]+):([\d]+)\?`)

type scanUntilCockroachReady struct {
	io.WriteCloser
	buf  bytes.Buffer
	done chan []string
}

func (scan *scanUntilCockroachReady) Write(data []byte) (int, error) {
	if scan.done != nil {
		scan.buf.Write(data)
		match := cockroachReadyMatcher.FindStringSubmatch(scan.buf.String())
		if len(match) > 0 {
			select {
			case scan.done <- match[1:]:
				close(scan.done)
				scan.done = nil
			default:
				panic("already done")
			}
		}
	}
	return scan.WriteCloser.Write(data)
}

func (scan *scanUntilCockroachReady) Close() error {
	if scan.done != nil {
		close(scan.done)
		scan.done = nil
	}
	return scan.WriteCloser.Close()
}

type testCockroachPostgresServer struct {
	cmd              *exec.Cmd
	testdir          string
	user, host, port string
}

func replaceOrSuppress(stscap *status.S, sts status.S) {
	if sts != nil {
		if *stscap == nil {
			*stscap = sts
		} else {
			*stscap = status.WithSuppressed(*stscap, sts)
		}
	}
}

func (s *testCockroachPostgresServer) start(ctx context.Context) (stscap status.S) {
	var defers []func()
	defer func() {
		for i := len(defers) - 1; i >= 0; i-- {
			defers[i]()
		}
	}()
	testdir, err := ioutil.TempDir("", "postgrespixurtest")
	if err != nil {
		return status.Unknown(err, "can't create temp dir")
	}
	defers = append(defers, func() {
		if err := os.RemoveAll(testdir); err != nil {
			replaceOrSuppress(&stscap, status.Unknown(err, "failed to remove testdir while cleaning up"))
		}
	})
	// Don't use context command as it would kill the cmd even after starting successfully
	cmd := exec.Command(
		"cockroach",
		"start",
		"--insecure",
		"--host", "localhost",
		"--port", "0",
		"--store", "path="+testdir,
		"--http-port", "0",
		"--http-host", "0.0.0.0",
	)

	stderr, err := os.Create(filepath.Join(testdir, "STDERR"))
	if err != nil {
		return status.Unknown(err, "can't create stderr")
	}
	defers = append(defers, func() {
		if err := stderr.Close(); err != nil {
			replaceOrSuppress(&stscap, status.Unknown(err, "failed to close stderr while cleaning up"))
		}
	})

	stdout, err := os.Create(filepath.Join(testdir, "STDOUT"))
	if err != nil {
		return status.Unknown(err, "can't create stdout")
	}
	defers = append(defers, func() {
		if err := stdout.Close(); err != nil {
			replaceOrSuppress(&stscap, status.Unknown(err, "failed to close stdout while cleaning up"))
		}
	})
	doneparts := make(chan []string, 1)
	scan := &scanUntilCockroachReady{
		WriteCloser: stdout,
		done:        doneparts,
	}

	cmd.Stderr = stderr
	cmd.Stdout = scan

	if err := cmd.Start(); err != nil {
		return status.Unknown(err, "can't start")
	}
	defers = append(defers, func() {
		if err := cmd.Process.Kill(); err != nil {
			replaceOrSuppress(&stscap, status.Unknown(err, "failed to kill process while cleaning up"))
		}
	})

	select {
	case parts := <-doneparts:
		s.user, s.host, s.port = parts[0], parts[1], parts[2]
	case <-ctx.Done():
		return status.From(ctx.Err())
	}

	s.cmd = cmd
	s.testdir = testdir
	defers = nil
	return nil
}

func (s *testCockroachPostgresServer) stop() status.S {
	var sts status.S
	if err := s.cmd.Process.Kill(); err != nil {
		replaceOrSuppress(&sts, status.Unknown(err, "failed to kill process"))
	}
	if err := s.cmd.Stderr.(io.Closer).Close(); err != nil {
		replaceOrSuppress(&sts, status.Unknown(err, "failed to close stderr"))
	}
	if err := s.cmd.Stdout.(io.Closer).Close(); err != nil {
		replaceOrSuppress(&sts, status.Unknown(err, "failed to close stdout"))
	}
	if err := os.RemoveAll(s.testdir); err != nil {
		replaceOrSuppress(&sts, status.Unknown(err, "failed to remove testdir"))
	}
	return sts
}
