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

	imaging "pixur.org/pixur/image"
	"pixur.org/pixur/schema"
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
	Filename string
	Filesize int64
}

func (t *UpsertPicTask) Run() error {
	tx, err := t.DB.Begin()
	if err != nil {
		return s.InternalError(err, "Can't begin TX")
	}
	defer tx.Rollback()

	if err := t.runInternal(tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return s.InternalError(err, "Can't commit")
	}
	return nil
}

func (t *UpsertPicTask) runInternal(tx *sql.Tx) error {
	if t.File == nil && t.FileURL == "" {
		return s.InvalidArgument(nil, "No pic specified")
	}
	now := t.Now()
	if len(t.Md5Hash) != 0 {
		if p, err := findExistingPic(tx, schema.PicIdentifier_MD5, t.Md5Hash); err != nil {
			return err
		} else if p != nil {
			return t.merge(tx, p, now, t.Header, t.FileURL, t.TagNames)
		}
	}

	f, err := t.TempFile(t.PixPath, "__")
	if err != nil {
		return s.InternalError(err, "Can't create tempfile")
	}
	// on success, the name of f will change and it won't be removed.
	defer os.Remove(f.Name())
	defer f.Close()

	var fh FileHeader
	if t.File == nil {
		if header, err := t.downloadFile(f, t.FileURL); err != nil {
			return err
		} else {
			fh = *header
		}
	} else {
		// Make sure to copy the file to pixPath, to make sure it's on the right partition.
		// Also get a copy of the size,
		if n, err := io.Copy(f, t.File); err != nil {
			return s.InternalError(err, "Can't save file")
		} else {
			fh = FileHeader{
				Filename: t.Header.Filename,
				Filesize: n,
			}
		}
	}

	// The file is now local.  Sync it, since external programs might read it.
	if err := f.Sync(); err != nil {
		return s.InternalError(err, "Can't sync file")
	}

	// Avoids the need to have to f.Seek everywhere.
	hashReader := io.NewSectionReader(f, 0, fh.Filesize)
	md5Hash, sha1Hash, sha256Hash, err := generatePicHashes(hashReader)
	if len(t.Md5Hash) != 0 && !bytes.Equal(t.Md5Hash, md5Hash) {
		return s.InvalidArgumentf(nil, "Md5 hash mismatch %x != %x", t.Md5Hash, md5Hash)
	}
	// Still double check that the sha1 hash is not in use, even if the md5 one was
	// checked up at the beginning of the function.
	if p, err := findExistingPic(tx, schema.PicIdentifier_SHA1, sha1Hash); err != nil {
		return err
	} else if p != nil {
		return t.merge(tx, p, now, fh, t.FileURL, t.TagNames)
	}

	var p = new(schema.Pic)
	if err := p.Insert(tx); err != nil {
		return s.InternalError(err, "Can't insert")
	}
	if err := insertPicHashes(tx, p.PicId, md5Hash, sha1Hash, sha256Hash); err != nil {
		return err
	}

	p.SetCreatedTime(now)
	// Modified time is set in merge
	p.FileSize = fh.Filesize
	// TODO: copy in the file name

	// TODO: review this block of code
	img, err := imaging.FillImageConfig(f, p)
	if err != nil {
		if err, ok := err.(*imaging.BadWebmFormatErr); ok {
			return s.InvalidArgument(err, "Bad Web Fmt")
		}
		return s.InvalidArgument(err, "Can't decode image")
	}
	if err := insertPerceptualHash(tx, p.PicId, img); err != nil {
		return err
	}

	thumbnail := imaging.MakeThumbnail(img)
	if err := imaging.SaveThumbnail(thumbnail, p, t.PixPath); err != nil {
		return s.InternalError(err, "Can't save thumbnail")
	}

	if err := t.merge(tx, p, now, fh, t.FileURL, t.TagNames); err != nil {
		return err
	}

	if err := t.MkdirAll(filepath.Dir(p.Path(t.PixPath)), 0770); err != nil {
		return s.InternalError(err, "Can't prepare pic dir")
	}
	if err := t.Rename(f.Name(), p.Path(t.PixPath)); err != nil {
		return s.InternalErrorf(err, "Can't rename %v to %v", f.Name(), p.Path(t.PixPath))
	}

	return nil
}

func (t *UpsertPicTask) merge(tx *sql.Tx, p *schema.Pic, now time.Time, fh FileHeader,
	fileURL string, tagNames []string) error {
	p.SetModifiedTime(now)
	// TODO: store file name
	// TODO: store file URL
	// TODO: handle tag merger

	if err := p.Update(tx); err != nil {
		return s.InternalError(err, "Can't update pic")
	}

	return nil
}

func findExistingPic(tx *sql.Tx, typ schema.PicIdentifier_Type, hash []byte) (*schema.Pic, error) {
	identStmt, err := schema.PicIdentifierPrepare(
		"SELECT * FROM_ WHERE %s = ? AND %s = ? FOR UPDATE;",
		tx, schema.PicIdentColType, schema.PicIdentColValue)
	if err != nil {
		return nil, s.InternalError(err, "Can't prepare identStmt")
	}
	defer identStmt.Close()
	idents, err := schema.FindPicIdentifiers(identStmt, typ, hash)
	if err != nil {
		return nil, s.InternalError(err, "Can't find idents")
	}
	switch len(idents) {
	case 0:
		return nil, nil
	case 1:
		return lookupPicForUpdate(idents[0].PicId, tx)
	default:
		return nil, s.InternalErrorf(nil, "Found duplicate idents %+v", idents)
	}
}

func insertPicHashes(tx *sql.Tx, picID int64, md5Hash, sha1Hash, sha256Hash []byte) error {
	md5Ident := &schema.PicIdentifier{
		PicId: picID,
		Type:  schema.PicIdentifier_MD5,
		Value: md5Hash,
	}
	if err := md5Ident.Insert(tx); err != nil {
		return s.InternalError(err, "Can't insert md5")
	}
	sha1Ident := &schema.PicIdentifier{
		PicId: picID,
		Type:  schema.PicIdentifier_SHA1,
		Value: sha1Hash,
	}
	if err := sha1Ident.Insert(tx); err != nil {
		return s.InternalError(err, "Can't insert sha1")
	}
	sha256Ident := &schema.PicIdentifier{
		PicId: picID,
		Type:  schema.PicIdentifier_SHA256,
		Value: sha256Hash,
	}
	if err := sha256Ident.Insert(tx); err != nil {
		return s.InternalError(err, "Can't insert sha256")
	}
	return nil
}

func insertPerceptualHash(tx *sql.Tx, picID int64, im image.Image) error {
	hash, inputs := imaging.PerceptualHash0(im)
	dct0Ident := &schema.PicIdentifier{
		PicId:      picID,
		Type:       schema.PicIdentifier_DCT_0,
		Value:      hash,
		Dct0Values: inputs,
	}
	if err := dct0Ident.Insert(tx); err != nil {
		return s.InternalError(err, "Can't insert dct0")
	}
	return nil
}

func (t *UpsertPicTask) downloadFile(f *os.File, rawurl string) (*FileHeader, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, s.InvalidArgument(err, "Can't parse ", rawurl)
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
		Filesize: bytesRead,
	}
	// Can happen for a url that is a dir like http://foo.com/
	if base := path.Base(u.Path); base != "." {
		header.Filename = base
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
