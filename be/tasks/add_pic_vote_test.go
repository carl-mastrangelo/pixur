package tasks

import (
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	anypb "github.com/golang/protobuf/ptypes/any"
	"google.golang.org/grpc/codes"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
)

func TestAddPicVoteTaskWorkFlow(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_VOTE_CREATE)
	u.Update()

	p := c.CreatePic()

	task := &AddPicVoteTask{
		Vote:  schema.PicVote_UP,
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   time.Now,
	}
	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	then := schema.ToTime(p.Pic.ModifiedTs)
	p.Refresh()

	if p.Pic.VoteUp != 1 || p.Pic.VoteDown != 0 {
		t.Error("wrong vote count", p.Pic)
	}
	if schema.ToTime(p.Pic.ModifiedTs).Before(then) {
		t.Error("modified time not updated")
	}

	if task.PicVote == nil || task.UnfilteredPicVote == nil {
		t.Fatal("no vote created")
	}

	if task.PicVote.CreatedTs == nil || !proto.Equal(task.PicVote.CreatedTs, task.PicVote.ModifiedTs) {
		t.Error("wrong timestamps", task.PicVote)
	}

	expected := &schema.PicVote{
		PicId:  p.Pic.PicId,
		UserId: u.User.UserId,
		Vote:   schema.PicVote_UP,
	}
	task.UnfilteredPicVote.CreatedTs = nil
	task.UnfilteredPicVote.ModifiedTs = nil

	if !proto.Equal(expected, task.UnfilteredPicVote) {
		t.Error("have", task.UnfilteredPicVote, "want", expected)
	}
}

func TestAddPicVoteTaskWork_NoDoubleVoting(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_VOTE_CREATE)
	u.Update()

	p := c.CreatePic()

	task := &AddPicVoteTask{
		Vote:  schema.PicVote_UP,
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   time.Now,
	}
	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	sts := new(TaskRunner).Run(ctx, task)
	if sts == nil {
		t.Fatal("expected non-nil status")
	}
	if have, want := sts.Code(), codes.AlreadyExists; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't double vote"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}

	p.Refresh()

	if p.Pic.VoteUp != 1 {
		t.Error("double voted")
	}
}

