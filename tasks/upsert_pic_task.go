package tasks

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"database/sql"
	"image"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"

	"pixur.org/pixur/imaging"
	"pixur.org/pixur/schema"
	"pixur.org/pixur/schema/db"
	tab "pixur.org/pixur/schema/tables"
	s "pixur.org/pixur/status"
)

type UpsertPicTask struct {
	// Deps
	PixPath    string
	DB         *sql.DB
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

func (t *UpsertPicTask) Run() (errCap error) {
	j, err := tab.NewJob(t.DB)
	if err != nil {
		return s.InternalError(err, "can't create job")
	}
	defer cleanUp(j, errCap)

	if err := t.runInternal(j); err != nil {
		return err
	}

	// TODO: Check if this delete the original pic on a failed merge.
	if err := j.Commit(); err != nil {
		os.Remove(t.CreatedPic.Path(t.PixPath))
		os.Remove(t.CreatedPic.ThumbnailPath(t.PixPath))
		t.CreatedPic = nil
		return s.InternalError(err, "can't commit job")
	}
	return nil
}

func (t *UpsertPicTask) runInternal(j tab.Job) error {
	if t.File == nil && t.FileURL == "" {
		return s.InvalidArgument(nil, "No pic specified")
	}
	now := t.Now()
	if len(t.Md5Hash) != 0 {
		p, err := findExistingPic(j, schema.PicIdent_MD5, t.Md5Hash)
		if err != nil {
			return err
		}
		if p != nil {
			if p.HardDeleted() {
				if !p.GetDeletionStatus().Temporary {
					return s.InvalidArgument(nil, "Can't upload deleted pic.")
				}
				// Fallthrough.  We still need to download, and then remerge.
			} else {
				t.CreatedPic = p
				return mergePic(j, p, now, t.Header, t.FileURL, t.TagNames)
			}
		}
	}

	f, fh, err := t.prepareFile(t.File, t.Header, t.FileURL)
	if err != nil {
		return err
	}
	// on success, the name of f will change and it won't be removed.
	defer os.Remove(f.Name())
	defer f.Close()

	md5Hash, sha1Hash, sha256Hash, err := generatePicHashes(io.NewSectionReader(f, 0, fh.Size))
	if len(t.Md5Hash) != 0 && !bytes.Equal(t.Md5Hash, md5Hash) {
		return s.InvalidArgumentf(nil, "Md5 hash mismatch %x != %x", t.Md5Hash, md5Hash)
	}
	im, err := imaging.ReadImage(io.NewSectionReader(f, 0, fh.Size))
	if err != nil {
		return s.InvalidArgument(err, "Can't decode image")
	}

	// Still double check that the sha1 hash is not in use, even if the md5 one was
	// checked up at the beginning of the function.
	p, err := findExistingPic(j, schema.PicIdent_SHA1, sha1Hash)
	if err != nil {
		return err
	}
	if p != nil {
		if p.HardDeleted() {
			if !p.GetDeletionStatus().Temporary {
				return s.InvalidArgument(nil, "Can't upload deleted pic.")
			}
			//  fall through, picture needs to be undeleted.
		} else {
			t.CreatedPic = p
			return mergePic(j, p, now, *fh, t.FileURL, t.TagNames)
		}
	} else {
		picID, err := j.AllocID()
		if err != nil {
			return s.InternalError(err, "can't allocate id")
		}
		p = &schema.Pic{
			PicId:         picID,
			FileSize:      fh.Size,
			Mime:          im.Mime,
			Width:         int64(im.Bounds().Dx()),
			Height:        int64(im.Bounds().Dy()),
			AnimationInfo: im.AnimationInfo,
			CreatedTs:     schema.ToTs(now),
			// ModifiedTime is set in mergePic
		}

		if err := j.InsertPic(p); err != nil {
			return s.InternalError(err, "Can't insert")
		}
		if err := insertPicHashes(j, p.PicId, md5Hash, sha1Hash, sha256Hash); err != nil {
			return err
		}
		if err := insertPerceptualHash(j, p.PicId, im); err != nil {
			return err
		}
	}

	ft, err := t.TempFile(t.PixPath, "__")
	if err != nil {
		return s.InternalError(err, "Can't create tempfile")
	}
	defer os.Remove(ft.Name())
	defer ft.Close()
	if err := imaging.OutputThumbnail(im, p.Mime, ft); err != nil {
		return s.InternalError(err, "Can't save thumbnail")
	}

	if err := mergePic(j, p, now, *fh, t.FileURL, t.TagNames); err != nil {
		return err
	}

	if err := t.MkdirAll(filepath.Dir(p.Path(t.PixPath)), 0770); err != nil {
		return s.InternalError(err, "Can't prepare pic dir")
	}
	if err := f.Close(); err != nil {
		return s.InternalErrorf(err, "Can't close %v", f.Name())
	}
	if err := t.Rename(f.Name(), p.Path(t.PixPath)); err != nil {
		return s.InternalErrorf(err, "Can't rename %v to %v", f.Name(), p.Path(t.PixPath))
	}
	if err := ft.Close(); err != nil {
		return s.InternalErrorf(err, "Can't close %v", ft.Name())
	}
	if err := t.Rename(ft.Name(), p.ThumbnailPath(t.PixPath)); err != nil {
		os.Remove(p.Path(t.PixPath))
		return s.InternalErrorf(err, "Can't rename %v to %v", ft.Name(), p.ThumbnailPath(t.PixPath))
	}

	t.CreatedPic = p

	return nil
}

func mergePic(j tab.Job, p *schema.Pic, now time.Time, fh FileHeader, fileURL string,
	tagNames []string) error {
	p.SetModifiedTime(now)
	if ds := p.GetDeletionStatus(); ds != nil {
		if ds.Temporary {
			// If the pic was soft deleted, it stays deleted, unless it was temporary.
			p.DeletionStatus = nil
		}
	}

	if err := upsertTags(j, tagNames, p.PicId, now); err != nil {
		return err
	}

	if fileURL != "" {
		// If filedata was provided, still check that the url is valid.  Also strips fragment
		u, err := validateURL(fileURL)
		if err != nil {
			return err
		}
		p.Source = append(p.Source, &schema.Pic_FileSource{
			Url:       u.String(),
			CreatedTs: schema.ToTs(now),
		})
	}
	if fh.Name != "" {
		p.FileName = append(p.FileName, fh.Name)
	}

	if err := j.UpdatePic(p); err != nil {
		return s.InternalError(err, "can't update pic")
	}

	return nil
}

func upsertTags(j tab.Job, rawTags []string, picID int64, now time.Time) error {
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

	if err := updateExistingTags(j, existingTags, now); err != nil {
		return err
	}
	newTags, err := createNewTags(j, unknownNames, now)
	if err != nil {
		return err
	}

	existingTags = append(existingTags, newTags...)
	if _, err := createPicTags(j, existingTags, picID, now); err != nil {
		return err
	}

	return nil
}

func findAttachedPicTags(j tab.Job, picID int64) ([]*schema.Tag, []*schema.PicTag, error) {
	pts, err := j.FindPicTags(db.Opts{
		Prefix: tab.PicTagsPrimary{PicId: &picID},
		Lock:   db.LockWrite,
	})
	if err != nil {
		return nil, nil, s.InternalError(err, "cant't find pic tags")
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
			return nil, nil, s.InternalError(err, "can't find tags")
		}
		if len(ts) != 1 {
			return nil, nil, s.InternalError(err, "can't lookup tag")
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

func findExistingTagsByName(j tab.Job, names []string) (
	tags []*schema.Tag, unknownNames []string, err error) {
	for _, name := range names {
		ts, err := j.FindTags(db.Opts{
			Prefix: tab.TagsName{&name},
			Limit:  1,
			Lock:   db.LockWrite,
		})
		if err != nil {
			return nil, nil, s.InternalError(err, "can't find tags")
		}
		if len(ts) == 1 {
			tags = append(tags, ts[0])
		} else {
			unknownNames = append(unknownNames, name)
		}
	}

	return
}

func updateExistingTags(j tab.Job, tags []*schema.Tag, now time.Time) error {
	for _, tag := range tags {
		tag.SetModifiedTime(now)
		tag.UsageCount++
		if err := j.UpdateTag(tag); err != nil {
			return s.InternalError(err, "can't update tag")
		}
	}
	return nil
}

func createNewTags(j tab.Job, tagNames []string, now time.Time) ([]*schema.Tag, error) {
	var tags []*schema.Tag
	for _, name := range tagNames {
		tagID, err := j.AllocID()
		if err != nil {
			return nil, s.InternalError(err, "can't allocate id")
		}
		tag := &schema.Tag{
			TagId:      tagID,
			Name:       name,
			UsageCount: 1,
			ModifiedTs: schema.ToTs(now),
			CreatedTs:  schema.ToTs(now),
		}
		if err := j.InsertTag(tag); err != nil {
			return nil, s.InternalError(err, "can't create tag")
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func createPicTags(j tab.Job, tags []*schema.Tag, picID int64, now time.Time) ([]*schema.PicTag, error) {
	var picTags []*schema.PicTag
	for _, tag := range tags {
		pt := &schema.PicTag{
			PicId:      picID,
			TagId:      tag.TagId,
			Name:       tag.Name,
			ModifiedTs: schema.ToTs(now),
			CreatedTs:  schema.ToTs(now),
		}
		if err := j.InsertPicTag(pt); err != nil {
			return nil, s.InternalError(err, "can't create pic tag")
		}
		picTags = append(picTags, pt)
	}
	return picTags, nil
}

func findExistingPic(j tab.Job, typ schema.PicIdent_Type, hash []byte) (*schema.Pic, error) {
	pis, err := j.FindPicIdents(db.Opts{
		Prefix: tab.PicIdentsIdent{
			Type:  &typ,
			Value: &hash,
		},
		Lock:  db.LockWrite,
		Limit: 1,
	})
	if err != nil {
		return nil, s.InternalError(err, "can't find pic idents")
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
		return nil, s.InternalError(err, "can't find pics")
	}
	if len(pics) != 1 {
		return nil, s.InternalError(err, "can't lookup pic")
	}

	return pics[0], nil
}

func insertPicHashes(j tab.Job, picID int64, md5Hash, sha1Hash, sha256Hash []byte) error {
	md5Ident := &schema.PicIdent{
		PicId: picID,
		Type:  schema.PicIdent_MD5,
		Value: md5Hash,
	}
	if err := j.InsertPicIdent(md5Ident); err != nil {
		return s.InternalError(err, "can't create md5")
	}
	sha1Ident := &schema.PicIdent{
		PicId: picID,
		Type:  schema.PicIdent_SHA1,
		Value: sha1Hash,
	}
	if err := j.InsertPicIdent(sha1Ident); err != nil {
		return s.InternalError(err, "can't create sha1")
	}
	sha256Ident := &schema.PicIdent{
		PicId: picID,
		Type:  schema.PicIdent_SHA256,
		Value: sha256Hash,
	}
	if err := j.InsertPicIdent(sha256Ident); err != nil {
		return s.InternalError(err, "can't create sha256")
	}
	return nil
}

func insertPerceptualHash(j tab.Job, picID int64, im image.Image) error {
	hash, inputs := imaging.PerceptualHash0(im)
	dct0Ident := &schema.PicIdent{
		PicId:      picID,
		Type:       schema.PicIdent_DCT_0,
		Value:      hash,
		Dct0Values: inputs,
	}
	if err := j.InsertPicIdent(dct0Ident); err != nil {
		return s.InternalError(err, "can't create dct0")
	}
	return nil
}

// prepareFile prepares the file for image processing.
func (t *UpsertPicTask) prepareFile(fd multipart.File, fh FileHeader, u string) (_ *os.File, _ *FileHeader, errCap error) {
	f, err := t.TempFile(t.PixPath, "__")
	if err != nil {
		return nil, nil, s.InternalError(err, "Can't create tempfile")
	}
	defer func() {
		if errCap != nil {
			closeAndRemove(f)
		}
	}()

	var h *FileHeader
	if fd == nil {
		if header, err := t.downloadFile(f, u); err != nil {
			return nil, nil, err
		} else {
			h = header
		}
	} else {
		// TODO: maybe extract the filename from the url, if not provided in FileHeader
		// Make sure to copy the file to pixPath, to make sure it's on the right partition.
		// Also get a copy of the size.  We don't want to move the file if it is on the
		// same partition, because then we can't retry the task on failure.
		if n, err := io.Copy(f, fd); err != nil {
			return nil, nil, s.InternalError(err, "Can't save file")
		} else {
			h = &FileHeader{
				Name: fh.Name,
				Size: n,
			}
		}
	}

	// The file is now local.  Sync it, since external programs might read it.
	if err := f.Sync(); err != nil {
		return nil, nil, s.InternalError(err, "Can't sync file")
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
func validateURL(rawurl string) (*url.URL, error) {
	if len(rawurl) > 1024 {
		return nil, s.InvalidArgument(nil, "Can't use long URL")
	}
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, s.InvalidArgument(err, "Can't parse ", rawurl)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, s.InvalidArgument(nil, "Can't use non HTTP")
	}
	if u.User != nil {
		return nil, s.InvalidArgument(nil, "Can't provide userinfo")
	}
	u.Fragment = ""

	return u, nil
}

func (t *UpsertPicTask) downloadFile(f *os.File, rawurl string) (*FileHeader, error) {
	u, err := validateURL(rawurl)
	if err != nil {
		return nil, err
	}

	// TODO: make sure this isn't reading from ourself
	resp, err := t.HTTPClient.Get(rawurl)
	if err != nil {
		return nil, s.InvalidArgument(err, "Can't download ", rawurl)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, s.InvalidArgumentf(nil, "Can't download %s [%d]", rawurl, resp.StatusCode)
	}

	bytesRead, err := io.Copy(f, resp.Body)
	// This could either be because the remote hung up or a file error on our side.  Assume that
	// our system is okay, making this an InvalidArgument
	if err != nil {
		return nil, s.InvalidArgumentf(err, "Can't copy downloaded file")
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

func generatePicHashes(f io.Reader) (md5Hash, sha1Hash, sha256Hash []byte, err error) {
	h1 := md5.New()
	h2 := sha1.New()
	h3 := sha256.New()

	if _, err := io.Copy(io.MultiWriter(h1, h2, h3), f); err != nil {
		return nil, nil, nil, s.InternalError(err, "Can't copy")
	}
	return h1.Sum(nil), h2.Sum(nil), h3.Sum(nil), nil
}
