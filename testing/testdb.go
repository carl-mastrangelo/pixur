package testing

import (
	"bufio"
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

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
)

func init() {
	go func() {
		initError := setupDB()
		for {
			dbs <- newDB{nil, initError}
		}
	}()
}

func GetDB() (*sql.DB, error) {
	db := <-dbs
	return db.db, db.err
}

func CleanUp() {
	close(done)
}

func setupDB() error {
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
			dbs <- newDB{db, nil}
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
