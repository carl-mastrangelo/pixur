package tasks

import (
	"context"
	"math"

	wpb "github.com/golang/protobuf/ptypes/wrappers"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

const (
	defaultDescIndexID = math.MaxInt64
	defaultAscIndexID  = 0
)

type ReadIndexPicsTask struct {
	// Deps
	Beg tab.JobBeginner

	// Inputs
	// Only get pics with Pic Id <= than this.  If unset, the latest pics will be returned.
	StartID int64
	// MaxPics is the maximum number of pics to return.  Note that the number of pictures returned
	// may be less than the number requested.  If unset, a default is used.
	MaxPics int64
	// Ascending determines the order of pics returned.
	Ascending bool

	// State

	// Results
	Pics []*schema.Pic

	NextID, PrevID int64
}

// This may return a deleted pic
func lookupStartPic(j *tab.Job, id int64, asc bool) (*schema.Pic, status.S) {
	var opts db.Opts
	if asc {
		opts = db.Opts{
			Limit: 1,
			Start: tab.PicsPrimary{
				Id: &id,
			},
		}
	} else {
		// Stop is exclusive, we want inclusive.
		upperId := id + 1
		opts = db.Opts{
			Limit:   1,
			Reverse: true,
			Stop: tab.PicsPrimary{
				Id: &upperId,
			},
		}
	}
	startPics, err := j.FindPics(opts)
	if err != nil {
		return nil, status.Internal(err, "can't find start pic")
	}
	if len(startPics) == 0 {
		return nil, nil
	}
	p := startPics[0]
	return p, nil
}

func getAndValidateMaxPics(ctx context.Context, requestedMax int64) (max, overmax int64, _ status.S) {
	if requestedMax < 0 {
		return 0, 0, status.InvalidArgument(nil, "negative max pics")
	}
	conf, sts := GetConfiguration(ctx)
	if sts != nil {
		return 0, 0, sts
	}
	maxPics, overMaxPics := getMaxPics(requestedMax, conf)
	return maxPics, overMaxPics, nil
}

func getMaxPics(requestedMax int64, conf *schema.Configuration) (max, overmax int64) {
	return getMaxConf(requestedMax, conf.DefaultFindIndexPics, conf.MaxFindIndexPics)
}

func getMaxConf(requestedMax int64, confDefault, confMax *wpb.Int64Value) (max, overmax int64) {
	var confDefaultPresent, confMaxPresent int64
	if confDefault != nil {
		confDefaultPresent = confDefault.Value
	} else {
		confDefaultPresent = math.MaxInt64 // seems crazy, but there is no default.
	}
	if confMax != nil {
		confMaxPresent = confMax.Value
	} else {
		confMaxPresent = math.MaxInt64
	}

	var maxPresent int64
	if requestedMax != 0 {
		if requestedMax < confMaxPresent {
			maxPresent = requestedMax
		} else {
			maxPresent = confMaxPresent
		}
	} else {
		maxPresent = confDefaultPresent
	}
	overMax := maxPresent
	if overMax < math.MaxInt64 {
		overMax++
	}
	return maxPresent, overMax
}

func (t *ReadIndexPicsTask) Run(ctx context.Context) (stscap status.S) {
	j, err := tab.NewJob(ctx, t.Beg)
	if err != nil {
		return status.Internal(err, "Unable to Begin TX")
	}
	defer revert(j, &stscap)

	if _, sts := requireCapability(ctx, j, schema.User_PIC_INDEX); sts != nil {
		return sts
	}

	_, overmax, sts := getAndValidateMaxPics(ctx, t.MaxPics)
	if sts != nil {
		return sts
	}

	var startPic *schema.Pic
	if t.StartID != 0 {
		if startPic, sts = lookupStartPic(j, t.StartID, t.Ascending); sts != nil {
			return sts
		}
	}

	minIndexOrder := int64(defaultAscIndexID)
	maxIndexOrder := int64(defaultDescIndexID)
	minIndexOrderPicId := int64(0)
	maxIndexOrderPicId := int64(math.MaxInt64)
	if startPic != nil {
		if t.Ascending {
			minIndexOrder = startPic.NonHiddenIndexOrder()
			minIndexOrderPicId = startPic.PicId
		} else {
			maxIndexOrder = startPic.NonHiddenIndexOrder()
			maxIndexOrderPicId = startPic.PicId + 1 // Stop is exclusive, we want inclusive
		}
	}

	opts := db.Opts{
		Limit: int(overmax),
		Lock:  db.LockNone,
		Start: tab.PicsIndexOrder{
			IndexOrder: &minIndexOrder,
			Id:         &minIndexOrderPicId,
		},
		Stop: tab.PicsIndexOrder{
			IndexOrder: &maxIndexOrder,
			Id:         &maxIndexOrderPicId,
		},
		Reverse: !t.Ascending,
	}

	pics, err := j.FindPics(opts)
	if err != nil {
		return status.Internal(err, "Unable to find pics")
	}

	var prevPicId int64
	if startPic != nil {
		if t.Ascending {
			maxIndexOrder, maxIndexOrderPicId = minIndexOrder, minIndexOrderPicId
			minIndexOrder, minIndexOrderPicId = defaultAscIndexID, 0
		} else {
			minIndexOrder, minIndexOrderPicId = maxIndexOrder, maxIndexOrderPicId
			maxIndexOrder, maxIndexOrderPicId = defaultDescIndexID, math.MaxInt64
		}
		prevOpts := db.Opts{
			Limit: 1,
			Lock:  db.LockNone,
			Start: tab.PicsIndexOrder{
				IndexOrder: &minIndexOrder,
				Id:         &minIndexOrderPicId,
			},
			Stop: tab.PicsIndexOrder{
				IndexOrder: &maxIndexOrder,
				Id:         &maxIndexOrderPicId,
			},
			Reverse: t.Ascending,
		}
		prevPics, err := j.FindPics(prevOpts)
		if err != nil {
			return status.Internal(err, "Unable to find prev pics")
		}
		if len(prevPics) > 0 {
			prevPicId = prevPics[0].PicId
		}
	}

	if n := len(pics); n > 0 && int64(n) == overmax {
		t.Pics = pics[:n-1]
		t.NextID = pics[n-1].PicId
	} else {
		t.Pics = pics
	}
	t.PrevID = prevPicId

	return nil
}
