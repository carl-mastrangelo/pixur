package main

import (
	"log"
	"os"
	"time"

	"pixur.org/pixur/image"
	"pixur.org/pixur/schema"
	"pixur.org/pixur/tools/batch"
)

func main() {
	err := batch.ForEachPic(func(p *schema.Pic, sc *batch.ServerConfig, err error) error {
		if err != nil {
			return err
		}
		if p.HardDeleted() {
			return nil
		}
		if p.Mime != schema.Pic_GIF && p.Mime != schema.Pic_WEBM {
			return nil
		}

		tx, err := sc.DB.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		stmt, err := schema.PicPrepare("SELECT * FROM_ WHERE %s = ? FOR UPDATE;", tx, schema.PicColId)
		if err != nil {
			return err
		}
		defer stmt.Close()

		// Look up pic for updating.
		p, err = schema.LookupPic(stmt, p.PicId)
		if err != nil {
			return err
		}

		f, err := os.Open(p.Path(sc.PixPath))
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := image.FillImageConfig(f, p); err != nil {
			log.Println("Bad", p)
			return err
		}
		p.SetModifiedTime(time.Now())

		if err := p.Update(tx); err != nil {
			return err
		}
		log.Println("Finished", p.GetVarPicID())

		return nil
	})

	if err != nil {
		log.Println(err)
	}
}
