package tasks

import (
	"testing"
	"time"

	anypb "github.com/golang/protobuf/ptypes/any"

	"pixur.org/pixur/be/schema"
)

func TestUpdatePicTagWorkflow(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_TAG_EXTENSION_CREATE)
	u.Update()

	p := c.CreatePic()
	tag := c.CreateTag()
	pt := c.CreatePicTag(p, tag)

	ctx := u.AuthedCtx(c.Ctx)
	task := &UpdatePicTagTask{
		Beg: c.DB(),
		Now: time.Now,

		PicId:   pt.PicTag.PicId,
		TagId:   pt.PicTag.TagId,
		Version: pt.PicTag.Version(),

		Ext: map[string]*anypb.Any{"foo": nil},
	}

	sts := new(TaskRunner).Run(ctx, task)
	if sts != nil {
		t.Fatal(sts)
	}
	pt.Refresh()

	if _, present := pt.PicTag.Ext["foo"]; !present {
		t.Error("missing ext")
	}
}
