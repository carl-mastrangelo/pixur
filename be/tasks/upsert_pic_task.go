package tasks

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha512"
	"hash"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/golang/protobuf/ptypes"
	any "github.com/golang/protobuf/ptypes/any"

	"pixur.org/pixur/be/imaging"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/text"
)

type readerAtReadSeeker interface {
	io.ReadSeeker
	io.ReaderAt
}

// UpsertPicTask inserts or updates a pic with the provided information.
type UpsertPicTask struct {
	// Deps
	PixPath    string
	Beg        tab.JobBeginner
	HTTPClient *http.Client
	// os functions
	TempFile func(dir, prefix string) (*os.File, error)
	Rename   func(oldpath, newpath string) error
	MkdirAll func(path string, perm os.FileMode) error
	Now      func() time.Time
	Remove   func(name string) error

	// Inputs
	FileURL, FileURLReferrer string
	File                     readerAtReadSeeker
	Md5Hash                  []byte
	// If the name is absent, UpsertPicTask will try to derive a name automatically from the FileURL.
	FileName string

	// Ext is additional extra data associated with this pic.  If a key is present in both the
	// new pic and the existing pic, Upsert will fail.
	Ext map[string]*any.Any

	// Results
	CreatedPic *schema.Pic
}

func (t *UpsertPicTask) Run(ctx context.Context) (stscap status.S) {
	j, err := tab.NewJob(ctx, t.Beg)
	if err != nil {
		return status.Internal(err, "can't create job")
	}
	defer revert(j, &stscap)

	if sts := t.runInternal(ctx, j); sts != nil {
		// Don't delete old pics, as the commit may have actually succeeded.  A cron job will clean
		// this up.
		t.CreatedPic = nil
		return sts
	}

	// TODO: Check if this delete the original pic on a failed merge.
	if err := j.Commit(); err != nil {
		// Don't delete old pics, as the commit may have actually succeeded.  A cron job will clean
		// this up.
		t.CreatedPic = nil
		return status.Internal(err, "can't commit job")
	}
	return nil
}