func TestAddPicVoteTaskWork_MissingPic(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_VOTE_CREATE)
	u.Update()

	task := &AddPicVoteTask{
		Vote:  schema.PicVote_UP,
		PicID: 0,
		Beg:   c.DB(),
		Now:   time.Now,
	}
	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	if sts == nil {
		t.Fatal("expected non-nil status")
	}
	if have, want := sts.Code(), codes.NotFound; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't find pic"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAddPicVoteTaskWork_CantVoteOnHardDeleted(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_VOTE_CREATE)
	u.Update()

	p := c.CreatePic()

	nowTs := schema.ToTspb(time.Now())
	p.Pic.DeletionStatus = &schema.Pic_DeletionStatus{
		MarkedDeletedTs:  nowTs,
		PendingDeletedTs: nowTs,
		ActualDeletedTs:  nowTs,
	}

	p.Update()

	task := &AddPicVoteTask{
		Vote:  schema.PicVote_UP,
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   time.Now,
	}
	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	sts := new(TaskRunner).Run(ctx, task)
	if sts == nil {
		t.Fatal("expected non-nil status")
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "can't vote on deleted pic"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAddPicVoteTask_BadVoteDir(t *testing.T) {
	c := Container(t)
	defer c.Close()

	task := &AddPicVoteTask{
		Vote: schema.PicVote_UNKNOWN,
		Beg:  c.DB(),
		Now:  time.Now,
	}

	sts := new(TaskRunner).Run(c.Ctx, task)
	if sts == nil {
		t.Fatal("expected non-nil status")
	}
	if have, want := sts.Code(), codes.InvalidArgument; have != want {
		t.Error("have", have, "want", want)
	}
	if have, want := sts.Message(), "bad vote dir"; !strings.Contains(have, want) {
		t.Error("have", have, "want", want)
	}
}

func TestAddPicVoteTask_AnonymousAllowed(t *testing.T) {
	c := Container(t)
	defer c.Close()

	conf := schema.GetDefaultConfiguration()
	conf.AnonymousCapability.Capability =
		append(conf.AnonymousCapability.Capability, schema.User_PIC_VOTE_CREATE)
	ctx := CtxFromTestConfig(c.Ctx, conf)

	p := c.CreatePic()

	task := &AddPicVoteTask{
		Vote:  schema.PicVote_UP,
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   time.Now,
	}

	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	then := schema.ToTime(p.Pic.ModifiedTs)
	p.Refresh()

	if p.Pic.VoteUp != 1 || p.Pic.VoteDown != 0 {
		t.Error("wrong vote count", p.Pic)
	}
	if schema.ToTime(p.Pic.ModifiedTs).Before(then) {
		t.Error("modified time not updated")
	}

	if task.PicVote == nil {
		t.Fatal("no vote created")
	}

	if task.PicVote.CreatedTs == nil || !proto.Equal(task.PicVote.CreatedTs, task.PicVote.ModifiedTs) {
		t.Error("wrong timestamps", task.PicVote)
	}

	expected := &schema.PicVote{
		PicId:  p.Pic.PicId,
		UserId: schema.AnonymousUserID,
		Vote:   schema.PicVote_UP,
	}
	task.PicVote.CreatedTs = nil
	task.PicVote.ModifiedTs = nil

	if !proto.Equal(expected, task.PicVote) {
		t.Error("have", task.PicVote, "want", expected)
	}
}

func TestAddPicVoteTask_AnonymousAllowed_DoubleVote(t *testing.T) {
	c := Container(t)
	defer c.Close()

	conf := schema.GetDefaultConfiguration()
	conf.AnonymousCapability.Capability =
		append(conf.AnonymousCapability.Capability, schema.User_PIC_VOTE_CREATE)
	ctx := CtxFromTestConfig(c.Ctx, conf)

	p := c.CreatePic()

	task1 := &AddPicVoteTask{
		Vote:  schema.PicVote_UP,
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   time.Now,
	}

	if sts := new(TaskRunner).Run(ctx, task1); sts != nil {
		t.Fatal(sts)
	}

	task2 := &AddPicVoteTask{
		Vote:  schema.PicVote_DOWN,
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   time.Now,
	}

	if sts := new(TaskRunner).Run(ctx, task2); sts != nil {
		t.Fatal(sts)
	}

	then := schema.ToTime(p.Pic.ModifiedTs)
	p.Refresh()

	if p.Pic.VoteUp != 1 || p.Pic.VoteDown != 1 {
		t.Error("wrong vote count", p.Pic)
	}
	if schema.ToTime(p.Pic.ModifiedTs).Before(then) {
		t.Error("modified time not updated")
	}

	if task2.PicVote == nil {
		t.Fatal("no vote created")
	}

	if task2.PicVote.CreatedTs == nil ||
		!proto.Equal(task2.PicVote.CreatedTs, task2.PicVote.ModifiedTs) {
		t.Error("wrong timestamps", task2.PicVote)
	}

	expected := &schema.PicVote{
		PicId:  p.Pic.PicId,
		UserId: schema.AnonymousUserID,
		Index:  1,
		Vote:   schema.PicVote_DOWN,
	}
	task2.PicVote.CreatedTs = nil
	task2.PicVote.ModifiedTs = nil

	if !proto.Equal(expected, task2.PicVote) {
		t.Error("have", task2.PicVote, "want", expected)
	}
}

func TestAddPicVote_Notification_Author_AnonPicParent(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_VOTE_CREATE)
	u.Update()
	p := c.CreatePic()
	for _, s := range p.Pic.Source {
		s.UserId = schema.AnonymousUserID
	}
	p.Update()

	tm := time.Now()
	now := func() time.Time { return tm }

	task := &AddPicVoteTask{
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   now,
		Vote:  schema.PicVote_UP,
	}

	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	j := c.Job()
	defer j.Rollback()

	ues, err := j.FindUserEvents(db.Opts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ues) != 1 {
		t.Fatal("wrong number of events", ues)
	}
	expect := &schema.UserEvent{
		UserId:     u.User.UserId,
		CreatedTs:  schema.ToTspb(now()),
		ModifiedTs: schema.ToTspb(now()),
		Evt: &schema.UserEvent_OutgoingUpsertPicVote_{
			OutgoingUpsertPicVote: &schema.UserEvent_OutgoingUpsertPicVote{
				PicId: p.Pic.PicId,
			},
		},
	}

	if !proto.Equal(expect, ues[0]) {
		t.Error("have", expect, "want", ues[0])
	}
}

