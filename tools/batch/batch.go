package batch

import (
	"database/sql"
	"encoding/json"
	"flag"
	"log"
	"os"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/server"
)

// copy of starter/
var (
	config = flag.String("config", ".config.json", "The default configuration file")
)

type ServerConfig struct {
	DB      *sql.DB
	PixPath string
}

var serverConfig *ServerConfig

func GetConfig() (*ServerConfig, error) {
	if !flag.Parsed() {
		flag.Parse()
	}
	if serverConfig != nil {
		return serverConfig, nil
	}

	c, err := getConfig(*config)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("mysql", c.MysqlConfig)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)

	sc := &ServerConfig{
		DB:      db,
		PixPath: c.PixPath,
	}

	serverConfig = sc

	return sc, nil
}

func ForEachPic(fn func(*schema.Pic, *ServerConfig, error) error) error {
	sc, err := GetConfig()
	if err != nil {
		return err
	}
	stmt, err := schema.PicPrepare("SELECT * FROM_ WHERE %s > ? ORDER BY %s LIMIT 1000;",
		sc.DB, schema.PicColId, schema.PicColId)
	if err != nil {
		return err
	}

	var i int64
	var fnErr error
	for {
		pics, err := schema.FindPics(stmt, i)
		if err != nil {
			return err
		}
		if len(pics) == 0 {
			break
		}

		for _, p := range pics {
			fnErr = fn(p, sc, fnErr)
			i = p.PicId
		}
	}
	if fnErr != nil {
		return fnErr
	}
	return nil
}

func getConfig(path string) (*server.Config, error) {
	var config = new(server.Config)
	f, err := os.Open(path)

	if os.IsNotExist(err) {
		log.Println("Unable to open config file, using defaults")
		return config, nil
	} else if err != nil {
		return nil, err
	}
	defer f.Close()

	configDecoder := json.NewDecoder(f)
	if err := configDecoder.Decode(config); err != nil {
		return nil, err
	}

	return config, nil
}