func (t *UpsertPicTask) runInternal(ctx context.Context, j *tab.Job) (stscap status.S) {
	u, sts := requireCapability(ctx, j, schema.User_PIC_CREATE)
	if sts != nil {
		return sts
	}
	var userID = schema.AnonymousUserID
	if u != nil {
		userID = u.UserId
	}

	conf, sts := GetConfiguration(ctx)
	if sts != nil {
		return sts
	}
	var ext map[string]*any.Any
	if len(t.Ext) != 0 {
		if sts := validateCapability(u, conf, schema.User_PIC_EXTENSION_CREATE); sts != nil {
			return sts
		}
		ext = t.Ext
	}

	var loc, ref *url.URL
	if t.FileURL != "" {
		minUrlLen, maxUrlLen := confUrlLen(conf)
		if fu, sts := validateAndNormalizeURL(t.FileURL, minUrlLen, maxUrlLen); sts != nil {
			return sts
		} else {
			fu.Fragment = ""
			loc = fu
		}
		if t.FileURLReferrer != "" {
			if fu, sts := validateAndNormalizeURL(t.FileURLReferrer, minUrlLen, maxUrlLen); sts != nil {
				return sts
			} else {
				// leave fragment in place
				ref = fu
			}
		}
	}

	minFileNameLen, maxFileNameLen := confFileNameLen(conf)
	var filenames []string
	if len(t.FileName) > 0 {
		// TODO: trim whitespace
		if name, err := text.DefaultValidateNoNewlineAndNormalize(
			t.FileName, "filename", minFileNameLen, maxFileNameLen); err != nil {
			return status.From(err)
		} else {
			filenames = append(filenames, name)
		}
	}

	// TODO: change md5 hash to:
	// Check if md5 is present, and return ALREADY_EXISTS or PERMISSION_DENIED if deleted
	//   User is expected to call new method: add file_source / referrer, and add tags
	// If present, verify md5 sum after download
	// Also, asser md5 has is the right length before doing queries.

	var f *os.File
	var size int64
	var fileCleanup func(*status.S)
	if t.File != nil {
		var sts status.S
		if f, fileCleanup, size, sts = t.prepareLocalFile(ctx, t.File); sts != nil {
			return sts
		}
	} else if loc != nil {
		var dispositionName string
		var sts status.S
		f, fileCleanup, size, dispositionName, sts = t.prepareRemoteFile(ctx, loc, ref)
		if sts != nil {
			return sts
		}
		if int64(len(dispositionName)) >= minFileNameLen {
			if name, err := text.DefaultValidateNoNewlineAndNormalize(
				dispositionName, "disposition", minFileNameLen, maxFileNameLen); err != nil {
				_ = err // TODO: log this
			} else {
				filenames = append(filenames, name)
			}
		}
	} else {
		return status.InvalidArgument(nil, "no pic specified")
	}
	destroyTempFile := true
	defer func() {
		if destroyTempFile {
			fileCleanup(&stscap)
		}
	}()

	now := t.Now()
	nowts := schema.ToTspb(now)
	// TODO: test this
	pfs := &schema.Pic_FileSource{
		CreatedTs: nowts,
		UserId:    userID,
	}
	if loc != nil {
		furl := loc.String()
		pfs.Url = furl
		if ref != nil {
			pfs.Referrer = ref.String()
		}
	}
	if len(filenames) != 0 {
		pfs.Name = filenames[0]
	}

	hashes, sts :=
		generatePicHashes(io.NewSectionReader(f, 0, size), md5.New, sha1.New, sha512.New512_256)
	if sts != nil {
		// TODO: test this case
		return sts
	}
	md5Hash, sha1Hash, sha512_256Hash := hashes[0], hashes[1], hashes[2]
	if len(t.Md5Hash) != 0 && !bytes.Equal(t.Md5Hash, md5Hash) {
		return status.InvalidArgumentf(nil, "Md5 hash mismatch %x != %x", t.Md5Hash, md5Hash)
	}
	im, sts := imaging.ReadImage(ctx, io.NewSectionReader(f, 0, size))
	if sts != nil {
		return sts
	}
	defer im.Close()

	immime, sts := imageFormatToMime(im.Format())
	if sts != nil {
		return sts
	}
	var imanim *schema.AnimationInfo
	if dur, sts := im.Duration(); sts != nil {
		return sts
	} else if dur != nil {
		// TODO: test this check
		if immime == schema.Pic_File_WEBM && conf.MaxWebmDuration != nil {
			maxDur, err := ptypes.Duration(conf.MaxWebmDuration)
			if err != nil {
				return status.Internal(err, "can't parse max duration")
			}
			if *dur > maxDur {
				return status.InvalidArgumentf(nil, "duration %v exceeds max %v", *dur, maxDur)
			}
		}
		imanim = &schema.AnimationInfo{
			Duration: ptypes.DurationProto(*dur),
		}
	}

	// Still double check that the sha1 hash is not in use, even if the md5 one was
	// checked up at the beginning of the function.
	p, sts := findExistingPic(j, schema.PicIdent_SHA1, sha1Hash)
	if sts != nil {
		return sts
	}
	if p != nil {
		if p.HardDeleted() {
			if !p.GetDeletionStatus().Temporary {
				return status.InvalidArgument(nil, "Can't upload deleted pic.")
			}
			//  fall through, picture needs to be undeleted.
		} else {
			t.CreatedPic = p
			return mergePic(j, p, now, pfs, userID, ext)
		}
	} else {
		picID, err := j.AllocID()
		if err != nil {
			return status.Internal(err, "can't allocate id")
		}

		width, height := im.Dimensions()
		p = &schema.Pic{
			PicId: picID,
			File: &schema.Pic_File{
				Index:         0, // always 0 for main pic
				Size:          size,
				Mime:          schema.Pic_File_Mime(immime),
				Width:         int64(width),
				Height:        int64(height),
				AnimationInfo: imanim,
			},
			// ModifiedTime is set in mergePic
		}
		p.SetCreatedTime(now)
		p.File.CreatedTs = p.CreatedTs
		// Just reuse Created Time
		p.File.ModifiedTs = p.File.CreatedTs

		if err := j.InsertPic(p); err != nil {
			return status.Internal(err, "Can't insert")
		}
		if sts := insertPicHashes(j, p.PicId, md5Hash, sha1Hash, sha512_256Hash); sts != nil {
			return sts
		}
		if sts := insertPerceptualHash(j, p.PicId, im); sts != nil {
			return sts
		}
	}

	ft, err := t.TempFile(t.PixPath, "__")
	if err != nil {
		return status.Internal(err, "Can't create tempfile")
	}
	defer t.Remove(ft.Name())
	defer ft.Close()

	thumb, sts := im.Thumbnail()
	if sts != nil {
		return sts
	}
	if sts := thumb.Write(ft); sts != nil {
		return sts
	}

	thumbfi, err := ft.Stat()
	if err != nil {
		return status.Internal(err, "unable to stat thumbnail")
	}
	imfmime, sts := imageFormatToMime(thumb.Format())
	if sts != nil {
		return sts
	}
	var imfanim *schema.AnimationInfo
	if dur, sts := thumb.Duration(); sts != nil {
		return sts
	} else if dur != nil {
		imfanim = &schema.AnimationInfo{
			Duration: ptypes.DurationProto(*dur),
		}
	}
	twidth, theight := thumb.Dimensions()
	p.Thumbnail = append(p.Thumbnail, &schema.Pic_File{
		Index:         nextThumbnailIndex(p.Thumbnail),
		Size:          thumbfi.Size(),
		Mime:          imfmime,
		Width:         int64(twidth),
		Height:        int64(theight),
		AnimationInfo: imfanim,
	})

	if sts := mergePic(j, p, now, pfs, userID, ext); sts != nil {
		return sts
	}

	if err := t.MkdirAll(schema.PicBaseDir(t.PixPath, p.PicId), 0770); err != nil {
		return status.Internal(err, "Can't prepare pic dir")
	}

	newpath, sts := schema.PicFilePath(t.PixPath, p.PicId, p.File.Mime)
	if sts != nil {
		return sts
	}

	destroyTempFile = false
	if err := f.Close(); err != nil {
		sts := status.Internalf(err, "Can't close %v", f.Name())
		if err2 := t.Remove(f.Name()); err2 != nil {
			sts = status.WithSuppressed(sts, status.Internal(err2, "can't remove"))
		}
		return sts
	}
	if err := t.Rename(f.Name(), newpath); err != nil {
		sts := status.Internalf(err, "Can't rename %v to %v", f.Name(), newpath)
		if err2 := t.Remove(f.Name()); err2 != nil {
			sts = status.WithSuppressed(sts, status.Internal(err2, "can't remove"))
		}
		return sts
	}
	if err := ft.Close(); err != nil {
		return status.Internalf(err, "Can't close %v", ft.Name())
	}
	destroyNewFile := true
	defer func() {
		if destroyNewFile {
			if err := t.Remove(newpath); err != nil {
				status.ReplaceOrSuppress(&stscap, status.Internal(err, "can't remove", newpath))
			}
		}
	}()

	lastthumbnail := p.Thumbnail[len(p.Thumbnail)-1]
	// TODO: by luck the format created by imaging and the mime type decided by thumbnail are the
	// same.  Thumbnails should be made into proper rows with their own mime type.
	newthumbpath, sts := schema.PicFileThumbnailPath(
		t.PixPath, p.PicId, lastthumbnail.Index, lastthumbnail.Mime)
	if sts != nil {
		return sts
	}
	if err := t.Rename(ft.Name(), newthumbpath); err != nil {
		sts := status.Internalf(err, "Can't rename %v to %v", ft.Name(), newthumbpath)
		if err2 := t.Remove(ft.Name()); err2 != nil {
			sts = status.WithSuppressed(sts, status.Internal(err2, "can't remove temp thumbnail"))
		}
		if err2 := t.Remove(newpath); err2 != nil {
			sts = status.WithSuppressed(sts, status.Internal(err2, "can't remove pic", newpath))
		}
		return sts
	}

	t.CreatedPic = p
	destroyNewFile = false

	return nil
}

