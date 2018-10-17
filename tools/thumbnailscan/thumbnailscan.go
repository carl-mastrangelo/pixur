package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"time"

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

	if !p.HardDeleted() && len(p.Thumbnail) != 1 {
		panic("bad pic")
	}

	if p.HardDeleted() || p.Thumbnail[0].Width == 192 {
		return nil
	}
	log.Println(p)

	if !p.HardDeleted() {
		path, sts := schema.PicFilePath(beconfig.Conf.PixPath, p.PicId, p.File.Mime)
		if sts != nil {
			return sts
		}

		f, err := os.Open(path)
		if err != nil {
			return status.InternalError(err, "can't open", p)
		}
		defer f.Close()
		im, sts := imaging.ReadImage(f)
		if sts != nil {
			return sts
		}
		defer im.Close()

		imt, sts := im.Thumbnail()
		if sts != nil {
			return sts
		}
		defer imt.Close()

		ft, err := ioutil.TempFile(beconfig.Conf.PixPath, "__oo")
		if err != nil {
			return status.InternalError(err, "can't make tempfile")
		}
		defer os.Remove(ft.Name())
		defer func() {
			if err := ft.Close(); err != nil {
				log.Println(err, "failed to close", p)
			}
		}()

		if sts := imt.Write(ft); sts != nil {
			return sts
		}
		fti, err := ft.Stat()
		if err != nil {
			return status.InternalError(err, "can't stat tempfile")
		}
		mimet, sts := imageFormatToMime(imt.Format())
		if sts != nil {
			return sts
		}
		wt, ht := imt.Dimensions()
		now := time.Now()
		nowts := schema.ToTspb(now)

		oldthumbnailpath, sts := schema.PicFileThumbnailPath(
			beconfig.Conf.PixPath, p.PicId, p.Thumbnail[0].Index, p.Thumbnail[0].Mime)
		if sts != nil {
			return sts
		}
		p.Thumbnail = nil

		p.Thumbnail = append(p.Thumbnail, &schema.Pic_File{
			Index:         0,
			Size:          fti.Size(),
			Mime:          mimet,
			Width:         int64(wt),
			Height:        int64(ht),
			CreatedTs:     nowts,
			ModifiedTs:    nowts,
			AnimationInfo: nil,
		})

		if err := ft.Close(); err != nil {
			return status.InternalError(err, "can't close old thumbnail")
		}

		newthumbnailpath, sts := schema.PicFileThumbnailPath(
			beconfig.Conf.PixPath, p.PicId, p.Thumbnail[0].Index, p.Thumbnail[0].Mime)
		if sts != nil {
			return sts
		}
		if err := os.Remove(oldthumbnailpath); err != nil {
			if os.IsNotExist(err) {
				log.Println(err, "No thumbnail found for ", p.PicId)
			} else {
				return status.InternalError(err, "can't remove old path")
			}
		}
		if err := os.Rename(ft.Name(), newthumbnailpath); err != nil {
			return status.InternalError(err, "can't rename thumbnail")
		}
	}
	log.Println(p)
	if err := j.UpdatePic(p); err != nil {
		return status.InternalError(err, "can't update pic")
	}

	if err := j.Commit(); err != nil {
		return status.InternalError(err, "can't commit")
	}

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
