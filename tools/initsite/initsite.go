package main // import "pixur.org/pixur/tools/initsite"

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/golang/protobuf/proto"
	"golang.org/x/crypto/ssh/terminal"

	"pixur.org/pixur/be/schema"
	sdb "pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	beconfig "pixur.org/pixur/be/server/config"
	"pixur.org/pixur/be/tasks"
	feconfig "pixur.org/pixur/fe/server/config"
)

var (
	initTables    = flag.Bool("init_tables", false, "If set, initialize database tables")
	createNewUser = flag.Bool("create_first_user", true, "If set, creates a new")
)

func buildFeConfig(input *bufio.Reader) (*feconfig.Config, error) {
	config := new(feconfig.Config)

	fmt.Printf("Listening http specification (default: %s)\n  ", feconfig.DefaultValues.HttpSpec)
	if val := read(input); val != "" {
		config.HttpSpec = val
	} else {
		config.HttpSpec = feconfig.DefaultValues.HttpSpec
	}

	fmt.Printf("Backend grpc specification: (default: %s)\n  ", feconfig.DefaultValues.PixurSpec)
	if val := read(input); val != "" {
		config.PixurSpec = val
	} else {
		config.PixurSpec = feconfig.DefaultValues.PixurSpec
	}

	fmt.Printf("Use insecure cookies: (default: %v)\n  ", feconfig.DefaultValues.Insecure)
	if val := read(input); val != "" {
		v, err := strconv.ParseBool(val)
		if err != nil {
			return nil, err
		}
		config.Insecure = v
	} else {
		config.Insecure = feconfig.DefaultValues.Insecure
	}

	fmt.Printf("Http Root: (default: %s)\n  ", feconfig.DefaultValues.HttpRoot)
	if val := read(input); val != "" {
		config.HttpRoot = val
	} else {
		config.HttpRoot = feconfig.DefaultValues.HttpRoot
	}
	return config, nil
}

func buildBeConfig(input *bufio.Reader, rand io.Reader) (*beconfig.Config, error) {
	config := new(beconfig.Config)

	fmt.Printf("Database name: (default: %s)\n  ", beconfig.DefaultValues.DbName)
	if val := read(input); val != "" {
		config.DbName = val
	} else {
		config.DbName = beconfig.DefaultValues.DbName
	}

	fmt.Printf("Database specification: (default: %s)\n  ", beconfig.DefaultValues.DbConfig)
	if val := read(input); val != "" {
		config.DbConfig = val
	} else {
		config.DbConfig = beconfig.DefaultValues.DbConfig
	}

	fmt.Printf("Listen network: (default: %s)\n  ", beconfig.DefaultValues.ListenNetwork)
	if val := read(input); val != "" {
		config.ListenNetwork = val
	} else {
		config.ListenNetwork = beconfig.DefaultValues.ListenNetwork
	}

	fmt.Printf("Listen address: (default: %s)\n  ", beconfig.DefaultValues.ListenAddress)
	if val := read(input); val != "" {
		config.ListenAddress = val
	} else {
		config.ListenAddress = beconfig.DefaultValues.ListenAddress
	}

	fmt.Printf("Picture storage path: (default: %s)\n  ", beconfig.DefaultValues.PixPath)
	if val := read(input); val != "" {
		config.PixPath = val
	} else {
		config.PixPath = beconfig.DefaultValues.PixPath
	}

	tokensecretdata := make([]byte, 16)
	if n, err := rand.Read(tokensecretdata); err != nil {
		return nil, err
	} else if n != len(tokensecretdata) {
		panic("rand did not produce any data")
	}

	// This doesn't matter, just make it plain text
	tokensecretdst := make([]byte, hex.EncodedLen(len(tokensecretdata)))
	hex.Encode(tokensecretdst, tokensecretdata)
	config.TokenSecret = string(tokensecretdst)
	return config, nil
}

func read(r *bufio.Reader) string {
	data, err := r.ReadString('\n')
	if err != nil {
		panic(err)
	}
	return string(data[:len(data)-1])
}