func TestAddPicVote_Notification_AnonAuthor_AnonPicParent(t *testing.T) {
	c := Container(t)
	defer c.Close()

	conf := schema.GetDefaultConfiguration()
	conf.AnonymousCapability.Capability =
		append(conf.AnonymousCapability.Capability, schema.User_PIC_VOTE_CREATE)
	ctx := CtxFromTestConfig(c.Ctx, conf)

	p := c.CreatePic()
	for _, s := range p.Pic.Source {
		s.UserId = schema.AnonymousUserID
	}
	p.Update()

	tm := time.Now()
	now := func() time.Time { return tm }

	task := &AddPicVoteTask{
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   now,
		Vote:  schema.PicVote_UP,
	}

	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	j := c.Job()
	defer j.Rollback()

	ues, err := j.FindUserEvents(db.Opts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ues) != 0 {
		t.Fatal("wrong number of events", ues)
	}
}

func TestAddPicVote_Notification_Author_PicParent(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_VOTE_CREATE)
	u.Update()
	p := c.CreatePic()
	u2 := c.CreateUser()
	p.Pic.Source = []*schema.Pic_FileSource{{
		UserId: u2.User.UserId,
	}}
	p.Update()

	tm := time.Now()
	now := func() time.Time { return tm }

	task := &AddPicVoteTask{
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   now,
		Vote:  schema.PicVote_UP,
	}

	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	j := c.Job()
	defer j.Rollback()

	ues, err := j.FindUserEvents(db.Opts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ues) != 2 {
		t.Fatal("wrong number of events", ues)
	}
	expect1 := &schema.UserEvent{
		UserId:     u.User.UserId,
		CreatedTs:  schema.ToTspb(now()),
		ModifiedTs: schema.ToTspb(now()),
		Evt: &schema.UserEvent_OutgoingUpsertPicVote_{
			OutgoingUpsertPicVote: &schema.UserEvent_OutgoingUpsertPicVote{
				PicId: p.Pic.PicId,
			},
		},
	}
	expect2 := &schema.UserEvent{
		UserId:     u2.User.UserId,
		CreatedTs:  schema.ToTspb(now()),
		ModifiedTs: schema.ToTspb(now()),
		Evt: &schema.UserEvent_IncomingUpsertPicVote_{
			IncomingUpsertPicVote: &schema.UserEvent_IncomingUpsertPicVote{
				PicId:         p.Pic.PicId,
				SubjectUserId: u.User.UserId,
			},
		},
	}
	found := 0
	for _, ue := range ues {
		if proto.Equal(expect1, ue) {
			expect1 = nil
			found++
		}
		if proto.Equal(expect2, ue) {
			expect2 = nil
			found++
		}
	}

	if found != 2 {
		t.Error("missing events", ues)
	}
}

