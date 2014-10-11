package testing

import (
	"bufio"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func GetDB() (*sql.DB, error) {
	once.Do(func() {
		db, initErr = initDB()
    if initErr != nil {
      CleanUp()
    }
	})
	return db, initErr
}

func CleanUp() {
  for i := len(cleanUpActions) - 1; i >=0; i-- {
    cleanUpActions[i]()
  }
}

var (
	db      *sql.DB
	initErr error
  cleanUpActions []func() 
	once    *sync.Once = new(sync.Once)
)

func initDB() (*sql.DB, error) {
	datadir, err := ioutil.TempDir("", "datadir")
	if err != nil {
		return nil, err
	}
  cleanUpActions = append(cleanUpActions, func() {
    os.RemoveAll(datadir)
  })

	socket, err := ioutil.TempFile("", "socket")
	if err != nil {
		return nil, err
	}
  cleanUpActions = append(cleanUpActions, func() {
    os.Remove(socket.Name())
  })

	pidFile, err := ioutil.TempFile("", "pidFile")
	if err != nil {
		return nil, err
	}
  cleanUpActions = append(cleanUpActions, func() {
    os.Remove(pidFile.Name())
  })

	cmd := exec.Command("mysqld",
		"--datadir", datadir,
		"--socket", socket.Name(),
		"--pid-file", pidFile.Name(),
		"--skip-grant-tables",
		"--skip-networking")
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
  cleanUpActions = append(cleanUpActions, func() {
    stderr.Close()
  })
	ready := make(chan error)

	go func() {
		r := bufio.NewReader(stderr)
		defer close(ready)
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				ready <- err
				return
			}
			if strings.Contains(line, "mysqld: ready for connections") {
				return
			}
		}
	}()

	if err := cmd.Start(); err != nil {
		return nil, err
	}
  cleanUpActions = append(cleanUpActions, func() {
    cmd.Process.Kill()
  })

	select {
	case err := <-ready:
		if err != nil {
			return nil, err
		}
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("Failed to start server")
	}

	db, err := sql.Open("mysql", "unix("+socket.Name()+")/")
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec("CREATE DATABASE IF NOT EXISTS test;"); err != nil {
		return nil, err
	}
	if _, err := db.Exec("USE test;"); err != nil {
		return nil, err
	}

	return db, nil
}
