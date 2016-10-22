package db

import (
	"errors"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type mysqlTestServer struct {
	testdir, datadir, socket, pidfile string
	cmd                               *exec.Cmd
	stderr, stdout                    *os.File
	active, total                     int
}

func (mts *mysqlTestServer) tearDownEnvOnErr(errcap *error) {
	if *errcap != nil {
		mts.tearDownEnv()
	}
}

func (mts *mysqlTestServer) tearDownEnv() {
	if mts.stdout != nil {
		if err := mts.stdout.Close(); err != nil {
			log.Println(err)
		}
		mts.stdout = nil
	}

	if mts.stderr != nil {
		if err := mts.stderr.Close(); err != nil {
			log.Println(err)
		}
		mts.stderr = nil
	}

	if mts.testdir != "" {
		if err := os.RemoveAll(mts.testdir); err != nil {
			log.Println(err)
		}
		mts.testdir = ""
	}
}

func (mts *mysqlTestServer) setupEnv() (errcap error) {
	defer mts.tearDownEnvOnErr(&errcap)
	testdir, err := ioutil.TempDir("", "mysqlpixurtest")
	if err != nil {
		return err
	}
	mts.testdir = testdir

	datadir := filepath.Join(testdir, "datadir")
	if err := os.Mkdir(datadir, 0700); err != nil {
		return err
	}
	mts.datadir = datadir

	mts.socket = filepath.Join(testdir, "socket")
	mts.pidfile = filepath.Join(testdir, "pidfile")

	stderr, err := os.Create(filepath.Join(testdir, "STDERR"))
	if err != nil {
		return err
	}
	mts.stderr = stderr

	stdout, err := os.Create(filepath.Join(testdir, "STDOUT"))
	if err != nil {
		return err
	}
	mts.stdout = stdout

	return nil
}

func (mts *mysqlTestServer) stopOnErr(errcap *error) {
	if *errcap != nil {
		mts.stop()
	}
}

func (mts *mysqlTestServer) stop() {
	if mts.cmd != nil {
		if mts.cmd.Process != nil {
			if err := mts.cmd.Process.Kill(); err != nil {
				log.Println(err)
			}
		}
		mts.cmd = nil
	}
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

func (mts *mysqlTestServer) start() (errcap error) {
	defer mts.stopOnErr(&errcap)
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
		return err
	}

	select {
	case <-time.After(3 * time.Second):
		return errors.New("failed to start")
	case <-s.done:
	}

	return nil
}
