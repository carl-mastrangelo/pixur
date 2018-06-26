package tasks

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"image"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"

	"pixur.org/pixur/be/imaging"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

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

	Header   FileHeader
	TagNames []string

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
		os.Remove(t.CreatedPic.Path(t.PixPath))
		os.Remove(t.CreatedPic.ThumbnailPath(t.PixPath))
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

	if t.File == nil && t.FileURL == "" {
		return status.InvalidArgument(nil, "No pic specified")
	}
	now := t.Now()
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
				return mergePic(j, p, now, t.Header, t.FileURL, t.TagNames, u.UserId)
			}
		}
	}

	f, fh, sts := t.prepareFile(ctx, t.File, t.Header, t.FileURL)
	if sts != nil {
		return sts
	}
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
	im, err := imaging.ReadImage(io.NewSectionReader(f, 0, fh.Size))
	if err != nil {
		return status.InvalidArgument(err, "Can't decode image")
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
			return mergePic(j, p, now, *fh, t.FileURL, t.TagNames, u.UserId)
		}
	} else {
		picID, err := j.AllocID()
		if err != nil {
			return status.InternalError(err, "can't allocate id")
		}
		p = &schema.Pic{
			PicId:         picID,
			FileSize:      fh.Size,
			Mime:          im.Mime,
			Width:         int64(im.Bounds().Dx()),
			Height:        int64(im.Bounds().Dy()),
			AnimationInfo: im.AnimationInfo,
			// ModifiedTime is set in mergePic
			// UserId is set in mergePic
		}
		p.SetCreatedTime(now)

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
	if err := imaging.OutputThumbnail(im, p.Mime, ft); err != nil {
		return status.InternalError(err, "Can't save thumbnail")
	}

	if err := mergePic(j, p, now, *fh, t.FileURL, t.TagNames, u.UserId); err != nil {
		return err
	}

	if err := t.MkdirAll(filepath.Dir(p.Path(t.PixPath)), 0770); err != nil {
		return status.InternalError(err, "Can't prepare pic dir")
	}
	if err := f.Close(); err != nil {
		return status.InternalErrorf(err, "Can't close %v", f.Name())
	}
	if err := t.Rename(f.Name(), p.Path(t.PixPath)); err != nil {
		return status.InternalErrorf(err, "Can't rename %v to %v", f.Name(), p.Path(t.PixPath))
	}
	if err := ft.Close(); err != nil {
		return status.InternalErrorf(err, "Can't close %v", ft.Name())
	}
	if err := t.Rename(ft.Name(), p.ThumbnailPath(t.PixPath)); err != nil {
		os.Remove(p.Path(t.PixPath))
		return status.InternalErrorf(err, "Can't rename %v to %v", ft.Name(), p.ThumbnailPath(t.PixPath))
	}

	t.CreatedPic = p

	return nil
}

func mergePic(j *tab.Job, p *schema.Pic, now time.Time, fh FileHeader, fileURL string,
	tagNames []string, userID int64) status.S {
	p.SetModifiedTime(now)
	if ds := p.GetDeletionStatus(); ds != nil {
		if ds.Temporary {
			// If the pic was soft deleted, it stays deleted, unless it was temporary.
			p.DeletionStatus = nil
		}
	}

	if err := upsertTags(j, tagNames, p.PicId, now, userID); err != nil {
		return err
	}

	if fileURL != "" {
		// If filedata was provided, still check that the url is valid.  Also strips fragment
		u, sts := validateURL(fileURL)
		if sts != nil {
			return sts
		}
		p.Source = append(p.Source, &schema.Pic_FileSource{
			Url:       u.String(),
			CreatedTs: schema.ToTspb(now),
		})
	}
	if fh.Name != "" {
		p.FileName = append(p.FileName, fh.Name)
	}
	// If this user is the first to create the pic, they get credit for uploading it.
	if p.UserId == schema.AnonymousUserID {
		p.UserId = userID
	}
	if err := j.UpdatePic(p); err != nil {
		return status.InternalError(err, "can't update pic")
	}

	return nil
}

func upsertTags(j *tab.Job, rawTags []string, picID int64, now time.Time, userID int64) status.S {
	newTagNames, err := cleanTagNames(rawTags)
	if err != nil {
		return err
	}

	attachedTags, _, err := findAttachedPicTags(j, picID)
	if err != nil {
		return err
	}

	unattachedTagNames := findUnattachedTagNames(attachedTags, newTagNames)
	existingTags, unknownNames, err := findExistingTagsByName(j, unattachedTagNames)
	if err != nil {
		return err
	}

	if sts := updateExistingTags(j, existingTags, now); err != nil {
		return sts
	}
	newTags, sts := createNewTags(j, unknownNames, now)
	if sts != nil {
		return sts
	}

	existingTags = append(existingTags, newTags...)
	if _, err := createPicTags(j, existingTags, picID, now, userID); err != nil {
		return err
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
			return nil, nil, status.InternalError(err, "can't lookup tag")
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
	tags []*schema.Tag, unknownNames []string, err status.S) {
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
		return nil, status.InternalError(err, "can't lookup pic")
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

func insertPerceptualHash(j *tab.Job, picID int64, im image.Image) status.S {
	hash, inputs := imaging.PerceptualHash0(im)
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
func (t *UpsertPicTask) prepareFile(ctx context.Context, fd multipart.File, fh FileHeader, u string) (
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
				Name: fh.Name,
				Size: n,
			}
		}
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

func (t *UpsertPicTask) downloadFile(ctx context.Context, f *os.File, rawurl string) (
	*FileHeader, status.S) {
	u, sts := validateURL(rawurl)
	if sts != nil {
		return nil, sts
	}

	// TODO: make sure this isn't reading from ourself
	req, err := http.NewRequest(http.MethodGet, rawurl, nil)
	if err != nil {
		// if this fails, it's probably our fault
		return nil, status.InternalError(err, "Can't create request")
	}
	req = req.WithContext(ctx)
	resp, err := t.HTTPClient.Do(req)
	if err != nil {
		return nil, status.InvalidArgument(err, "Can't download", rawurl)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// todo: log the response and headers
		return nil, status.InvalidArgumentf(nil, "Can't download %s [%d]", rawurl, resp.StatusCode)
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
