package tasks

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"image"
	"image/gif"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"pixur.org/pixur/schema"
	ptest "pixur.org/pixur/testing"
)

type TestContainer struct {
	T       testing.TB
	db      *sql.DB
	tempdir string
}

func Container(t testing.TB) *TestContainer {
	return &TestContainer{
		T: t,
	}
}

func (c *TestContainer) DB() *sql.DB {
	if c.db == nil {
		db, err := ptest.GetDB()
		if err != nil {
			c.T.Fatal(err)
		}
		if err := schema.CreateTables(db); err != nil {
			c.T.Fatal(err)
		}
		c.db = db
	}
	return c.db
}

func (c *TestContainer) Close() {
	if c.db != nil {
		if err := c.db.Close(); err != nil {
			c.T.Fatal(err)
		}
		c.db = nil
	}
	if c.tempdir != "" {
		if err := os.RemoveAll(c.tempdir); err != nil {
			c.T.Fatal(err)
		}
		c.tempdir = ""
	}
}

func (c *TestContainer) Tx() *sql.Tx {
	tx, err := c.DB().Begin()
	if err != nil {
		c.T.Fatal(err)
	}
	return tx
}

func (c *TestContainer) TempDir() string {
	if c.tempdir == "" {
		path, err := ioutil.TempDir("", "pixurtest")
		if err != nil {
			c.T.Fatal(err)
		}
		c.tempdir = path
	}
	return c.tempdir
}

func (c *TestContainer) TempFile() *os.File {
	f, err := ioutil.TempFile(c.TempDir(), "__")
	if err != nil {
		c.T.Fatal(err)
	}
	return f
}

func (c *TestContainer) WrapPic(p *schema.Pic) *TestPic {
	return &TestPic{
		Pic: p,
		c:   c,
	}
}

func (c *TestContainer) CreatePic() *TestPic {
	now := time.Now()
	p := &schema.Pic{
		PicId:      c.ID(),
		CreatedTs:  schema.ToTs(now),
		ModifiedTs: schema.ToTs(now),
		Mime:       schema.Pic_GIF,
	}

	if err := p.Insert(c.DB()); err != nil {
		c.T.Fatal(err)
	}

	img := makeImage(p.PicId)
	buf := makeImageData(img, c)
	p.Width = int64(img.Bounds().Dx())
	p.Height = int64(img.Bounds().Dx())
	if err := p.Update(c.DB()); err != nil {
		c.T.Fatal(err)
	}

	h1 := sha256.New()
	h2 := sha1.New()
	h3 := md5.New()
	if err := os.MkdirAll(filepath.Dir(p.Path(c.TempDir())), 0700); err != nil {
		c.T.Fatal(err)
	}
	f, err := os.Create(p.Path(c.TempDir()))
	if err != nil {
		c.T.Fatal(err)
	}
	defer f.Close()
	if err := os.MkdirAll(filepath.Dir(p.ThumbnailPath(c.TempDir())), 0700); err != nil {
		c.T.Fatal(err)
	}
	tf, err := os.Create(p.ThumbnailPath(c.TempDir()))
	if err != nil {
		c.T.Fatal(err)
	}
	defer tf.Close()

	if _, err := io.Copy(io.MultiWriter(h1, h2, h3, f, tf), buf); err != nil {
		c.T.Fatal(err)
	}

	pi1 := &schema.PicIdent{
		PicId: p.PicId,
		Type:  schema.PicIdent_SHA256,
		Value: h1.Sum(nil),
	}
	if err := pi1.Insert(c.DB()); err != nil {
		c.T.Fatal(err, p)
	}

	pi2 := &schema.PicIdent{
		PicId: p.PicId,
		Type:  schema.PicIdent_SHA1,
		Value: h2.Sum(nil),
	}
	if err := pi2.Insert(c.DB()); err != nil {
		c.T.Fatal(err, p)
	}

	pi3 := &schema.PicIdent{
		PicId: p.PicId,
		Type:  schema.PicIdent_MD5,
		Value: h3.Sum(nil),
	}
	if err := pi3.Insert(c.DB()); err != nil {
		c.T.Fatal(err, p)
	}

	return c.WrapPic(p)
}

func makeImageData(img image.Image, c *TestContainer) *bytes.Reader {
	buf := bytes.NewBuffer(nil)
	if err := gif.Encode(buf, img, &gif.Options{}); err != nil {
		c.T.Fatal(err)
	}
	return bytes.NewReader(buf.Bytes())
}

func makeImage(picID int64) image.Image {
	data := make([]uint8, 8)
	binary.LittleEndian.PutUint64(data, uint64(picID))
	return &image.Gray{
		Pix:    data,
		Stride: 8,
		Rect:   image.Rect(0, 0, 8, 1),
	}
}

func (p *TestPic) Update() {
	if err := p.Pic.Update(p.c.DB()); err != nil {
		p.c.T.Fatal(err)
	}
}

func (p *TestPic) Refresh() (exists bool) {
	stmt, err := schema.PicPrepare("SELECT * FROM_ WHERE %s = ?;", p.c.DB(), schema.PicColId)
	if err != nil {
		p.c.T.Fatal(err)
	}
	if updated, err := schema.LookupPic(stmt, p.Pic.PicId); err == sql.ErrNoRows {
		p.Pic = nil
		return false
	} else if err != nil {
		p.c.T.Fatal(err)
		return
	} else {
		p.Pic = updated
		return true
	}
}

type TestPic struct {
	Pic *schema.Pic
	c   *TestContainer
}

