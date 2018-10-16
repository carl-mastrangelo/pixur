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
	log.Println(p)
	if p.File != nil {
		return nil
	}

	if p.HardDeleted() {
		p.File = &schema.Pic_File{
			Index:         0,
			Size:          p.FileSize,
			Mime:          schema.Pic_File_Mime(p.Mime),
			Width:         p.Width,
			Height:        p.Height,
			CreatedTs:     p.CreatedTs,
			ModifiedTs:    p.ModifiedTs,
			AnimationInfo: p.AnimationInfo,
		}
		// We don't have any thumbnail info.
	} else {
		f, err := os.Open(p.Path(beconfig.Conf.PixPath))
		if err != nil {
			return status.InternalError(err, "can't open", p)
		}
		defer f.Close()
		im, sts := imaging.ReadImage(f)
		if sts != nil {
			return sts
		}
		defer im.Close()

		mime, sts := imageFormatToMime(im.Format())
		if sts != nil {
			return sts
		}
		if mime != schema.Pic_File_Mime(p.Mime) {
			return status.InternalError(nil, "wrong image type", p)
		}
		fi, err := f.Stat()
		if err != nil {
			return status.InternalError(err, "unable to stat file", p)
		}
		if fi.Size() != p.FileSize {
			return status.InternalError(nil, "bad file size", p)
		}
		w, h := im.Dimensions()
		if int64(w) != p.Width || int64(h) != p.Height {
			return status.InternalError(nil, "bad image dims", p)
		}
		dur, sts := im.Duration()
		if sts != nil {
			return sts
		}
		if (dur == nil) != (p.AnimationInfo == nil) {
			log.Println("animation mismatch", p, dur)
		}

		p.File = &schema.Pic_File{
			Index:         0,
			Size:          p.FileSize,
			Mime:          mime,
			Width:         p.Width,
			Height:        p.Height,
			CreatedTs:     p.CreatedTs,
			ModifiedTs:    p.ModifiedTs,
			AnimationInfo: p.AnimationInfo,
		}
		if dur != nil {
			p.File.AnimationInfo = &schema.AnimationInfo{
				Duration: schema.ToDurpb(*dur),
			}
		}

		// At this point, the file is correct.

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
		if err := os.Remove(p.ThumbnailPath(beconfig.Conf.PixPath)); err != nil {
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