func confUrlLen(conf *schema.Configuration) (int64, int64) {
	var minUrlLen, maxUrlLen int64
	if conf.MinUrlLength != nil {
		minUrlLen = conf.MinUrlLength.Value
	} else {
		minUrlLen = math.MinInt64
	}
	if conf.MaxUrlLength != nil {
		maxUrlLen = conf.MaxUrlLength.Value
	} else {
		maxUrlLen = math.MaxInt64
	}
	return minUrlLen, maxUrlLen
}

func confFileNameLen(conf *schema.Configuration) (int64, int64) {
	var minFileNameLen, maxFileNameLen int64
	if conf.MinFileNameLength != nil {
		minFileNameLen = conf.MinFileNameLength.Value
	} else {
		minFileNameLen = math.MinInt64
	}
	if conf.MaxFileNameLength != nil {
		maxFileNameLen = conf.MaxFileNameLength.Value
	} else {
		maxFileNameLen = math.MaxInt64
	}
	return minFileNameLen, maxFileNameLen
}

// TODO: test
func nextThumbnailIndex(pfs []*schema.Pic_File) int64 {
	used := make(map[int64]bool)
	for _, pf := range pfs {
		if used[pf.Index] {
			panic("Index already used")
		}
		used[pf.Index] = true
	}
	for i := int64(0); ; i++ {
		if !used[i] {
			return i
		}
	}
}

