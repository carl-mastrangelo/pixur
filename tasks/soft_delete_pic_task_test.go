package tasks

import (
	"os"
	"testing"
	"time"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
)

func TestSoftDeleteWorkflow(test *testing.T) {
	c := &container{
		t:  test,
		db: testDB,
	}
	defer c.CleanUp()

	p := c.CreatePic()

	task := &SoftDeletePicTask{
		DB:     testDB,
		PicId:  p.PicId,
		Reason: "LowQuality",
	}

	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		test.Fatal(err)
	}

	if _, err := os.Stat(p.Path(c.pixPath)); os.IsNotExist(err) {
		test.Fatal("Expected file to exist", err)
	}

	c.RefreshPic(&p)

	if !p.SoftDeleted() {
		test.Fatal("Expected pic to be soft deleted", p)
	}
	if p.HardDeleted() {
		test.Fatal("Expected pic not to be hard deleted", p)
	}

	if p.DeletionStatus.Reason != "LowQuality" {
		test.Fatal("Reason not preserved", p)
	}

	if p.DeletionStatus.PendingDeletedTs != nil {
		test.Fatal("Should not have a pending deleted timestamp", p)
	}
}

func TestSoftDelete_OverwritePendingTimestamp(test *testing.T) {
	c := &container{
		t:  test,
		db: testDB,
	}
	defer c.CleanUp()

	now := time.Now().UTC()
	then := now.AddDate(0, 0, -1)

	p := c.CreatePic()
	p.DeletionStatus = &schema.Pic_DeletionStatus{
		MarkedDeletedTs:  schema.FromTime(then),
		PendingDeletedTs: schema.FromTime(then),
	}
	if err := p.Update(testDB); err != nil {
		test.Fatal(err)
	}

	task := &SoftDeletePicTask{
		DB:                  testDB,
		PicId:               p.PicId,
		PendingDeletionTime: &now,
	}

	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		test.Fatal(err)
	}

	c.RefreshPic(&p)

	if schema.ToTime(p.DeletionStatus.MarkedDeletedTs) != then {
		test.Fatal("Marked deleted timestamp not preserved", p, then)
	}

	if schema.ToTime(p.DeletionStatus.PendingDeletedTs) != now {
		test.Fatal("Pending deleted timestamp not incremented", p, then)
	}
}

func TestSoftDelete_CannotSoftDeleteHardDeletedPic(test *testing.T) {
	c := &container{
		t:  test,
		db: testDB,
	}
	defer c.CleanUp()

	now := time.Now().UTC()

	p := c.CreatePic()
	p.DeletionStatus = &schema.Pic_DeletionStatus{
		MarkedDeletedTs: schema.FromTime(now),
		ActualDeletedTs: schema.FromTime(now),
	}
	if err := p.Update(testDB); err != nil {
		test.Fatal(err)
	}

	task := &SoftDeletePicTask{
		DB:                  testDB,
		PicId:               p.PicId,
		PendingDeletionTime: &now,
	}

	runner := new(TaskRunner)
	if err := runner.Run(task); err != nil {
		if st, ok := err.(status.Status); !ok {
			test.Fatal(err)
		} else {
			if st.GetCode() != status.Code_INVALID_ARGUMENT {
				test.Fatal(st)
			}
		}

	}
}
