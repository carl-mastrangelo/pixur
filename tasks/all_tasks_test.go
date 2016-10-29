package tasks

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/schema/db"
	tab "pixur.org/pixur/schema/tables"
)

var sqlAdapterName = "mysql"

type TestContainer struct {
	T       testing.TB
	db      db.DB
	tempdir string
}

func Container(t testing.TB) *TestContainer {
	return &TestContainer{
		T: t,
	}
}

var allDbs []db.DB

func (c *TestContainer) DB() db.DB {
	if c.db == nil {
		db, err := db.OpenForTest(sqlAdapterName)
		if err != nil {
			c.T.Fatal(err)
		}
		allDbs = append(allDbs, db)
		var stmts []string
		stmts = append(stmts, tab.SqlTables[db.Adapter().Name()]...)
		stmts = append(stmts, tab.SqlInitTables[db.Adapter().Name()]...)
		if err := db.InitSchema(stmts); err != nil {
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

func (c *TestContainer) Job() *tab.Job {
	j, err := tab.NewJob(c.DB())
	if err != nil {
		c.T.Fatal(err)
	}
	return j
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

func (c *TestContainer) AutoJob(cb func(j *tab.Job) error) {
	j := c.Job()
	if err := cb(j); err != nil {
		c.T.Log("Failure: ", err)
		if err := j.Rollback(); err != nil {
			c.T.Log("Also Failure: ", err)
		}
		c.T.FailNow()
	}
	if err := j.Commit(); err != nil {
		c.T.Log("Failure: ", err)
		c.T.FailNow()
	}
}

func (c *TestContainer) CreatePic() *TestPic {
	now := time.Now()
	p := &schema.Pic{
		PicId:      c.ID(),
		CreatedTs:  schema.ToTs(now),
		ModifiedTs: schema.ToTs(now),
		Mime:       schema.Pic_PNG,
	}

	c.AutoJob(func(j *tab.Job) error {
		return j.InsertPic(p)
	})

	img := makeImage(p.PicId)
	buf := makeImageData(img, c)
	p.Width = int64(img.Bounds().Dx())
	p.Height = int64(img.Bounds().Dx())
	c.AutoJob(func(j *tab.Job) error {
		return j.UpdatePic(p)
	})

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
	c.AutoJob(func(j *tab.Job) error {
		return j.InsertPicIdent(pi1)
	})

	pi2 := &schema.PicIdent{
		PicId: p.PicId,
		Type:  schema.PicIdent_SHA1,
		Value: h2.Sum(nil),
	}
	c.AutoJob(func(j *tab.Job) error {
		return j.InsertPicIdent(pi2)
	})

	pi3 := &schema.PicIdent{
		PicId: p.PicId,
		Type:  schema.PicIdent_MD5,
		Value: h3.Sum(nil),
	}
	c.AutoJob(func(j *tab.Job) error {
		return j.InsertPicIdent(pi3)
	})

	return c.WrapPic(p)
}

func makeImageData(img image.Image, c *TestContainer) *bytes.Reader {
	buf := bytes.NewBuffer(nil)
	if err := png.Encode(buf, img); err != nil {
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
	p.c.AutoJob(func(j *tab.Job) error {
		return j.UpdatePic(p.Pic)
	})
}

func (p *TestPic) Refresh() (exists bool) {
	p.c.AutoJob(func(j *tab.Job) error {
		pics, err := j.FindPics(db.Opts{
			Prefix: tab.PicsPrimary{&p.Pic.PicId},
		})
		if err != nil {
			return err
		}
		if len(pics) == 1 {
			p.Pic = pics[0]
			exists = true
		} else {
			p.Pic = nil
			exists = false
		}
		return nil
	})
	return
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
	p.c.AutoJob(func(j *tab.Job) error {
		pis, err := j.FindPicIdents(db.Opts{
			Prefix: tab.PicIdentsPrimary{PicId: &p.Pic.PicId},
		})
		if err != nil {
			return err
		}
		for _, pi := range pis {
			picIdents = append(picIdents, &TestPicIdent{
				TestPic:  p,
				PicIdent: pi,
				c:        p.c,
			})
		}
		return nil
	})
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
	p.c.AutoJob(func(j *tab.Job) error {
		pts, err := j.FindPicTags(db.Opts{
			Prefix: tab.PicTagsPrimary{PicId: &p.Pic.PicId},
		})
		if err != nil {
			return err
		}
		for _, pt := range pts {
			ts, err := j.FindTags(db.Opts{
				Prefix: tab.TagsPrimary{&pt.TagId},
			})
			if err != nil {
				return err
			}
			tt := &TestTag{
				Tag: ts[0],
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
		return nil
	})
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
	c.AutoJob(func(j *tab.Job) error {
		return j.InsertTag(t)
	})
	return &TestTag{
		Tag: t,
		c:   c,
	}
}

func (t *TestTag) Update() {
	t.c.AutoJob(func(j *tab.Job) error {
		return j.UpdateTag(t.Tag)
	})
}

func (t *TestTag) Refresh() (exists bool) {
	t.c.AutoJob(func(j *tab.Job) error {
		tags, err := j.FindTags(db.Opts{
			Prefix: tab.TagsPrimary{&t.Tag.TagId},
		})
		if err != nil {
			return err
		}
		if len(tags) == 1 {
			t.Tag = tags[0]
			exists = true
		} else {
			t.Tag = nil
			exists = false
		}
		return nil
	})
	return
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
	c.AutoJob(func(j *tab.Job) error {
		return j.InsertPicTag(pt)
	})
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
	pt.c.AutoJob(func(j *tab.Job) error {
		pts, err := j.FindPicTags(db.Opts{
			Prefix: tab.PicTagsPrimary{&pt.PicTag.PicId, &pt.PicTag.TagId},
		})
		if err != nil {
			return nil
		}
		if len(pts) == 1 {
			pt.PicTag = pts[0]
			exists = true
		} else {
			pt.PicTag = nil
			exists = false
		}
		return nil
	})
	return
}

type TestPicComment struct {
	PicComment *schema.PicComment
	c          *TestContainer
}

func (p *TestPic) Comment() *TestPicComment {
	return p.c.createPicComment(p.Pic.PicId, 0)
}

func (pc *TestPicComment) Comment() *TestPicComment {
	return pc.c.createPicComment(pc.PicComment.PicId, pc.PicComment.CommentId)
}

func (c *TestContainer) createPicComment(picID, commentParentID int64) *TestPicComment {
	now := time.Now()
	id := c.ID()

	pc := &schema.PicComment{
		PicId:           picID,
		CommentParentId: commentParentID,
		CommentId:       id,
		CreatedTs:       schema.ToTs(now),
		ModifiedTs:      schema.ToTs(now),
	}
	c.AutoJob(func(j *tab.Job) error {
		return j.InsertPicComment(pc)
	})
	return &TestPicComment{
		PicComment: pc,
		c:          c,
	}
}

func (pc *TestPicComment) Update() {
	pc.c.AutoJob(func(j *tab.Job) error {
		return j.UpdatePicComment(pc.PicComment)
	})
}

func (pc *TestPicComment) Refresh() (exists bool) {
	pc.c.AutoJob(func(j *tab.Job) error {
		pcs, err := j.FindPicComments(db.Opts{
			Prefix: tab.PicCommentsPrimary{&pc.PicComment.PicId, &pc.PicComment.CommentId},
		})
		if err != nil {
			return err
		}
		if len(pcs) == 1 {
			pc.PicComment = pcs[0]
			exists = true
		} else {
			pc.PicComment = nil
			exists = false
		}
		return nil
	})
	return
}

type TestUser struct {
	User *schema.User
	c    *TestContainer
}

func (c *TestContainer) CreateUser() *TestUser {
	now := time.Now()
	id := c.ID()

	hashed, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	if err != nil {
		c.T.Fatal(err)
	}

	u := &schema.User{
		UserId:     id,
		Secret:     hashed,
		Ident:      fmt.Sprintf("%d@example.com", id),
		CreatedTs:  schema.ToTs(now),
		ModifiedTs: schema.ToTs(now),
	}
	c.AutoJob(func(j *tab.Job) error {
		return j.InsertUser(u)
	})
	return &TestUser{
		User: u,
		c:    c,
	}
}

func (u *TestUser) Update() {
	u.c.AutoJob(func(j *tab.Job) error {
		return j.UpdateUser(u.User)
	})
}

func (u *TestUser) Refresh() (exists bool) {
	u.c.AutoJob(func(j *tab.Job) error {
		users, err := j.FindUsers(db.Opts{
			Prefix: tab.UsersPrimary{&u.User.UserId},
		})
		if err != nil {
			return err
		}
		if len(users) == 1 {
			u.User = users[0]
			exists = true
		} else {
			u.User = nil
			exists = false
		}
		return nil
	})
	return
}

func (c *TestContainer) ID() int64 {
	var idCap int64
	c.AutoJob(func(j *tab.Job) error {
		id, err := j.AllocID()
		if err != nil {
			return err
		}
		idCap = id
		return nil
	})
	return idCap
}

func runTests(m *testing.M) int {
	defer func() {
		for _, db := range allDbs {
			db.Close()
		}
	}()

	// Open a dummy db to keep the server alive during tests
	db, err := db.OpenForTest(sqlAdapterName)
	if err != nil {
		panic(err)
	}
	allDbs = append(allDbs, db)

	return m.Run()
}

func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}
