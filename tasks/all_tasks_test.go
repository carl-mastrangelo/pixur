package tasks

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"image"
	"image/color"
	"image/gif"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"pixur.org/pixur/schema"
	ptest "pixur.org/pixur/testing"
)

type container struct {
	t       testing.TB
	db      *sql.DB
	tempDir string
}

func NewContainer(t testing.TB) *container {
	if t == nil {
		panic("t cannot be nil")
	}
	return &container{
		t: t,
	}
}

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
		CreatedTs:  schema.ToTs(now),
		ModifiedTs: schema.ToTs(now),
		Mime:       schema.Pic_GIF,
	}

	if err := p.Insert(c.DB()); err != nil {
		c.T.Fatal(err)
	}

	img := makeImage(p.PicId)
	buf := bytes.NewBuffer(nil)
	if err := gif.Encode(buf, img, &gif.Options{}); err != nil {
		c.T.Fatal(err)
	}
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

	if _, err := io.Copy(io.MultiWriter(h1, h2, h3, f, tf), bytes.NewReader(buf.Bytes())); err != nil {
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

func (c *container) GetDB() *sql.DB {
	if c.db == nil {
		db, err := ptest.GetDB()
		if err != nil {
			c.t.Fatal(err)
		}
		if err := schema.CreateTables(db); err != nil {
			c.t.Fatal(err)
		}
		c.db = db
	}
	return c.db
}

func (c *container) GetTx() *sql.Tx {
	db := c.GetDB()
	tx, err := db.Begin()
	if err != nil {
		c.t.Fatal(err)
	}
	return tx
}

func (c *container) GetTempDir() string {
	if c.tempDir != "" {
		return c.tempDir
	}
	if path, err := ioutil.TempDir("", "unitTestTempDir"); err != nil {
		c.t.Fatal(err)
	} else {
		c.tempDir = path
	}
	return c.tempDir
}

func (c *container) GetTempFile() *os.File {
	f, err := ioutil.TempFile(c.GetTempDir(), "__")
	if err != nil {
		c.t.Fatal(err)
	}
	return f
}

func (c *container) CleanUp() {
	if c.db != nil {
		if err := c.db.Close(); err != nil {
			c.t.Error(err)
		}
	}
	c.db = nil
	if c.tempDir != "" {
		if err := os.RemoveAll(c.tempDir); err != nil {
			c.t.Error(err)
		}
	}
	c.tempDir = ""
}

func (c *container) CreatePic() *schema.Pic {
	p := &schema.Pic{}

	if err := p.Insert(c.GetDB()); err != nil {
		c.t.Fatal(err, p)
	}

	h1 := sha256.New()
	h2 := sha1.New()
	h3 := md5.New()
	if err := binary.Write(io.MultiWriter(h1, h2, h3), binary.LittleEndian, p.PicId); err != nil {
		c.t.Fatal(err)
	}

	pi1 := &schema.PicIdent{
		PicId: p.PicId,
		Type:  schema.PicIdent_SHA256,
		Value: h1.Sum(nil),
	}
	if err := pi1.Insert(c.GetDB()); err != nil {
		c.t.Fatal(err, p)
	}

	pi2 := &schema.PicIdent{
		PicId: p.PicId,
		Type:  schema.PicIdent_SHA1,
		Value: h2.Sum(nil),
	}
	if err := pi2.Insert(c.GetDB()); err != nil {
		c.t.Fatal(err, p)
	}

	pi3 := &schema.PicIdent{
		PicId: p.PicId,
		Type:  schema.PicIdent_MD5,
		Value: h3.Sum(nil),
	}
	if err := pi3.Insert(c.GetDB()); err != nil {
		c.t.Fatal(err, p)
	}

	if err := c.writeImageData(p); err != nil {
		c.t.Fatal(err)
	}
	if err := c.writeThumbnailData(p); err != nil {
		c.t.Fatal(err)
	}

	return p
}

func (c *container) CreateTag() *schema.Tag {
	dictionary := "abcdefghijklmnopqrstuvwxyz"
	var name string
	for i := 0; i < 6; i++ {
		name += string(dictionary[rand.Intn(len(dictionary))])
	}
	t := &schema.Tag{Name: name}
	if err := t.Insert(c.GetDB()); err != nil {
		c.t.Fatal(err)
	}

	return t
}

func (c *container) CreatePicTag(p *schema.Pic, t *schema.Tag) *schema.PicTag {
	picTag := &schema.PicTag{
		PicId: p.PicId,
		TagId: t.TagId,
		Name:  t.Name,
	}
	if _, err := picTag.Insert(c.GetDB()); err != nil {
		c.t.Fatal(err)
	}
	t.UsageCount++
	if err := t.Update(c.GetDB()); err != nil {
		c.t.Fatal(err)
	}

	return picTag
}

func (c *container) RefreshPic(p **schema.Pic) {
	stmt, err := schema.PicPrepare("SELECT * FROM_ WHERE %s = ?;", c.GetDB(), schema.PicColId)
	if err != nil {
		c.t.Fatal(err)
	}
	updated, err := schema.LookupPic(stmt, (*p).PicId)
	if err == sql.ErrNoRows {
		*p = nil
	} else if err != nil {
		c.t.Fatal(err)
	}
	*p = updated
}

func (c *container) UpdatePic(p *schema.Pic) {
	if err := p.Update(c.GetDB()); err != nil {
		c.t.Fatal(err)
	}
}

func (c *container) RefreshTag(t **schema.Tag) {
	stmt, err := schema.TagPrepare("SELECT * FROM_ WHERE %s = ?;", c.GetDB(), schema.TagColId)
	if err != nil {
		c.t.Fatal(err)
	}
	updated, err := schema.LookupTag(stmt, (*t).TagId)
	if err == sql.ErrNoRows {
		*t = nil
	} else if err != nil {
		c.t.Fatal(err)
	}
	*t = updated
}

func (c *container) RefreshPicTag(pt **schema.PicTag) {
	stmt, err := schema.PicTagPrepare("SELECT * FROM_ WHERE %s = ? AND %s = ?;",
		c.GetDB(), schema.PicTagColPicId, schema.PicTagColTagId)
	if err != nil {
		c.t.Fatal(err)
	}
	updated, err := schema.LookupPicTag(stmt, (*pt).PicId, (*pt).TagId)
	if err == sql.ErrNoRows {
		*pt = nil
	} else if err != nil {
		c.t.Fatal(err)
	}
	*pt = updated
}

func (c *container) getRandomImageData() *bytes.Reader {
	bounds := image.Rect(0, 0, 5, 10)
	img := image.NewGray(bounds)
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			img.SetGray(x, y, color.Gray{Y: uint8(rand.Int31n(math.MaxUint8))})
		}
	}
	f := bytes.NewBuffer(nil)

	if err := gif.Encode(f, img, &gif.Options{}); err != nil {
		c.t.Fatal(err)
	}
	return bytes.NewReader(f.Bytes())
}

func (c *container) writeImageData(p *schema.Pic) error {
	path := p.Path(c.GetTempDir())
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0770); err != nil {
		c.t.Fatal(err)
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, c.getRandomImageData()); err != nil {
		return err
	}
	return nil
}

func (c *container) writeThumbnailData(p *schema.Pic) error {
	path := p.ThumbnailPath(c.GetTempDir())
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0770); err != nil {
		c.t.Fatal(err)
	}
	f, err := os.Create(p.ThumbnailPath(c.GetTempDir()))
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, c.getRandomImageData()); err != nil {
		return err
	}
	return nil
}

func runTests(m *testing.M) int {
	defer ptest.CleanUp()

	return m.Run()
}

func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}
