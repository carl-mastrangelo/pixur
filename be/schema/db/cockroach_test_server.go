package db

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

func (s *testCockroachPostgresServer) start(ctx context.Context) error {
	var defers []func()
	defer func() {
		for i := len(defers) - 1; i >= 0; i-- {
			defers[i]()
		}
	}()
	testdir, err := ioutil.TempDir("", "postgrespixurtest")
	if err != nil {
		return err
	}
	defers = append(defers, func() {
		if err := os.RemoveAll(testdir); err != nil {
			log.Println("failed to remove testdir while cleaning up", err)
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
		return err
	}
	defers = append(defers, func() {
		if err := stderr.Close(); err != nil {
			log.Println("failed to close stderr while cleaning up", err)
		}
	})

	stdout, err := os.Create(filepath.Join(testdir, "STDOUT"))
	if err != nil {
		return err
	}
	defers = append(defers, func() {
		if err := stdout.Close(); err != nil {
			log.Println("failed to close stdout while cleaning up", err)
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
		return err
	}
	defers = append(defers, func() {
		if err := cmd.Process.Kill(); err != nil {
			log.Println("failed to kill process while cleaning up", err)
		}
	})

	select {
	case parts := <-doneparts:
		s.user, s.host, s.port = parts[0], parts[1], parts[2]
	case <-ctx.Done():
		return ctx.Err()
	}

	s.cmd = cmd
	s.testdir = testdir
	defers = nil
	return nil
}

func (s *testCockroachPostgresServer) stop() error {
	var lasterr error
	if err := s.cmd.Process.Kill(); err != nil {
		lasterr = err
		log.Println("failed to kill process", err)
	}
	if err := s.cmd.Stderr.(io.Closer).Close(); err != nil {
		lasterr = err
		log.Println("failed to close stderr", err)
	}
	if err := s.cmd.Stdout.(io.Closer).Close(); err != nil {
		lasterr = err
		log.Println("failed to close stdout", err)
	}
	if err := os.RemoveAll(s.testdir); err != nil {
		lasterr = err
		log.Println("failed to remove testdir", err)
	}
	return lasterr
}
