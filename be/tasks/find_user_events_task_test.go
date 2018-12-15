package tasks

import (
	"testing"

	"pixur.org/pixur/be/schema"
)

func TestFilterUserEvents_noCap(t *testing.T) {
	c := Container(t)
	defer c.Close()

	u := c.CreateUser()
	ue := u.CreateEvent()
	_ = ue
	conf, sts := GetConfiguration(c.Ctx)
	if sts != nil {
		t.Fatal(sts)
	}

	ues := filterUserEvents([]*schema.UserEvent{ue}, u.User, conf)

	if len(ues) != 0 {
		t.Error(ues)
	}

}
