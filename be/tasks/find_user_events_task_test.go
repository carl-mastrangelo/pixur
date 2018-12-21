package tasks

import (
	"testing"

	"github.com/golang/protobuf/proto"

	"pixur.org/pixur/be/schema"
)

func TestFilterUserEvents_noCap(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	ue := u.CreateEvent()

	conf, sts := GetConfiguration(c.Ctx)
	if sts != nil {
		t.Fatal(sts)
	}

	ues := filterUserEvents([]*schema.UserEvent{ue.UserEvent}, u.User, conf)

	if len(ues) != 0 {
		t.Error(ues)
	}
}

func TestFilterUserEvents_readAll(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u1 := c.CreateUser()
	u2 := c.CreateUser()
	u2.User.Capability = append(u2.User.Capability, schema.User_USER_READ_ALL)
	u2.Update()
	ue := u1.CreateEvent()
	conf, sts := GetConfiguration(c.Ctx)
	if sts != nil {
		t.Fatal(sts)
	}

	ues := filterUserEvents([]*schema.UserEvent{ue.UserEvent}, u2.User, conf)

	if len(ues) != 1 || !proto.Equal(ues[0], ue.UserEvent) {
		t.Error(ues, ue)
	}
}

func TestFilterUserEvents_readSelf(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(u.User.Capability, schema.User_USER_READ_SELF)
	u.Update()
	ue := u.CreateEvent()

	conf, sts := GetConfiguration(c.Ctx)
	if sts != nil {
		t.Fatal(sts)
	}

	ues := filterUserEvents([]*schema.UserEvent{ue.UserEvent}, u.User, conf)

	if len(ues) != 1 || !proto.Equal(ues[0], ue.UserEvent) {
		t.Error(ues, ue)
	}
}

func TestFilterUserEvents_readOtherPicVote(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(
		u.User.Capability, schema.User_USER_READ_PUBLIC, schema.User_USER_READ_PIC_VOTE)
	u.Update()
	ue1 := u.CreateEvent()
	ue1.UserEvent.Evt = &schema.UserEvent_OutgoingUpsertPicVote_{
		OutgoingUpsertPicVote: &schema.UserEvent_OutgoingUpsertPicVote{},
	}
	ue1.Update()
	// make another to show it doesn't show up.
	ue2 := u.CreateEvent()
	ue2.UserEvent.Evt = &schema.UserEvent_IncomingUpsertPicVote_{
		IncomingUpsertPicVote: &schema.UserEvent_IncomingUpsertPicVote{},
	}
	ue2.Update()

	conf, sts := GetConfiguration(c.Ctx)
	if sts != nil {
		t.Fatal(sts)
	}

	ues := filterUserEvents([]*schema.UserEvent{ue1.UserEvent, ue2.UserEvent}, u.User, conf)

	if len(ues) != 1 || !proto.Equal(ues[0], ue1.UserEvent) {
		t.Error(ues, ue1)
	}
}

func TestFilterUserEvents_readOtherPicComment(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(
		u.User.Capability, schema.User_USER_READ_PUBLIC, schema.User_USER_READ_PIC_COMMENT)
	u.Update()
	ue1 := u.CreateEvent()
	ue1.UserEvent.Evt = &schema.UserEvent_OutgoingPicComment_{
		OutgoingPicComment: &schema.UserEvent_OutgoingPicComment{},
	}
	ue1.Update()
	// make another to show it doesn't show up.
	ue2 := u.CreateEvent()
	ue2.UserEvent.Evt = &schema.UserEvent_IncomingPicComment_{
		IncomingPicComment: &schema.UserEvent_IncomingPicComment{},
	}
	ue2.Update()

	conf, sts := GetConfiguration(c.Ctx)
	if sts != nil {
		t.Fatal(sts)
	}

	ues := filterUserEvents([]*schema.UserEvent{ue1.UserEvent, ue2.UserEvent}, u.User, conf)

	if len(ues) != 1 || !proto.Equal(ues[0], ue1.UserEvent) {
		t.Error(ues, ue1)
	}
}

func TestFilterUserEvents_readOtherPic(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	u.User.Capability = append(
		u.User.Capability, schema.User_USER_READ_PUBLIC, schema.User_USER_READ_PICS)
	u.Update()
	ue1 := u.CreateEvent()
	ue1.UserEvent.Evt = &schema.UserEvent_UpsertPic_{
		UpsertPic: &schema.UserEvent_UpsertPic{},
	}
	ue1.Update()
	// make another to show it doesn't show up.
	ue2 := u.CreateEvent()
	ue2.UserEvent.Evt = &schema.UserEvent_OutgoingPicComment_{
		OutgoingPicComment: &schema.UserEvent_OutgoingPicComment{},
	}
	ue2.Update()

	conf, sts := GetConfiguration(c.Ctx)
	if sts != nil {
		t.Fatal(sts)
	}

	ues := filterUserEvents([]*schema.UserEvent{ue1.UserEvent, ue2.UserEvent}, u.User, conf)

	if len(ues) != 1 || !proto.Equal(ues[0], ue1.UserEvent) {
		t.Error(ues, ue1)
	}
}
