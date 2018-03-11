package tasks

import (
	"context"
	"math"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

const (
	defaultDescIndexID = math.MaxInt64
	defaultAscIndexID  = 0
	DefaultMaxPics     = 30
)

type ReadIndexPicsTask struct {
	// Deps
	DB db.DB

	// Inputs
	// Only get pics with Pic Id <= than this.  If unset, the latest pics will be returned.
	StartID int64
	// MaxPics is the maximum number of pics to return.  Note that the number of pictures returned
	// may be less than the number requested.  If unset, the default is used.
	MaxPics int
	// Ascending determines the order of pics returned.
	Ascending bool
	Ctx       context.Context

	// State

	// Results
	Pics []*schema.Pic
}

func (t *ReadIndexPicsTask) ResetForRetry() {
	t.Pics = nil
}

func (t *ReadIndexPicsTask) CleanUp() {
	// no op
}

func lookupStartPic(j *tab.Job, id int64, asc bool) (*schema.Pic, status.S) {
	opts := db.Opts{
		Limit: 1,
	}
	// TODO: This should actually scan for non hidden pics.  If a hidden pic
	// id is used here, the index order will be negative.
	idx := tab.PicsPrimary{
		Id: &id,
	}
	if asc {
		opts.Start = idx
	} else {
		id += 1 // Stop is exclusive, we want inclusive.
		opts.Stop = idx
		opts.Reverse = true
	}
	startPics, err := j.FindPics(opts)
	if err != nil {
		return nil, status.InternalError(err, "Unable to get Start Pic")
	}
	if len(startPics) == 0 {
		// TODO: log info that there were no pics
		return nil, nil
	}
	return startPics[0], nil
}

func (t *ReadIndexPicsTask) Run() (errCap status.S) {
	j, err := tab.NewJob(t.Ctx, t.DB)
	if err != nil {
		return status.InternalError(err, "Unable to Begin TX")
	}
	defer cleanUp(j, &errCap)

	if _, sts := requireCapability(t.Ctx, j, schema.User_PIC_INDEX); sts != nil {
		return sts
	}

	var indexID int64
	if t.StartID != 0 {
		startPic, err := lookupStartPic(j, t.StartID, t.Ascending)
		if err != nil {
			return err
		}
		if startPic == nil {
			return nil
		}
		if t.Ascending {
			indexID = startPic.NonHiddenIndexOrder()
		} else {
			indexID = startPic.NonHiddenIndexOrder() + 1
		}
	} else {
		if t.Ascending {
			indexID = defaultAscIndexID
		} else {
			indexID = defaultDescIndexID
		}
	}

	var maxPics int
	if t.MaxPics != 0 {
		maxPics = t.MaxPics
	} else {
		maxPics = DefaultMaxPics
	}

	opts := db.Opts{
		Limit: maxPics,
	}
	if t.Ascending {
		opts.Start = tab.PicsIndexOrder{
			IndexOrder: &indexID,
		}
	} else {
		opts.Stop = tab.PicsIndexOrder{
			IndexOrder: &indexID,
		}
		min := int64(defaultAscIndexID)
		opts.Start = tab.PicsIndexOrder{
			IndexOrder: &min,
		}
		opts.Reverse = true
	}

	pics, err := j.FindPics(opts)
	if err != nil {
		return status.InternalError(err, "Unable to find pics")
	}

	if err := j.Rollback(); err != nil {
		return status.InternalError(err, "can't rollback job")
	}

	t.Pics = pics

	return nil
}
