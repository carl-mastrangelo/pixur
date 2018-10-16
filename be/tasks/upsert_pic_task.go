package tasks

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"io"
	"mime/multipart"
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
)

// UpsertPicTask inserts or updates a pic with the provided information.
type UpsertPicTask struct {
	// Deps
	PixPath    string
	DB         db.DB
	HTTPClient *http.Client
	// os functions
	TempFile func(dir, prefix string) (*os.File, error)
	Rename   func(oldpath, newpath string) error
	MkdirAll func(path string, perm os.FileMode) error
	Now      func() time.Time

	// Inputs
	FileURL string
	File    multipart.File
	Md5Hash []byte

	// Header is the name (and size) of the file.  Currently only the Name is used.  If the name is
	// absent, UpsertPicTask will try to derive a name automatically from the FileURL.
	Header   FileHeader
	TagNames []string

	// Ext is additional extra data associated with this pic.  If a key is present in both the
	// new pic and the existing pic, Upsert will fail.
	Ext map[string]*any.Any

	// TODO: eventually take the Referer[sic].  This is to pass to HTTPClient when retrieving the
	// pic.

	// Results
	CreatedPic *schema.Pic
}

type FileHeader struct {
	Name string
	Size int64
}

func (t *UpsertPicTask) Run(ctx context.Context) (stsCap status.S) {
	j, err := tab.NewJob(ctx, t.DB)
	if err != nil {
		return status.InternalError(err, "can't create job")
	}
	defer cleanUp(j, &stsCap)

	if sts := t.runInternal(ctx, j); sts != nil {
		return sts
	}

	// TODO: Check if this delete the original pic on a failed merge.
	if err := j.Commit(); err != nil {
		// Don't delete old pics, as the commit may have actually succeeded.  A cron job will clean
		// this up.
		t.CreatedPic = nil
		return status.InternalError(err, "can't commit job")
	}
	return nil
}

