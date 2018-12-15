package tasks

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
)

var sqlAdapterName = "sqlite3"

type TestContainer struct {
	T       testing.TB
	Ctx     context.Context
	cancel  func()
	db      db.DB
	tempdir string
}

func Container(t testing.TB) *TestContainer {
	ctx, cancel := context.WithCancel(context.Background())
	c := &TestContainer{
		T:      t,
		Ctx:    ctx,
		cancel: cancel,
	}
	task := &LoadConfigurationTask{
		Beg: c.DB(),
	}
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}
	return c
}

var allDbs []db.DB

func (c *TestContainer) DB() db.DB {
	if c.db == nil {
		db, err := db.OpenForTest(c.Ctx, sqlAdapterName)
		if err != nil {
			c.T.Fatal(err)
		}
		allDbs = append(allDbs, db)
		var stmts []string
		stmts = append(stmts, tab.SqlTables[db.Adapter().Name()]...)
		stmts = append(stmts, tab.SqlInitTables[db.Adapter().Name()]...)
		if err := db.InitSchema(c.Ctx, stmts); err != nil {
			c.T.Fatal(err)
		}
		c.db = db
	}
	return c.db
}

func (c *TestContainer) Close() {
	c.cancel()
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
	j, err := tab.NewJob(c.Ctx, c.DB())
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
		PicId: c.ID(),
		File: &schema.Pic_File{
			Index: 0,
			Mime:  schema.Pic_File_PNG,
		},
		Thumbnail: []*schema.Pic_File{{
			Index: 0,
			Mime:  schema.Pic_File_PNG,
		}},
	}
	p.SetCreatedTime(now)
	p.File.CreatedTs = p.CreatedTs
	p.Thumbnail[0].CreatedTs = p.File.CreatedTs
	p.SetModifiedTime(now)
	p.File.ModifiedTs = p.ModifiedTs
	p.Thumbnail[0].ModifiedTs = p.File.ModifiedTs

	u := c.CreateUser()
	p.Source = []*schema.Pic_FileSource{{
		UserId:    u.User.UserId,
		CreatedTs: p.CreatedTs,
	}}

	c.AutoJob(func(j *tab.Job) error {
		return j.InsertPic(p)
	})

	img := makeImage(p.PicId)
	buf := makeImageData(img, c)
	p.File.Width = int64(img.Bounds().Dx())
	p.Thumbnail[0].Width = p.File.Width
	p.File.Height = int64(img.Bounds().Dx())
	p.Thumbnail[0].Height = p.File.Height
	c.AutoJob(func(j *tab.Job) error {
		return j.UpdatePic(p)
	})

	h1 := sha512.New512_256()
	h2 := sha1.New()
	h3 := md5.New()
	base := schema.PicBaseDir(c.TempDir(), p.PicId)
	if err := os.MkdirAll(base, 0700); err != nil {
		c.T.Fatal(err)
	}
	path, sts := schema.PicFilePath(c.TempDir(), p.PicId, p.File.Mime)
	if sts != nil {
		c.T.Fatal(sts)
	}
	f, err := os.Create(path)
	if err != nil {
		c.T.Fatal(err)
	}
	defer f.Close()
	thumbpath, sts := schema.PicFileThumbnailPath(
		c.TempDir(), p.PicId, p.Thumbnail[0].Index, p.Thumbnail[0].Mime)
	if sts != nil {
		c.T.Fatal(sts)
	}
	tf, err := os.Create(thumbpath)
	if err != nil {
		c.T.Fatal(err)
	}
	defer tf.Close()

	if _, err := io.Copy(io.MultiWriter(h1, h2, h3, f, tf), buf); err != nil {
		c.T.Fatal(err)
	}

	pi1 := &schema.PicIdent{
		PicId: p.PicId,
		Type:  schema.PicIdent_SHA512_256,
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
		TagId: id,
		Name:  "tag" + strconv.FormatInt(id, 10),
	}
	t.SetCreatedTime(now)
	t.SetModifiedTime(now)
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
		PicId: p.Pic.PicId,
		TagId: t.Tag.TagId,
		Name:  t.Tag.Name,
	}
	pt.SetCreatedTime(now)
	pt.SetModifiedTime(now)
	c.AutoJob(func(j *tab.Job) error {
		return j.InsertPicTag(pt)
	})
	t.Tag.UsageCount++
	t.Tag.SetModifiedTime(now)
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
	}
	pc.SetCreatedTime(now)
	pc.SetModifiedTime(now)
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
		UserId: id,
		Secret: hashed,
		Ident:  fmt.Sprintf("%d@example.com", id),
	}
	u.SetCreatedTime(now)
	u.SetModifiedTime(now)
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

func (u *TestUser) CreateEvent() *schema.UserEvent {
	now := time.Now()
	nowts := schema.ToTspb(now)
	ue := &schema.UserEvent{
		UserId:     u.User.UserId,
		CreatedTs:  nowts,
		ModifiedTs: nowts,
		Index:      0,
	}
	u.c.AutoJob(func(j *tab.Job) error {
		return j.InsertUserEvent(ue)
	})
	return ue
}

type TestUserEvent struct {
	UserEvent *schema.UserEvent
	c         *TestContainer
}

func (ue *TestUserEvent) Update() {
	ue.c.AutoJob(func(j *tab.Job) error {
		return j.UpdateUserEvent(ue.UserEvent)
	})
}

func (ue *TestUserEvent) Refresh() (exists bool) {
	ue.c.AutoJob(func(j *tab.Job) error {
		ues, err := j.FindUserEvents(db.Opts{
			Prefix: tab.KeyForUserEvent(ue.UserEvent),
		})
		if err != nil {
			return err
		}
		if len(ues) == 1 {
			ue.UserEvent = ues[0]
			exists = true
		} else {
			ue.UserEvent = nil
			exists = false
		}
		return nil
	})
	return
}

type TestPicVote struct {
	PicVote *schema.PicVote
	c       *TestContainer
}

func (c *TestContainer) CreatePicVote(p *TestPic, u *TestUser) *TestPicVote {
	now := time.Now()
	pv := &schema.PicVote{
		PicId:  p.Pic.PicId,
		UserId: u.User.UserId,
		Vote:   schema.PicVote_NEUTRAL,
	}
	pv.SetCreatedTime(now)
	pv.SetModifiedTime(now)

	c.AutoJob(func(j *tab.Job) error {
		return j.InsertPicVote(pv)
	})

	return &TestPicVote{
		PicVote: pv,
		c:       c,
	}
}

func (pv *TestPicVote) Update() {
	pv.c.AutoJob(func(j *tab.Job) error {
		return j.UpdatePicVote(pv.PicVote)
	})
}

func (pv *TestPicVote) Refresh() (exists bool) {
	pv.c.AutoJob(func(j *tab.Job) error {
		pvs, err := j.FindPicVotes(db.Opts{
			Prefix: tab.PicVotesPrimary{
				PicId:  &pv.PicVote.PicId,
				UserId: &pv.PicVote.UserId,
			},
		})
		if err != nil {
			return err
		}
		if len(pvs) == 1 {
			pv.PicVote = pvs[0]
			exists = true
		} else {
			pv.PicVote = nil
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
	db, err := db.OpenForTest(context.Background(), sqlAdapterName)
	if err != nil {
		panic(err)
	}
	allDbs = append(allDbs, db)

	return m.Run()
}

func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}
