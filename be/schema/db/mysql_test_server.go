package db

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"pixur.org/pixur/be/status"
)

type mysqlTestServer struct {
	testdir, datadir, socket, pidfile string
	cmd                               *exec.Cmd
	stderr, stdout                    *os.File
	active, total                     int
}

func (mts *mysqlTestServer) tearDownEnvOnErr(stscap *status.S) {
	if *stscap != nil {
		teardownsts := mts.tearDownEnv()
		if teardownsts != nil {
			*stscap = status.WithSuppressed(*stscap, teardownsts)
		}
	}
}

func (mts *mysqlTestServer) tearDownEnv() status.S {
	var sts status.S
	if mts.stdout != nil {
		if err := mts.stdout.Close(); err != nil {
			replaceOrSuppress(&sts, status.Unknown(err, "can't close stdout"))
		}
		mts.stdout = nil
	}

	if mts.stderr != nil {
		if err := mts.stderr.Close(); err != nil {
			replaceOrSuppress(&sts, status.Unknown(err, "can't close err"))
		}
		mts.stderr = nil
	}

	if mts.testdir != "" {
		if err := os.RemoveAll(mts.testdir); err != nil {
			replaceOrSuppress(&sts, status.Unknown(err, "can't remove testdir"))
		}
		mts.testdir = ""
	}
	return sts
}

func (mts *mysqlTestServer) setupEnv() (stscap status.S) {
	defer mts.tearDownEnvOnErr(&stscap)
	testdir, err := ioutil.TempDir("", "mysqlpixurtest")
	if err != nil {
		return status.Unknown(err, "can't create tempdir")
	}
	mts.testdir = testdir

	datadir := filepath.Join(testdir, "datadir")
	if err := os.Mkdir(datadir, 0700); err != nil {
		return status.Unknown(err, "can't create datadir")
	}
	mts.datadir = datadir

	mts.socket = filepath.Join(testdir, "socket")
	mts.pidfile = filepath.Join(testdir, "pidfile")

	stderr, err := os.Create(filepath.Join(testdir, "STDERR"))
	if err != nil {
		return status.Unknown(err, "can't create stderr")
	}
	mts.stderr = stderr

	stdout, err := os.Create(filepath.Join(testdir, "STDOUT"))
	if err != nil {
		return status.Unknown(err, "can't create stdout")
	}
	mts.stdout = stdout

	return nil
}

func (mts *mysqlTestServer) stopOnErr(stscap *status.S) {
	if *stscap != nil {
		if stopsts := mts.stop(); stopsts != nil {
			*stscap = status.WithSuppressed(*stscap, stopsts)
		}
	}
}

func (mts *mysqlTestServer) stop() status.S {
	if mts.cmd != nil {
		cmd := mts.cmd
		mts.cmd = nil
		if cmd.Process != nil {
			if err := cmd.Process.Kill(); err != nil {
				return status.Unknown(err, "can't kill process")
			}
		}
	}
	return nil
}

type scanUntilReady struct {
	io.Writer
	match int
	done  chan (struct{})
}

var mysqlServerReady = []byte("mysqld: ready for connections")

func (s *scanUntilReady) Write(data []byte) (int, error) {
	if s.match != len(mysqlServerReady) {
		for _, b := range data {
			if mysqlServerReady[s.match] == b {
				s.match++
			} else if mysqlServerReady[0] == b {
				s.match = 1
			} else {
				s.match = 0
			}
			if s.match == len(mysqlServerReady) {
				close(s.done)
				break
			}
		}
	}
	return s.Writer.Write(data)
}

func (mts *mysqlTestServer) start() (stscap status.S) {
	defer mts.stopOnErr(&stscap)
	mts.cmd = exec.Command(
		"mysqld",
		"--datadir", mts.datadir,
		"--socket", mts.socket,
		"--pid-file", mts.pidfile,
		"--secure-file-priv", "",
		"--skip-grant-tables",
		"--skip-networking",
	)

	s := &scanUntilReady{
		Writer: mts.stderr,
		done:   make(chan struct{}),
	}
	mts.cmd.Stderr = s
	mts.cmd.Stdout = mts.stdout

	if err := mts.cmd.Start(); err != nil {
		return status.Unknown(err, "can't start server")
	}

	select {
	case <-time.After(3 * time.Second):
		return status.InternalError(nil, "failed to start")
	case <-s.done:
	}

	return nil
}