func TestAddPicVote_Notification_Author_AuthorPicParent(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_VOTE_CREATE)
	u.Update()
	p := c.CreatePic()
	p.Pic.Source = []*schema.Pic_FileSource{{
		UserId: u.User.UserId,
	}}
	p.Update()

	tm := time.Now()
	now := func() time.Time { return tm }

	task := &AddPicVoteTask{
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   now,
		Vote:  schema.PicVote_UP,
	}

	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	j := c.Job()
	defer j.Rollback()

	ues, err := j.FindUserEvents(db.Opts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ues) != 1 {
		t.Fatal("wrong number of events", ues)
	}
	expect1 := &schema.UserEvent{
		UserId:     u.User.UserId,
		CreatedTs:  schema.ToTspb(now()),
		ModifiedTs: schema.ToTspb(now()),
		Evt: &schema.UserEvent_OutgoingUpsertPicVote_{
			OutgoingUpsertPicVote: &schema.UserEvent_OutgoingUpsertPicVote{
				PicId: p.Pic.PicId,
			},
		},
	}

	found := 0
	for _, ue := range ues {
		if proto.Equal(expect1, ue) {
			expect1 = nil
			found++
		}
	}

	if found != 1 {
		t.Error("missing events", ues)
	}
}

func TestAddPicVote_Notification_AnonAuthor_PicParent(t *testing.T) {
	c := Container(t)
	defer c.Close()

	conf := schema.GetDefaultConfiguration()
	conf.AnonymousCapability.Capability =
		append(conf.AnonymousCapability.Capability, schema.User_PIC_VOTE_CREATE)
	ctx := CtxFromTestConfig(c.Ctx, conf)

	p := c.CreatePic()
	u2 := c.CreateUser()
	p.Pic.Source = []*schema.Pic_FileSource{{
		UserId: u2.User.UserId,
	}}
	p.Update()

	tm := time.Now()
	now := func() time.Time { return tm }

	task := &AddPicVoteTask{
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   now,
		Vote:  schema.PicVote_UP,
	}

	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	j := c.Job()
	defer j.Rollback()

	ues, err := j.FindUserEvents(db.Opts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ues) != 1 {
		t.Fatal("wrong number of events", ues)
	}

	expect2 := &schema.UserEvent{
		UserId:     u2.User.UserId,
		CreatedTs:  schema.ToTspb(now()),
		ModifiedTs: schema.ToTspb(now()),
		Evt: &schema.UserEvent_IncomingUpsertPicVote_{
			IncomingUpsertPicVote: &schema.UserEvent_IncomingUpsertPicVote{
				PicId:         p.Pic.PicId,
				SubjectUserId: schema.AnonymousUserID,
			},
		},
	}
	found := 0
	for _, ue := range ues {
		if proto.Equal(expect2, ue) {
			expect2 = nil
			found++
		}
	}

	if found != 1 {
		t.Error("missing events", ues)
	}
}

// Checks to see a next index is used.
func TestAddPicVote_Notification_Author_PicParent_ExistingEvents(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u1 := c.CreateUser()
	u1.User.Capability = append(u1.User.Capability, schema.User_PIC_VOTE_CREATE)
	u1.Update()

	p := c.CreatePic()
	u2 := c.CreateUser()
	p.Pic.Source = []*schema.Pic_FileSource{{
		UserId: u2.User.UserId,
	}}
	p.Update()

	tm := time.Now()
	now := func() time.Time { return tm }

	c.AutoJob(func(j *tab.Job) error {
		return j.InsertUserEvent(&schema.UserEvent{
			UserId:     u2.User.UserId,
			CreatedTs:  schema.ToTspb(now()),
			ModifiedTs: schema.ToTspb(now()),
			Evt: &schema.UserEvent_IncomingUpsertPicVote_{
				IncomingUpsertPicVote: &schema.UserEvent_IncomingUpsertPicVote{
					PicId:         p.Pic.PicId,
					SubjectUserId: u1.User.UserId,
				},
			},
		})
	})

	task := &AddPicVoteTask{
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   now,
		Vote:  schema.PicVote_UP,
	}

	ctx := CtxFromUserID(c.Ctx, u1.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	j := c.Job()
	defer j.Rollback()

	ues, err := j.FindUserEvents(db.Opts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ues) != 3 {
		t.Fatal("wrong number of events", ues)
	}

	expect2 := &schema.UserEvent{
		UserId:     u2.User.UserId,
		CreatedTs:  schema.ToTspb(now()),
		ModifiedTs: schema.ToTspb(now()),
		Index:      1,
		Evt: &schema.UserEvent_IncomingUpsertPicVote_{
			IncomingUpsertPicVote: &schema.UserEvent_IncomingUpsertPicVote{
				PicId:         p.Pic.PicId,
				SubjectUserId: u1.User.UserId,
			},
		},
	}
	found := 2 // ignore the original one and the one of the news
	for _, ue := range ues {
		if proto.Equal(expect2, ue) {
			expect2 = nil
			found++
		}
	}

	if found != 3 {
		t.Error("missing events", ues)
	}
}

func TestAddPicVote_Notification_Author_AnonPicParent_Neutral(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_VOTE_CREATE)
	u.Update()
	p := c.CreatePic()
	for _, s := range p.Pic.Source {
		s.UserId = schema.AnonymousUserID
	}
	p.Update()

	tm := time.Now()
	now := func() time.Time { return tm }

	task := &AddPicVoteTask{
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   now,
		Vote:  schema.PicVote_NEUTRAL,
	}

	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	j := c.Job()
	defer j.Rollback()

	ues, err := j.FindUserEvents(db.Opts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ues) != 0 {
		t.Fatal("wrong number of events", ues)
	}
}