type TestTag struct {
	Tag *schema.Tag
	c   *TestContainer
}

type TestPicTag struct {
	TestPic *TestPic
	TestTag *TestTag
	PicTag  *schema.PicTag
	c       *TestContainer
}

type TestPicIdent struct {
	TestPic  *TestPic
	PicIdent *schema.PicIdent
	c        *TestContainer
}

func (p *TestPic) Idents() (picIdents []*TestPicIdent) {
	stmt, err := schema.PicIdentPrepare("SELECT * FROM_ WHERE %s = ?;",
		p.c.DB(), schema.PicIdentColPicId)
	if err != nil {
		p.c.T.Fatal(err)
	}
	defer stmt.Close()

	pis, err := schema.FindPicIdents(stmt, p.Pic.PicId)
	if err != nil {
		p.c.T.Fatal(err)
	}
	for _, pi := range pis {
		picIdents = append(picIdents, &TestPicIdent{
			TestPic:  p,
			PicIdent: pi,
			c:        p.c,
		})
	}
	return
}

func (p *TestPic) Md5() []byte {
	for _, ident := range p.Idents() {
		if ident.PicIdent.Type == schema.PicIdent_MD5 {
			return ident.PicIdent.Value
		}
	}
	p.c.T.Fatal("Can't find MD5")
	return nil
}

func (p *TestPic) Tags() (tags []*TestTag, picTags []*TestPicTag) {
	picTagStmt, err := schema.PicTagPrepare("SELECT * FROM_ WHERE %s = ?;",
		p.c.DB(), schema.PicTagColPicId)
	if err != nil {
		p.c.T.Fatal(err)
	}
	defer picTagStmt.Close()
	pts, err := schema.FindPicTags(picTagStmt, p.Pic.PicId)
	if err != nil {
		p.c.T.Fatal(err)
	}
	tagStmt, err := schema.TagPrepare("SELECT * FROM_ WHERE %s = ?;", p.c.DB(), schema.TagColId)
	if err != nil {
		p.c.T.Fatal(err)
	}
	defer tagStmt.Close()
	for _, pt := range pts {
		tag, err := schema.LookupTag(tagStmt, pt.TagId)
		if err != nil {
			p.c.T.Fatal(err)
		}
		tt := &TestTag{
			Tag: tag,
			c:   p.c,
		}
		tags = append(tags, tt)
		picTags = append(picTags, &TestPicTag{
			TestPic: p,
			TestTag: tt,
			PicTag:  pt,
			c:       p.c,
		})
	}

	return
}

func (c *TestContainer) CreateTag() *TestTag {
	now := time.Now()
	id := c.ID()

	t := &schema.Tag{
		TagId:      id,
		Name:       "tag" + strconv.FormatInt(id, 10),
		CreatedTs:  schema.ToTs(now),
		ModifiedTs: schema.ToTs(now),
	}
	if err := t.Insert(c.DB()); err != nil {
		c.T.Fatal(err, t)
	}
	return &TestTag{
		Tag: t,
		c:   c,
	}
}

func (t *TestTag) Update() {
	if err := t.Tag.Update(t.c.DB()); err != nil {
		t.c.T.Fatal(err)
	}
}

func (t *TestTag) Refresh() (exists bool) {
	stmt, err := schema.TagPrepare("SELECT * FROM_ WHERE %s = ?;", t.c.DB(), schema.TagColId)
	if err != nil {
		t.c.T.Fatal(err)
	}
	if updated, err := schema.LookupTag(stmt, t.Tag.TagId); err == sql.ErrNoRows {
		t.Tag = nil
		return false
	} else if err != nil {
		t.c.T.Fatal(err)
		return
	} else {
		t.Tag = updated
		return true
	}
}

func (c *TestContainer) CreatePicTag(p *TestPic, t *TestTag) *TestPicTag {
	now := time.Now()
	pt := &schema.PicTag{
		PicId:      p.Pic.PicId,
		TagId:      t.Tag.TagId,
		Name:       t.Tag.Name,
		CreatedTs:  schema.ToTs(now),
		ModifiedTs: schema.ToTs(now),
	}
	if _, err := pt.Insert(c.DB()); err != nil {
		c.T.Fatal(err, pt)
	}
	t.Tag.UsageCount++
	t.Tag.ModifiedTs = schema.ToTs(now)
	t.Update()
	return &TestPicTag{
		TestPic: p,
		TestTag: t,
		PicTag:  pt,
		c:       c,
	}
}

func (pt *TestPicTag) Refresh() (exists bool) {
	stmt, err := schema.PicTagPrepare("SELECT * FROM_ WHERE %s = ? AND %s = ?;", pt.c.DB(),
		schema.PicTagColPicId, schema.PicTagColTagId)
	if err != nil {
		pt.c.T.Fatal(err)
	}
	if updated, err := schema.LookupPicTag(stmt, pt.PicTag.PicId, pt.PicTag.TagId); err == sql.ErrNoRows {
		pt.PicTag = nil
		return false
	} else if err != nil {
		pt.c.T.Fatal(err)
		return
	} else {
		pt.PicTag = updated
		return true
	}
}

func (c *TestContainer) ID() int64 {
	id, err := c.IDAlloc()()
	if err != nil {
		c.T.Fatal(err)
	}
	return id
}

func (c *TestContainer) IDAlloc() func() (int64, error) {
	return func() (int64, error) {
		var alloc *schema.IDAllocator
		return alloc.Next(c.DB())
	}
}

func runTests(m *testing.M) int {
	defer ptest.CleanUp()

	return m.Run()
}

func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}