// TODO: test
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

func mergePic(j *tab.Job, p *schema.Pic, now time.Time, pfs *schema.Pic_FileSource, userID int64,
	ext map[string]*any.Any) status.S {
	tim := schema.ToTspb(now)
	p.ModifiedTs = tim
	if ds := p.DeletionStatus; ds != nil {
		if ds.Temporary {
			// If the pic was soft deleted, it stays deleted, unless it was temporary.
			p.DeletionStatus = nil
		}
	}

	pfsExists := false
	for _, s := range p.Source {
		// Ignore pfs.Name and pfs.Referrer as those aren't sources.
		if s.Url == pfs.Url {
			pfsExists = true
			break
		}
		// At most one (non-anonymous) user can be in a source.
		// ignore sources from the same user after the first one
		if s.UserId != schema.AnonymousUserID && s.UserId == pfs.UserId {
			pfsExists = true
			break
		}
	}
	if !pfsExists {
		// Only accept the source if new information is being added, or there isn't any already.
		p.Source = append(p.Source, pfs)

		// Also, only give notification if they added something new.
		if userID != schema.AnonymousUserID {
			createdTs := schema.UserEventCreatedTsCol(tim)
			idx, sts := nextUserEventIndex(j, userID, createdTs)
			if sts != nil {
				return sts
			}
			oue := &schema.UserEvent{
				UserId:     userID,
				Index:      idx,
				CreatedTs:  tim,
				ModifiedTs: tim,
				Evt: &schema.UserEvent_UpsertPic_{
					UpsertPic: &schema.UserEvent_UpsertPic{
						PicId: p.PicId,
					},
				},
			}
			if err := j.InsertUserEvent(oue); err != nil {
				return status.Internal(err, "can't create user event")
			}
		}
	}
	if len(ext) != 0 && len(p.Ext) == 0 {
		p.Ext = make(map[string]*any.Any)
	}
	for k, v := range ext {
		if _, present := p.Ext[k]; present {
			return status.InvalidArgumentf(nil, "duplicate key %v in extension map", k)
		}
		p.Ext[k] = v
	}

	if err := j.UpdatePic(p); err != nil {
		return status.Internal(err, "can't update pic")
	}

	return nil
}