func (t *UpsertPicTask) runInternal(ctx context.Context, j *tab.Job) status.S {
	u, sts := requireCapability(ctx, j, schema.User_PIC_CREATE)
	if sts != nil {
		return sts
	}

	var furl *url.URL
	if t.FileURL != "" {
		fu, sts := validateURL(t.FileURL)
		if sts != nil {
			return sts
		}
		furl = fu
	} else if t.File == nil {
		return status.InvalidArgument(nil, "No pic specified")
	}
	now := t.Now()
	// TODO: test this
	if len(t.Header.Name) > 1024 {
		return status.InvalidArgument(nil, "filename is too long")
	}

	pfs := &schema.Pic_FileSource{
		CreatedTs: schema.ToTspb(now),
		UserId:    u.UserId,
		Name:      t.Header.Name,
	}
	if furl != nil {
		pfs.Url = furl.String()
	}

	// TODO: change md5 hash to:
	// Check if md5 is present, and return ALREADY_EXISTS or PERMISSION_DENIED if deleted
	//   User is expected to call new method: add file_source / referrer, and add tags
	// If present, verify md5 sum after download

	if len(t.Md5Hash) != 0 {
		p, sts := findExistingPic(j, schema.PicIdent_MD5, t.Md5Hash)
		if sts != nil {
			return sts
		}
		if p != nil {
			if p.HardDeleted() {
				if !p.GetDeletionStatus().Temporary {
					return status.InvalidArgument(nil, "Can't upload deleted pic.")
				}
				// Fallthrough.  We still need to download, and then remerge.
			} else {
				t.CreatedPic = p
				return mergePic(j, p, now, pfs, t.Ext, t.TagNames, u.UserId)
			}
		}
	}

	f, fh, sts := t.prepareFile(ctx, t.File, t.Header, furl)
	if sts != nil {
		return sts
	}
	// after preparing the f, fh, is authoritative.
	pfs.Name = fh.Name
	// on success, the name of f will change and it won't be removed.
	defer os.Remove(f.Name())
	defer f.Close()

	md5Hash, sha1Hash, sha256Hash, sts := generatePicHashes(io.NewSectionReader(f, 0, fh.Size))
	if sts != nil {
		// TODO: test this case
		return sts
	}
	if len(t.Md5Hash) != 0 && !bytes.Equal(t.Md5Hash, md5Hash) {
		return status.InvalidArgumentf(nil, "Md5 hash mismatch %x != %x", t.Md5Hash, md5Hash)
	}
	im, sts := imaging.ReadImage(io.NewSectionReader(f, 0, fh.Size))
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
			return mergePic(j, p, now, pfs, t.Ext, t.TagNames, u.UserId)
		}
	} else {
		picID, err := j.AllocID()
		if err != nil {
			return status.InternalError(err, "can't allocate id")
		}

		width, height := im.Dimensions()
		p = &schema.Pic{
			PicId:         picID,
			FileSize:      fh.Size,
			Mime:          schema.Pic_Mime(immime),
			Width:         int64(width),
			Height:        int64(height),
			AnimationInfo: imanim,
			File: &schema.Pic_File{
				Index:         0, // always 0 for main pic
				Size:          fh.Size,
				Mime:          schema.Pic_File_Mime(immime),
				Width:         int64(width),
				Height:        int64(height),
				AnimationInfo: imanim,
			},
			// ModifiedTime is set in mergePic
		}
		p.SetCreatedTime(now)
		p.File.CreatedTs = p.CreatedTs
		p.File.ModifiedTs = p.File.CreatedTs

		if err := j.InsertPic(p); err != nil {
			return status.InternalError(err, "Can't insert")
		}
		if sts := insertPicHashes(j, p.PicId, md5Hash, sha1Hash, sha256Hash); sts != nil {
			return sts
		}
		if sts := insertPerceptualHash(j, p.PicId, im); sts != nil {
			return sts
		}
	}

	ft, err := t.TempFile(t.PixPath, "__")
	if err != nil {
		return status.InternalError(err, "Can't create tempfile")
	}
	defer os.Remove(ft.Name())
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
		return status.InternalError(err, "unable to stat thumbnail")
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

	if sts := mergePic(j, p, now, pfs, t.Ext, t.TagNames, u.UserId); sts != nil {
		return sts
	}

	if err := t.MkdirAll(schema.PicBaseDir(t.PixPath, p.PicId), 0770); err != nil {
		return status.InternalError(err, "Can't prepare pic dir")
	}
	if err := f.Close(); err != nil {
		return status.InternalErrorf(err, "Can't close %v", f.Name())
	}
	newpath, sts := schema.PicFilePath(t.PixPath, p.PicId, p.File.Mime)
	if sts != nil {
		return sts
	}
	if err := t.Rename(f.Name(), newpath); err != nil {
		return status.InternalErrorf(err, "Can't rename %v to %v", f.Name(), newpath)
	}
	if err := ft.Close(); err != nil {
		return status.InternalErrorf(err, "Can't close %v", ft.Name())
	}

	lastthumbnail := p.Thumbnail[len(p.Thumbnail)-1]
	// TODO: by luck the format created by imaging and the mime type decided by thumbnail are the
	// same.  Thumbnails should be made into proper rows with their own mime type.
	newthumbpath, sts := schema.PicFileThumbnailPath(
		t.PixPath, p.PicId, lastthumbnail.Index, lastthumbnail.Mime)
	if sts != nil {
		return sts
	}
	if err := t.Rename(ft.Name(), newthumbpath); err != nil {
		os.Remove(newpath)
		return status.InternalErrorf(err, "Can't rename %v to %v", ft.Name(), newthumbpath)
	}

	t.CreatedPic = p

	return nil
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