func TestAddPicVote_Notification_AnonAuthor_AnonPicParent_Neutral(t *testing.T) {
	c := Container(t)
	defer c.Close()

	conf := schema.GetDefaultConfiguration()
	conf.AnonymousCapability.Capability =
		append(conf.AnonymousCapability.Capability, schema.User_PIC_VOTE_CREATE)
	ctx := CtxFromTestConfig(c.Ctx, conf)

	p := c.CreatePic()
	for _, s := range p.Pic.Source {
		s.UserId = schema.AnonymousUserID
	}
	p.Update()

	tm := time.Now()
	now := func() time.Time { return tm }

	task := &AddPicVoteTask{
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   now,
		Vote:  schema.PicVote_NEUTRAL,
	}

	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	j := c.Job()
	defer j.Rollback()

	ues, err := j.FindUserEvents(db.Opts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ues) != 0 {
		t.Fatal("wrong number of events", ues)
	}
}

func TestAddPicVote_Notification_Author_PicParent_Neutral(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_VOTE_CREATE)
	u.Update()
	p := c.CreatePic()
	u2 := c.CreateUser()
	p.Pic.Source = []*schema.Pic_FileSource{{
		UserId: u2.User.UserId,
	}}
	p.Update()

	tm := time.Now()
	now := func() time.Time { return tm }

	task := &AddPicVoteTask{
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   now,
		Vote:  schema.PicVote_NEUTRAL,
	}

	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	j := c.Job()
	defer j.Rollback()

	ues, err := j.FindUserEvents(db.Opts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ues) != 0 {
		t.Fatal("wrong number of events", ues)
	}
}

func TestAddPicVote_Notification_Author_AuthorPicParent_Neutral(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_PIC_VOTE_CREATE)
	u.Update()
	p := c.CreatePic()
	p.Pic.Source = []*schema.Pic_FileSource{{
		UserId: u.User.UserId,
	}}
	p.Update()

	tm := time.Now()
	now := func() time.Time { return tm }

	task := &AddPicVoteTask{
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   now,
		Vote:  schema.PicVote_NEUTRAL,
	}

	ctx := CtxFromUserID(c.Ctx, u.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	j := c.Job()
	defer j.Rollback()

	ues, err := j.FindUserEvents(db.Opts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ues) != 0 {
		t.Fatal("wrong number of events", ues)
	}
}