func findExistingPic(j *tab.Job, typ schema.PicIdent_Type, hash []byte) (*schema.Pic, status.S) {
	pis, err := j.FindPicIdents(db.Opts{
		Prefix: tab.PicIdentsIdent{
			Type:  &typ,
			Value: &hash,
		},
		Lock:  db.LockWrite,
		Limit: 1,
	})
	if err != nil {
		return nil, status.Internal(err, "can't find pic idents")
	}
	if len(pis) == 0 {
		return nil, nil
	}
	pics, err := j.FindPics(db.Opts{
		Prefix: tab.PicsPrimary{&pis[0].PicId},
		Lock:   db.LockWrite,
		Limit:  1,
	})
	if err != nil {
		return nil, status.Internal(err, "can't find pics")
	}
	if len(pics) != 1 {
		return nil, status.Internal(nil, "can't lookup pic")
	}

	return pics[0], nil
}

func insertPicHashes(j *tab.Job, picID int64, md5Hash, sha1Hash, sha512_256Hash []byte) status.S {
	md5Ident := &schema.PicIdent{
		PicId: picID,
		Type:  schema.PicIdent_MD5,
		Value: md5Hash,
	}
	if err := j.InsertPicIdent(md5Ident); err != nil {
		return status.Internal(err, "can't create md5")
	}
	sha1Ident := &schema.PicIdent{
		PicId: picID,
		Type:  schema.PicIdent_SHA1,
		Value: sha1Hash,
	}
	if err := j.InsertPicIdent(sha1Ident); err != nil {
		return status.Internal(err, "can't create sha1")
	}
	sha512_256Ident := &schema.PicIdent{
		PicId: picID,
		Type:  schema.PicIdent_SHA512_256,
		Value: sha512_256Hash,
	}
	if err := j.InsertPicIdent(sha512_256Ident); err != nil {
		return status.Internal(err, "can't create sha512_256")
	}
	return nil
}

func insertPerceptualHash(j *tab.Job, picID int64, im imaging.PixurImage) status.S {
	hash, inputs, sts := im.PerceptualHash0()
	if sts != nil {
		return sts
	}
	dct0Ident := &schema.PicIdent{
		PicId:      picID,
		Type:       schema.PicIdent_DCT_0,
		Value:      hash,
		Dct0Values: inputs,
	}
	if err := j.InsertPicIdent(dct0Ident); err != nil {
		return status.Internal(err, "can't create dct0")
	}
	return nil
}

// TODO: test
func (t *UpsertPicTask) prepareLocalFile(ctx context.Context, r io.ReadSeeker) (
	_ *os.File, _ func(*status.S), _ int64, stscap status.S) {
	off, err := r.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, nil, 0, status.Internal(err, "can't seek")
	}
	defer func() {
		if _, err := r.Seek(off, io.SeekStart); err != nil {
			status.ReplaceOrSuppress(&stscap, status.Internal(err, "can't seek"))
		}
	}()

	var size int64
	f, cleanup, sts := t.prepareFile(func(w io.Writer) status.S {
		n, err := io.Copy(w, r)
		if err != nil {
			return status.Internal(err, "can't copy file")
		}
		size = n
		return nil
	})
	if sts != nil {
		return nil, nil, 0, sts
	}
	return f, cleanup, size, nil
}

// TODO: test
func (t *UpsertPicTask) prepareFile(move func(io.Writer) status.S) (
	_ *os.File, _ func(*status.S), stscap status.S) {
	f, cleanup, sts := t.tempFile()
	if sts != nil {
		return nil, nil, sts
	}
	destroy := true
	defer func() {
		if destroy {
			cleanup(&stscap)
		}
	}()
	if sts := move(f); sts != nil {
		return nil, nil, sts
	}
	// The file is now local.  Sync it, since external programs might read it.
	if err := f.Sync(); err != nil {
		return nil, nil, status.Internal(err, "can't sync file")
	}
	destroy = false
	return f, cleanup, nil
}

