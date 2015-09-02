package testing

import (
	"bufio"
	"database/sql"
	"fmt"
  "log"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"
  "sync"

	_ "github.com/go-sql-driver/mysql"
)

type newDB struct {
	db  *sql.DB
	err error
}

var (
	timeout = 5 * time.Second

	done = make(chan struct{})
	dbs  = make(chan newDB)
  
  initOnce sync.Once
)

func GetDB() (*sql.DB, error) {
  initOnce.Do(func() {
    go func() {
      initError := setupDB(done, dbs)
      if initError != nil {
        log.Println(initError)
      }
      close(dbs)
    }()
  })
	db, present := <-dbs
  if !present {
    return nil, fmt.Errorf("shutdown")
  }
  if db.err != nil {
    return nil, db.err
  }
	return db.db, nil
}

func CleanUp() {
	close(done)
}

func setupDB(done chan struct{}, dbs chan newDB) error {
	datadir, err := ioutil.TempDir("", "datadir")
	if err != nil {
		return err
	}
	defer os.RemoveAll(datadir)

	socket, err := ioutil.TempFile("", "socket")
	if err != nil {
		return err
	}
	defer os.Remove(socket.Name())

	pidFile, err := ioutil.TempFile("", "pidFile")
	if err != nil {
		return err
	}
	defer os.Remove(pidFile.Name())

	cmd := exec.Command("mysqld",
		"--datadir", datadir,
		"--socket", socket.Name(),
		"--pid-file", pidFile.Name(),
		"--skip-grant-tables",
		"--skip-networking",
	)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	defer stderr.Close()

	if err := cmd.Start(); err != nil {
		return err
	}
	defer cmd.Process.Kill()

	ready := make(chan error)
	stderrlines := make(chan string, 20)
	go func() {
		defer close(ready)
		s := bufio.NewScanner(stderr)
		for s.Scan() {
			line := s.Text()
			select {
			case stderrlines <- line:
			default:
			}
			if strings.Contains(line, "mysqld: ready for connections") {
				return
			}
		}
		if err := s.Err(); err != nil {
			ready <- err
		}
	}()

	select {
	case err := <-ready:
		if err != nil {
			return err
		}
	case <-time.After(timeout):
		lines := make([]string, 0, cap(stderrlines))
		for i := 0; i < cap(stderrlines); i++ {
			select {
			case line := <-stderrlines:
				lines = append(lines, line)
			default:
				break
			}
		}
		return fmt.Errorf("Failed to start server after %v\n\n%s", timeout, strings.Join(lines, "\n"))
	}

	for i := 0; ; i++ {
		select {
		case <-done:
			return nil
		default:
			db, err := getDb(socket.Name(), i)
			if err != nil {
				return err
			}
      select {
        case dbs <- newDB{db, nil}:
        case <-done:
        return nil
      }
			
		}
	}
}

func getDb(socketname string, id int) (*sql.DB, error) {
	db, err := sql.Open("mysql", "unix("+socketname+")/")
	if err != nil {
		return nil, err
	}

	dbName := fmt.Sprintf("testdb%d", id)

	if _, err := db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s;", dbName)); err != nil {
		return nil, err
	}

	// Close our connection, so we can reopen with the correct db name.  Other threads
	// will not use the correct database by default.
	if err := db.Close(); err != nil {
		return nil, err
	}

	db, err = sql.Open("mysql", "unix("+socketname+")/"+dbName)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