func mergePic(j *tab.Job, p *schema.Pic, now time.Time, pfs *schema.Pic_FileSource,
	ext map[string]*any.Any, tagNames []string, userID int64) status.S {
	p.SetModifiedTime(now)
	if ds := p.GetDeletionStatus(); ds != nil {
		if ds.Temporary {
			// If the pic was soft deleted, it stays deleted, unless it was temporary.
			p.DeletionStatus = nil
		}
	}

	if sts := upsertTags(j, tagNames, p.PicId, now, userID); sts != nil {
		return sts
	}

	// ignore sources from the same user after the first one
	userFirstSource := true
	if userID != schema.AnonymousUserID {
		for _, s := range p.Source {
			if s.UserId == userID {
				userFirstSource = false
				break
			}
		}
	}
	if userFirstSource {
		// Okay, it's their first time uploading, let's consider adding it.
		if pfs.Url != "" || len(p.Source) == 0 {
			// Only accept the source if new information is being added, or there isn't any already.
			// Ignore pfs.Name and pfs.Referrer as those aren't sources.
			p.Source = append(p.Source, pfs)
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
		return status.InternalError(err, "can't update pic")
	}

	return nil
}

func upsertTags(j *tab.Job, rawTags []string, picID int64, now time.Time, userID int64) status.S {
	newTagNames, sts := cleanTagNames(rawTags)
	if sts != nil {
		return sts
	}

	attachedTags, _, sts := findAttachedPicTags(j, picID)
	if sts != nil {
		return sts
	}

	unattachedTagNames := findUnattachedTagNames(attachedTags, newTagNames)
	existingTags, unknownNames, sts := findExistingTagsByName(j, unattachedTagNames)
	if sts != nil {
		return sts
	}

	if sts := updateExistingTags(j, existingTags, now); sts != nil {
		return sts
	}
	newTags, sts := createNewTags(j, unknownNames, now)
	if sts != nil {
		return sts
	}

	existingTags = append(existingTags, newTags...)
	if _, sts := createPicTags(j, existingTags, picID, now, userID); sts != nil {
		return sts
	}

	return nil
}

func findAttachedPicTags(j *tab.Job, picID int64) ([]*schema.Tag, []*schema.PicTag, status.S) {
	pts, err := j.FindPicTags(db.Opts{
		Prefix: tab.PicTagsPrimary{PicId: &picID},
		Lock:   db.LockWrite,
	})
	if err != nil {
		return nil, nil, status.InternalError(err, "cant't find pic tags")
	}

	var tags []*schema.Tag
	// TODO: maybe do something with lock ordering?
	for _, pt := range pts {
		ts, err := j.FindTags(db.Opts{
			Prefix: tab.TagsPrimary{&pt.TagId},
			Limit:  1,
			Lock:   db.LockWrite,
		})
		if err != nil {
			return nil, nil, status.InternalError(err, "can't find tags")
		}
		if len(ts) != 1 {
			return nil, nil, status.InternalError(nil, "can't lookup tag", len(ts))
		}
		tags = append(tags, ts[0])
	}
	return tags, pts, nil
}

// findUnattachedTagNames finds tag names that are not part of a pic's tags.
// While pic tags are the SoT for attachment, only the Tag is the SoT for the name.
func findUnattachedTagNames(attachedTags []*schema.Tag, newTagNames []string) []string {
	attachedTagNames := make(map[string]struct{}, len(attachedTags))

	for _, tag := range attachedTags {
		attachedTagNames[tag.Name] = struct{}{}
	}
	var unattachedTagNames []string
	for _, newTagName := range newTagNames {
		if _, attached := attachedTagNames[newTagName]; !attached {
			unattachedTagNames = append(unattachedTagNames, newTagName)
		}
	}

	return unattachedTagNames
}

func findExistingTagsByName(j *tab.Job, names []string) (
	tags []*schema.Tag, unknownNames []string, _ status.S) {
	for _, name := range names {
		ts, err := j.FindTags(db.Opts{
			Prefix: tab.TagsName{&name},
			Limit:  1,
			Lock:   db.LockWrite,
		})
		if err != nil {
			return nil, nil, status.InternalError(err, "can't find tags")
		}
		if len(ts) == 1 {
			tags = append(tags, ts[0])
		} else {
			unknownNames = append(unknownNames, name)
		}
	}

	return
}

func updateExistingTags(j *tab.Job, tags []*schema.Tag, now time.Time) status.S {
	for _, tag := range tags {
		tag.SetModifiedTime(now)
		tag.UsageCount++
		if err := j.UpdateTag(tag); err != nil {
			return status.InternalError(err, "can't update tag")
		}
	}
	return nil
}

func createNewTags(j *tab.Job, tagNames []string, now time.Time) ([]*schema.Tag, status.S) {
	var tags []*schema.Tag
	for _, name := range tagNames {
		tagID, err := j.AllocID()
		if err != nil {
			return nil, status.InternalError(err, "can't allocate id")
		}
		tag := &schema.Tag{
			TagId:      tagID,
			Name:       name,
			UsageCount: 1,
		}
		tag.SetCreatedTime(now)
		tag.SetModifiedTime(now)
		if err := j.InsertTag(tag); err != nil {
			return nil, status.InternalError(err, "can't create tag")
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func createPicTags(j *tab.Job, tags []*schema.Tag, picID int64, now time.Time, userID int64) (
	[]*schema.PicTag, status.S) {
	var picTags []*schema.PicTag
	for _, tag := range tags {
		pt := &schema.PicTag{
			PicId:  picID,
			TagId:  tag.TagId,
			Name:   tag.Name,
			UserId: userID,
		}
		pt.SetCreatedTime(now)
		pt.SetModifiedTime(now)
		if err := j.InsertPicTag(pt); err != nil {
			return nil, status.InternalError(err, "can't create pic tag")
		}
		picTags = append(picTags, pt)
	}
	return picTags, nil
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
		return nil, status.InternalError(err, "can't find pic idents")
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
		return nil, status.InternalError(err, "can't find pics")
	}
	if len(pics) != 1 {
		return nil, status.InternalError(nil, "can't lookup pic")
	}

	return pics[0], nil
}

func insertPicHashes(j *tab.Job, picID int64, md5Hash, sha1Hash, sha256Hash []byte) status.S {
	md5Ident := &schema.PicIdent{
		PicId: picID,
		Type:  schema.PicIdent_MD5,
		Value: md5Hash,
	}
	if err := j.InsertPicIdent(md5Ident); err != nil {
		return status.InternalError(err, "can't create md5")
	}
	sha1Ident := &schema.PicIdent{
		PicId: picID,
		Type:  schema.PicIdent_SHA1,
		Value: sha1Hash,
	}
	if err := j.InsertPicIdent(sha1Ident); err != nil {
		return status.InternalError(err, "can't create sha1")
	}
	sha256Ident := &schema.PicIdent{
		PicId: picID,
		Type:  schema.PicIdent_SHA256,
		Value: sha256Hash,
	}
	if err := j.InsertPicIdent(sha256Ident); err != nil {
		return status.InternalError(err, "can't create sha256")
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
		return status.InternalError(err, "can't create dct0")
	}
	return nil
}

// prepareFile prepares the file for image processing.
func (t *UpsertPicTask) prepareFile(ctx context.Context, fd multipart.File, fh FileHeader, u *url.URL) (
	_ *os.File, _ *FileHeader, stsCap status.S) {
	f, err := t.TempFile(t.PixPath, "__")
	if err != nil {
		return nil, nil, status.InternalError(err, "Can't create tempfile")
	}
	defer func() {
		if stsCap != nil {
			closeAndRemove(f)
		}
	}()

	var h *FileHeader
	if fd == nil {
		if header, sts := t.downloadFile(ctx, f, u); sts != nil {
			return nil, nil, sts
		} else {
			h = header
		}
	} else {
		// TODO: maybe extract the filename from the url, if not provided in FileHeader
		// Make sure to copy the file to pixPath, to make sure it's on the right partition.
		// Also get a copy of the size.  We don't want to move the file if it is on the
		// same partition, because then we can't retry the task on failure.
		if n, err := io.Copy(f, fd); err != nil {
			return nil, nil, status.InternalError(err, "Can't save file")
		} else {
			h = &FileHeader{
				Size: n,
			}
		}
	}
	// Provided header name takes precedence
	if fh.Name != "" {
		h.Name = fh.Name
	}

	// The file is now local.  Sync it, since external programs might read it.
	if err := f.Sync(); err != nil {
		return nil, nil, status.InternalError(err, "Can't sync file")
	}

	return f, h, nil
}

// closeAndRemove cleans up in the event of an error.  Windows needs the file to
// be closed so it is important to do it in order.
func closeAndRemove(f *os.File) {
	f.Close()
	os.Remove(f.Name())
}

// TODO: add tests
func validateURL(rawurl string) (*url.URL, status.S) {
	if len(rawurl) > 1024 {
		return nil, status.InvalidArgument(nil, "Can't use long URL")
	}
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, status.InvalidArgument(err, "Can't parse", rawurl)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, status.InvalidArgument(nil, "Can't use non HTTP")
	}
	if u.User != nil {
		return nil, status.InvalidArgument(nil, "Can't provide userinfo")
	}
	u.Fragment = ""

	return u, nil
}

func (t *UpsertPicTask) downloadFile(ctx context.Context, f *os.File, u *url.URL) (
	*FileHeader, status.S) {
	if u == nil {
		return nil, status.InvalidArgument(nil, "Missing URL")
	}

	// TODO: make sure this isn't reading from ourself
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		// if this fails, it's probably our fault
		return nil, status.InternalError(err, "Can't create request")
	}
	req = req.WithContext(ctx)
	resp, err := t.HTTPClient.Do(req)
	if err != nil {
		return nil, status.InvalidArgument(err, "Can't download", u)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// todo: log the response and headers
		return nil, status.InvalidArgumentf(nil, "Can't download %s [%d]", u, resp.StatusCode)
	}

	bytesRead, err := io.Copy(f, resp.Body)
	// This could either be because the remote hung up or a file error on our side.  Assume that
	// our system is okay, making this an InvalidArgument
	if err != nil {
		return nil, status.InvalidArgumentf(err, "Can't copy downloaded file")
	}
	header := &FileHeader{
		Size: bytesRead,
	}
	// Can happen for a url that is a dir like http://foo.com/
	if base := path.Base(u.Path); base != "." {
		header.Name = base
	}
	// TODO: support Content-disposition
	return header, nil
}

func generatePicHashes(f io.Reader) (md5Hash, sha1Hash, sha256Hash []byte, sts status.S) {
	h1 := md5.New()
	h2 := sha1.New()
	h3 := sha256.New()

	if _, err := io.Copy(io.MultiWriter(h1, h2, h3), f); err != nil {
		return nil, nil, nil, status.InternalError(err, "Can't copy")
	}
	return h1.Sum(nil), h2.Sum(nil), h3.Sum(nil), nil
}
