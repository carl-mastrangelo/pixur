package main

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

func run() error {
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
		"--skip-networking")
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	defer stderr.Close()
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
		return err
	}
	defer cmd.Process.Kill()

	select {
	case err := <-ready:
		if err != nil {
			return err
		}
	case <-time.After(5 * time.Second):
		return fmt.Errorf("Failed to start server")
	}

	db, err := sql.Open("mysql", "unix("+socket.Name()+")/")
	if err != nil {
		return err
	}

	if _, err := db.Exec("CREATE DATABASE IF NOT EXISTS test;"); err != nil {
		return err
	}
	if _, err := db.Exec("USE test;"); err != nil {
		return err
	}

	return nil
}

func main() {
	fmt.Println(run())
}
