package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"

	"pixur.org/pixur"
	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
)

var (
	config      = flag.String("config", ".config.json", "The default configuration file")
	mysqlConfig = flag.String("mysql_config", "", "The default mysql config")
	pixPath     = flag.String("pix_path", "pix", "Default picture storage directory")
)

func run(db *sql.DB) error {
	// ADD Work HERE
	var i int64

	stmt, err := schema.PicPrepare("SELECT * FROM_ WHERE %s > ? ORDER BY %s LIMIT 1000 FOR UPDATE;",
		db, schema.PicColId, schema.PicColId)
	if err != nil {
		return err
	}
	for {
		pics, err := schema.FindPics(stmt, i)
		if err != nil {
			return err
		}

		if len(pics) == 0 {
			break
		}

		for _, p := range pics {
			if err := fixIdents(p, db); err != nil {
				return err
			}
			i = p.PicId
		}
		log.Printf("Finished %d", i)
	}
	return nil
}

func fixIdents(p *schema.Pic, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	p.Sha256Hash = nil
	if err := p.Update(tx); err != nil {
		return err
	}

	stmt, err := schema.PicIdentifierPrepare("SELECT * FROM_ WHERE %s = ? FOR UPDATE;",
		tx, schema.PicIdentColPicId)
	if err != nil {
		return err
	}

	idents, err := schema.FindPicIdentifiers(stmt, p.PicId)
	if err != nil {
		return err
	}
	for _, ident := range idents {
		if err := ident.Delete(tx); err != nil {
			return err
		}
	}

	f, err := os.Open(p.Path(*pixPath))
	if err != nil {
		return err
	}
	defer f.Close()

	newIdents, err := generatePicIdentities(f)
	if err != nil {
		return err
	}

	for typ, val := range newIdents {
		ident := &schema.PicIdentifier{
			PicId: p.PicId,
			Type:  typ,
			Value: val,
		}
		if err := ident.Insert(tx); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func generatePicIdentities(f io.ReadSeeker) (map[schema.PicIdentifier_Type][]byte, error) {
	if _, err := f.Seek(0, os.SEEK_SET); err != nil {
		return nil, status.InternalError(err.Error(), err)
	}
	defer f.Seek(0, os.SEEK_SET)
	h1 := sha256.New()
	h2 := sha1.New()
	h3 := md5.New()

	w := io.MultiWriter(h1, h2, h3)

	if _, err := io.Copy(w, f); err != nil {
		return nil, status.InternalError(err.Error(), err)
	}
	return map[schema.PicIdentifier_Type][]byte{
		schema.PicIdentifier_SHA256: h1.Sum(nil),
		schema.PicIdentifier_SHA1:   h2.Sum(nil),
		schema.PicIdentifier_MD5:    h3.Sum(nil),
	}, nil
}

func getConfig(path string) (*pixur.Config, error) {
	var config = new(pixur.Config)
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

func main() {
	flag.Parse()

	c, err := getConfig(*config)
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("mysql", c.MysqlConfig)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	if err := run(db); err != nil {
		log.Fatal(err)
	} else {
		log.Println("Success!")
	}
}
