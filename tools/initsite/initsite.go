package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"
	"time"

	"pixur.org/pixur/schema"
	sdb "pixur.org/pixur/schema/db"
	tab "pixur.org/pixur/schema/tables"
	"pixur.org/pixur/server/config"
	"pixur.org/pixur/tasks"

	"golang.org/x/crypto/ssh/terminal"
)

var (
	initTables    = flag.Bool("init_tables", false, "If set, initialize database tables")
	createNewUser = flag.Bool("create_first_user", true, "If set, creates a new")
)

func run() error {
	log.Println("Opening Database")
	db, err := sdb.Open(config.Conf.DbName, config.Conf.DbConfig)
	if err != nil {
		return err
	}
	defer db.Close()
	if *initTables {
		var stmts []string
		stmts = append(stmts, tab.SqlTables[db.Adapter().Name()]...)
		stmts = append(stmts, tab.SqlInitTables[db.Adapter().Name()]...)
		log.Println("Initializing tables")
		if err := db.InitSchema(stmts); err != nil {
			return err
		}
	}
	if *createNewUser {
		r := bufio.NewReader(os.Stdin)
		fmt.Print("Admin Ident (e.g. foo@bar.com): ")
		ident, err := r.ReadString('\n')
		if err != nil {
			return err
		}

		fmt.Print("Admin Secret (e.g. 12345): ")
		secret, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return err
		}

		task := &tasks.CreateUserTask{
			DB:  db,
			Now: time.Now,

			Ident:  strings.TrimSpace(ident),
			Secret: string(secret),
			Ctx:    context.Background(),
			Capability: append(schema.UserNewCap,
				schema.User_PIC_SOFT_DELETE,
				schema.User_USER_UPDATE_CAPABILITY),
		}

		// Presumably there is nobody in the database yet, so we need to temporarily relax permissions
		// on the anonymous user.
		oldcap := schema.AnonymousUser.Capability
		schema.AnonymousUser.Capability = []schema.User_Capability{schema.User_USER_CREATE}
		sts := new(tasks.TaskRunner).Run(task)
		schema.AnonymousUser.Capability = oldcap
		if sts != nil {
			return sts
		}

		log.Println("Created user")
	}

	return nil
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		log.Println(err)
	}
}
