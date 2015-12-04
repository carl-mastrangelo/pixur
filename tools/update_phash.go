package main

import (
	"encoding/binary"
	"image"
	"log"
	"os"

	"pixur.org/pixur/imaging"
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

		tx, err := sc.DB.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		stmt, err := schema.PicPrepare("SELECT * FROM_ WHERE %s = ? LOCK IN SHARE MODE;", tx, schema.PicColId)
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

		img, err := imaging.FillImageConfig(f, p)
		if err != nil {
			log.Println("Bad", p)
			return err
		}

		pIdent := getPerceptualHash(p, img)
		if err := pIdent.Insert(tx); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		binary.BigEndian.Uint64(pIdent.Value)

		log.Println("Finished", p.PicId, p.GetVarPicID(), binary.BigEndian.Uint64(pIdent.Value))

		return nil
	})

	if err != nil {
		log.Println(err)
	}
}

func getPerceptualHash(p *schema.Pic, im image.Image) *schema.PicIdentifier {
	hash, inputs := imaging.PerceptualHash0(im)
	return &schema.PicIdentifier{
		PicId:      p.PicId,
		Type:       schema.PicIdentifier_DCT_0,
		Value:      hash,
		Dct0Values: inputs,
	}
}
