package tasks

import (
	"testing"
)

func TestPicViewCountUpdated(t *testing.T) {
	ctnr := NewContainer(t)
	defer ctnr.CleanUp()

	p := ctnr.CreatePic()
	oldTime := p.GetModifiedTime()

	task := IncrementViewCountTask{
		DB:    ctnr.GetDB(),
		PicID: p.PicId,
	}
	if err := task.Run(); err != nil {
		t.Fatal(err)
	}

	ctnr.RefreshPic(&p)
	if p.ViewCount != 1 {
		t.Fatalf("Expected view count %v but was %v", 1, p.ViewCount)
	}
	if p.GetModifiedTime() == oldTime {
		t.Fatalf("Expected Modified Time to be updated but is  %v but was %v", oldTime)
	}
}