func TestAddPicVote_Notification_AnonAuthor_PicParent_Neutral(t *testing.T) {
	c := Container(t)
	defer c.Close()

	conf := schema.GetDefaultConfiguration()
	conf.AnonymousCapability.Capability =
		append(conf.AnonymousCapability.Capability, schema.User_PIC_VOTE_CREATE)
	ctx := CtxFromTestConfig(c.Ctx, conf)

	p := c.CreatePic()
	u2 := c.CreateUser()
	p.Pic.Source = []*schema.Pic_FileSource{{
		UserId: u2.User.UserId,
	}}
	p.Update()

	tm := time.Now()
	now := func() time.Time { return tm }

	task := &AddPicVoteTask{
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   now,
		Vote:  schema.PicVote_NEUTRAL,
	}

	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	j := c.Job()
	defer j.Rollback()

	ues, err := j.FindUserEvents(db.Opts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ues) != 0 {
		t.Fatal("wrong number of events", ues)
	}
}

// Checks to see a next index is used.
func TestAddPicVote_Notification_Author_PicParent_ExistingEvents_Neutral(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u1 := c.CreateUser()
	u1.User.Capability = append(u1.User.Capability, schema.User_PIC_VOTE_CREATE)
	u1.Update()

	p := c.CreatePic()
	u2 := c.CreateUser()
	p.Pic.Source = []*schema.Pic_FileSource{{
		UserId: u2.User.UserId,
	}}
	p.Update()

	tm := time.Now()
	now := func() time.Time { return tm }

	c.AutoJob(func(j *tab.Job) error {
		return j.InsertUserEvent(&schema.UserEvent{
			UserId:     u2.User.UserId,
			CreatedTs:  schema.ToTspb(now()),
			ModifiedTs: schema.ToTspb(now()),
			Evt: &schema.UserEvent_IncomingUpsertPicVote_{
				IncomingUpsertPicVote: &schema.UserEvent_IncomingUpsertPicVote{
					PicId:         p.Pic.PicId,
					SubjectUserId: u1.User.UserId,
				},
			},
		})
	})

	task := &AddPicVoteTask{
		PicID: p.Pic.PicId,
		Beg:   c.DB(),
		Now:   now,
		Vote:  schema.PicVote_NEUTRAL,
	}

	ctx := CtxFromUserID(c.Ctx, u1.User.UserId)
	if sts := new(TaskRunner).Run(ctx, task); sts != nil {
		t.Fatal(sts)
	}

	j := c.Job()
	defer j.Rollback()

	ues, err := j.FindUserEvents(db.Opts{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ues) != 1 {
		t.Fatal("wrong number of events", ues)
	}
}

func TestFilterPicVoteInternal_extAllowed(t *testing.T) {
	pv := &schema.PicVote{
		PicId:  1,
		UserId: 5,
		Index:  0,
		Ext:    map[string]*anypb.Any{"six": nil},
	}
	dupe := *pv
	uc := &userCred{
		subjectUserId: 5,
		cs:            schema.CapSetOf(schema.User_USER_READ_SELF, schema.User_PIC_VOTE_EXTENSION_READ),
	}
	pvd, _ := filterPicVoteInternal(pv, uc)
	if !proto.Equal(pv, &dupe) {
		t.Error("original changed", pv, dupe)
	}
	if !proto.Equal(pv, pvd) {
		t.Error("missing field", pv, pvd)
	}
}

func TestFilterPicVoteInternal_extRemoved(t *testing.T) {
	pv := &schema.PicVote{
		PicId:  1,
		UserId: 5,
		Index:  0,
		Ext:    map[string]*anypb.Any{"six": nil},
	}
	dupe := *pv
	uc := &userCred{
		subjectUserId: 5,
		cs:            schema.CapSetOf(schema.User_USER_READ_SELF),
	}
	pvd, _ := filterPicVoteInternal(pv, uc)
	if !proto.Equal(pv, &dupe) {
		t.Error("original changed", pv, dupe)
	}
	pv.Ext = nil
	if !proto.Equal(pv, pvd) {
		t.Error("expected ext removed", pv, pvd)
	}
}

func TestFilterPicVoteInternal_userReadAll(t *testing.T) {
	pv := &schema.PicVote{
		PicId:  1,
		UserId: 5,
		Index:  0,
	}
	dupe := *pv
	uc := &userCred{
		subjectUserId: 7,
		cs:            schema.CapSetOf(schema.User_USER_READ_ALL),
	}
	pvd, _ := filterPicVoteInternal(pv, uc)
	if !proto.Equal(pv, &dupe) {
		t.Error("original changed", pv, dupe)
	}
	if !proto.Equal(pv, pvd) {
		t.Error("missing field", pv, pvd)
	}
}

func TestFilterPicVoteInternal_userReadPicVote(t *testing.T) {
	pv := &schema.PicVote{
		PicId:  1,
		UserId: 5,
		Index:  0,
	}
	dupe := *pv
	uc := &userCred{
		subjectUserId: schema.AnonymousUserID,
		cs:            schema.CapSetOf(schema.User_USER_READ_PUBLIC, schema.User_USER_READ_PIC_VOTE),
	}
	pvd, _ := filterPicVoteInternal(pv, uc)
	if !proto.Equal(pv, &dupe) {
		t.Error("original changed", pv, dupe)
	}
	if !proto.Equal(pv, pvd) {
		t.Error("missing field", pv, pvd)
	}
}

func TestFilterPicVoteInternal_userReadSelf(t *testing.T) {
	pv := &schema.PicVote{
		PicId:  1,
		UserId: 5,
		Index:  0,
	}
	dupe := *pv
	uc := &userCred{
		subjectUserId: 5,
		cs:            schema.CapSetOf(schema.User_USER_READ_SELF),
	}
	pvd, _ := filterPicVoteInternal(pv, uc)
	if !proto.Equal(pv, &dupe) {
		t.Error("original changed", pv, dupe)
	}
	if !proto.Equal(pv, pvd) {
		t.Error("missing field", pv, pvd)
	}
}

func TestFilterPicVoteInternal_userIdRemoved(t *testing.T) {
	pv := &schema.PicVote{
		PicId:  1,
		UserId: 5,
		Index:  0,
	}
	dupe := *pv
	uc := &userCred{
		subjectUserId: 7,
		cs:            schema.CapSetOf(schema.User_USER_READ_SELF),
	}
	pvd, _ := filterPicVoteInternal(pv, uc)
	if !proto.Equal(pv, &dupe) {
		t.Error("original changed", pv, dupe)
	}
	pv.UserId = schema.AnonymousUserID
	if !proto.Equal(pv, pvd) {
		t.Error("missing field", pv, pvd)
	}
}

func TestFilterPicVote(t *testing.T) {
	pv := &schema.PicVote{
		PicId:  1,
		UserId: 5,
		Index:  0,
	}
	dupe := *pv
	u := &schema.User{
		UserId: 7,
	}
	pvd := filterPicVote(pv, u, schema.GetDefaultConfiguration())
	if !proto.Equal(pv, &dupe) {
		t.Error("original changed", pv, dupe)
	}
	pv.UserId = schema.AnonymousUserID
	if !proto.Equal(pv, pvd) {
		t.Error("missing field", pv, pvd)
	}
}

func TestFilterPicVotes(t *testing.T) {
	pv1 := &schema.PicVote{
		PicId:  1,
		UserId: 5,
		Index:  0,
	}
	pv2 := &schema.PicVote{
		PicId:  1,
		UserId: 7,
		Index:  0,
	}
	dupe := *pv2
	u := &schema.User{
		UserId:     7,
		Capability: []schema.User_Capability{schema.User_USER_READ_SELF},
	}
	pvsd := filterPicVotes([]*schema.PicVote{pv1, pv2}, u, schema.GetDefaultConfiguration())
	if !proto.Equal(pv2, &dupe) {
		t.Error("original changed", pv2, dupe)
	}
	if len(pvsd) != 1 || !proto.Equal(pv2, pvsd[0]) {
		t.Error("expected field", pv2, pvsd)
	}
}
