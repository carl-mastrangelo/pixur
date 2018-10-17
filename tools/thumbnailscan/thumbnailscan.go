package main

import (
	"context"
	"flag"
	"log"

	"pixur.org/pixur/be/imaging"
	"pixur.org/pixur/be/schema"
	sdb "pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	beconfig "pixur.org/pixur/be/server/config"
	"pixur.org/pixur/be/status"
)

func handlePic(db sdb.DB, picId int64) status.S {
	j, err := tab.NewJob(context.Background(), db)
	if err != nil {
		return status.InternalError(err, "cant make job")
	}
	defer j.Rollback()

	pics, err := j.FindPics(sdb.Opts{Lock: sdb.LockWrite, Prefix: tab.PicsPrimary{&picId}})
	if err != nil {
		return status.InternalError(err, "can't find pics")
	}
	if len(pics) != 1 {
		return status.NotFound(nil, "can't find pic")
	}
	p := pics[0]

	if p.FileSize == 0 {
		return nil
	}
	p.FileSize = 0
	p.Mime = schema.Pic_UNKNOWN
	p.Width = 0
	p.Height = 0
	p.AnimationInfo = nil

	if err := j.UpdatePic(p); err != nil {
		return status.InternalError(err, "can't update pic")
	}

	if err := j.Commit(); err != nil {
		return status.InternalError(err, "can't commit")
	}
	log.Println(p.GetVarPicID())

	return nil
}

func run() error {
	olddb, err := sdb.Open(beconfig.Conf.DbName, beconfig.Conf.DbConfig)
	if err != nil {
		return err
	}
	defer olddb.Close()

	oldj, err := tab.NewJob(context.Background(), olddb)
	if err != nil {
		return err
	}
	iiidd := int64(0)
	pics, err := oldj.FindPics(sdb.Opts{
		Start: tab.PicsPrimary{&iiidd},
		Lock:  sdb.LockNone,
	})
	oldj.Rollback()
	if err != nil {
		return err
	}

	for _, p := range pics {
		if err := handlePic(olddb, p.PicId); err != nil {
			return err
		}
	}

	return nil
}

func imageFormatToMime(f imaging.ImageFormat) (schema.Pic_File_Mime, status.S) {
	switch {
	case f.IsJpeg():
		return schema.Pic_File_JPEG, nil
	case f.IsGif():
		return schema.Pic_File_GIF, nil
	case f.IsPng():
		return schema.Pic_File_PNG, nil
	case f.IsWebm():
		return schema.Pic_File_WEBM, nil
	default:
		return schema.Pic_File_UNKNOWN, status.InvalidArgument(nil, "Unknown image type", f)
	}
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		log.Println(err)
	}
}
