package main

import (
	"database/sql"
	"flag"
	"log"
	"os"

	"pixur.org/pixur/schema"

	_ "github.com/go-sql-driver/mysql"
)

var (
	mysqlConfig = flag.String("mysql_config", "", "The default mysql config")
	pixPath     = flag.String("pix_path", "../pix", "Default picture storage directory")
)

func run() error {
	db, err := sql.Open("mysql", *mysqlConfig)
	if err != nil {
		return err
	}

	var startID int64
	for {
		pics, err := getNextPics(db, startID)
		if err != nil {
			return err
		}
		if len(pics) <= 1 {
			break
		}

		for _, pic := range pics {
			if err := fixThumbnail(pic); err != nil {
				return err
			}
		}

		startID = pics[len(pics)-1].Id
	}

	log.Println("Done!")
	return nil
}

func fixThumbnail(pic *pixur.Pic) error {
	_, err := os.Stat(pic.ThumbnailPath(*pixPath))
	if os.IsNotExist(err) {
		log.Println("Fixing ", pic.Id, pic.Mime)
		f, err := os.Open(pic.Path(*pixPath))
		if err != nil {
			return err
		}
		defer f.Close()

		// we dont want to actually modify pic
		img, err := pixur.FillImageConfig(f, new(pixur.Pic))
		if err != nil {
			return err
		}

		thumb := pixur.MakeThumbnail(img)

		if err := pixur.SaveThumbnail(thumb, pic, *pixPath); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

func getNextPics(db *sql.DB, startID int64) ([]*pixur.Pic, error) {
	task := &pixur.ReadIndexPicsTask{
		DB:      db,
		StartID: startID,
	}
	runner := new(pixur.TaskRunner)
	if err := runner.Run(task); err != nil {
		return nil, err
	}

	return task.Pics, nil
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		log.Println(err)
	}
}