func readbool(r *bufio.Reader, def bool) bool {
	val := read(r)
	switch val {
	case "":
		return def
	case "y":
		return true
	case "yes":
		return true
	case "Yes":
		return true
	case "n":
		return false
	case "no":
		return false
	case "No":
		return false
	default:
		panic(val)
	}
}

func run(args []string) error {
	fmt.Println("Initializing Pixur installation")
	fmt.Println()
	r := bufio.NewReader(os.Stdin)

	var beconf *beconfig.Config
	fmt.Println("Create Backend configuration? (default: y)")
	if y := readbool(r, true); y {
		err := (func() error {

			config, err := buildBeConfig(r, rand.Reader)
			if err != nil {
				return err
			}
			beconf = config
			fmt.Println("Creating", beconfig.DefaultConfigPath)
			f, err := os.OpenFile(beconfig.DefaultConfigPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
			if err != nil {
				return err
			}
			defer f.Close()
			if err := proto.MarshalText(f, config); err != nil {
				return err
			}
			return nil
		})()
		if err != nil {
			return err
		}
		fmt.Println("Successfully created", beconfig.DefaultConfigPath)
	}

	fmt.Println("Create Frontend configuration? (default: y)")
	if y := readbool(r, true); y {
		err := (func() error {
			config, err := buildFeConfig(r)
			if err != nil {
				return err
			}
			fmt.Println("Creating", feconfig.DefaultConfigPath)
			f, err := os.OpenFile(feconfig.DefaultConfigPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
			if err != nil {
				return err
			}
			defer f.Close()
			if err := proto.MarshalText(f, config); err != nil {
				return err
			}
			return nil
		})()
		if err != nil {
			return err
		}
		fmt.Println("Successfully created", feconfig.DefaultConfigPath)
	}

	if err := flag.CommandLine.Parse(args); err != nil {
		return err
	}
	if beconf == nil {
		beconf = beconfig.Conf
	}
	fmt.Println("Opening Database " + beconf.DbConfig)
	db, err := sdb.Open(beconf.DbName, beconf.DbConfig)
	if err != nil {
		return err
	}
	defer db.Close()

	ctx := context.Background()
	fmt.Println("Create initial tables? (default: y)")
	if y := readbool(r, true); y {
		var stmts []string
		stmts = append(stmts, tab.SqlTables[db.Adapter().Name()]...)
		stmts = append(stmts, tab.SqlInitTables[db.Adapter().Name()]...)
		fmt.Println("Initializing tables")
		if err := db.InitSchema(ctx, stmts); err != nil {
			return err
		}
		fmt.Println("Successfully initialized tables")
	}

	fmt.Println("Create admin user? (default: y)")
	if y := readbool(r, true); y {
		fmt.Print("Admin Ident (e.g. foo@bar.com): ")
		ident := read(r)

		fmt.Print("Admin Secret Password (e.g. 12345): ")
		secret, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return err
		}

		conf := schema.GetDefaultConfiguration()
		if conf.NewUserCapability == nil {
			conf.NewUserCapability = &schema.Configuration_CapabilitySet{}
		}
		conf.NewUserCapability.Capability = append(conf.NewUserCapability.Capability,
			schema.User_PIC_SOFT_DELETE, schema.User_USER_UPDATE_CAPABILITY)

		task := &tasks.CreateUserTask{
			Beg: db,
			Now: time.Now,

			Ident:      ident,
			Secret:     string(secret),
			Capability: conf.NewUserCapability.Capability,
		}

		// Presumably there is nobody in the database yet, so we need to temporarily relax permissions
		// on the anonymous user.
		conf.AnonymousCapability = &schema.Configuration_CapabilitySet{
			Capability: []schema.User_Capability{
				schema.User_USER_CREATE,
			},
		}

		ctx = tasks.CtxFromTestConfig(ctx, conf)
		sts := new(tasks.TaskRunner).Run(ctx, task)
		if sts != nil {
			return sts
		}

		fmt.Println("\nCreated admin user")
	}

	return nil
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