// TODO: test
func (t *UpsertPicTask) tempFile() (*os.File, func(*status.S), status.S) {
	f, err := t.TempFile(t.PixPath, "__")
	if err != nil {
		return nil, nil, status.Internal(err, "can't create tempfile")
	}
	return f, func(stscap *status.S) {
		if err := f.Close(); err != nil {
			status.ReplaceOrSuppress(stscap, status.Internal(err, "can't close tempfile", f.Name()))
		}
		if err := t.Remove(f.Name()); err != nil {
			status.ReplaceOrSuppress(stscap, status.Internal(err, "can't remove tempfile", f.Name()))
		}
	}, nil
}

// TODO: add tests
func validateAndNormalizeURL(rawurl string, minUrlLen, maxUrlLen int64) (*url.URL, status.S) {
	normrawurl, err := text.DefaultValidateNoNewlineAndNormalize(rawurl, "url", minUrlLen, maxUrlLen)
	if err != nil {
		return nil, status.From(err)
	}
	loc, err := url.Parse(normrawurl)
	if err != nil {
		return nil, status.InvalidArgument(err, "can't parse", normrawurl)
	}
	if loc.Scheme != "http" && loc.Scheme != "https" {
		return nil, status.InvalidArgument(nil, "can't use non HTTP")
	}
	if loc.User != nil {
		return nil, status.InvalidArgument(nil, "can't provide userinfo")
	}

	return loc, nil
}

func (t *UpsertPicTask) prepareRemoteFile(ctx context.Context, loc, ref *url.URL) (
	_ *os.File, _ func(*status.S), _ int64, _ string, stscap status.S) {
	if loc == nil {
		return nil, nil, 0, "", status.InvalidArgument(nil, "missing URL")
	}

	req, err := http.NewRequest(http.MethodGet, loc.String(), nil)
	if err != nil {
		// if this fails, it's probably our fault
		return nil, nil, 0, "", status.Internal(err, "can't create request")
	}
	if ref != nil {
		ref2 := *ref
		ref2.Fragment = ""
		req.Header.Add("Referer", ref2.String())
	}
	req = req.WithContext(ctx)
	resp, err := t.HTTPClient.Do(req)
	if err != nil {
		return nil, nil, 0, "", status.InvalidArgument(err, "can't download", loc)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			status.ReplaceOrSuppress(&stscap, status.Internal(err, "can't close url req", loc))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		// TODO: log the response and headers
		return nil, nil, 0, "",
			status.InvalidArgumentf(nil, "can't download %s [%d]", loc, resp.StatusCode)
	}

	// TODO: log error
	rawname, _ := parseContentDispositionName(resp.Header.Get("Content-Disposition"))

	var size int64
	f, cleanup, sts := t.prepareFile(func(w io.Writer) status.S {
		n, err := io.Copy(w, resp.Body)
		if err != nil {
			// This could either be because the remote hung up or a file error on our side.  Assume that
			// our system is okay, making this an InvalidArgument
			return status.InvalidArgument(err, "can't copy file", loc)
		}
		size = n
		return nil
	})
	if sts != nil {
		return nil, nil, 0, "", sts
	}
	return f, cleanup, size, rawname, nil
}

func parseUrlName(loc *url.URL) (string, status.S) {
	if len(loc.Path) > 0 && loc.Path[len(loc.Path)-1] == '/' {
		return "", nil
	}
	// Can happen for a url that is a dir like http://foo.com
	if base := path.Base(loc.Path); base != "." {
		return base, nil
	}

	return "", nil
}

// TODO: implement
func parseContentDispositionName(value string) (string, status.S) {
	return "", nil
}

func generatePicHashes(f io.Reader, fns ...func() hash.Hash) ([][]byte, status.S) {
	hs := make([]hash.Hash, len(fns))
	for i, fn := range fns {
		hs[i] = fn()
	}
	ws := make([]io.Writer, len(hs))
	for i, h := range hs {
		ws[i] = h // Go lacks contravariance, so we have to do this.
	}
	if _, err := io.Copy(io.MultiWriter(ws...), f); err != nil {
		return nil, status.Internal(err, "Can't copy")
	}
	sums := make([][]byte, len(hs))
	for i, h := range hs {
		sums[i] = h.Sum(nil)
	}
	return sums, nil
}
