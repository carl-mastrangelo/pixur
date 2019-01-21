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
	defaultDescIndexId = math.MaxInt64
	defaultAscIndexId  = 0
)

type ReadIndexPicsTask struct {
	// Deps
	Beg tab.JobBeginner

	// Inputs
	// Only get pics with Pic Id <= than this.  If unset, the latest pics will be returned.
	StartId int64
	// MaxPics is the maximum number of pics to return.  Note that the number of pictures returned
	// may be less than the number requested.  If unset, a default is used.
	MaxPics int64
	// Ascending determines the order of pics returned.
	Ascending bool

	// Results
	UnfilteredPics []*schema.Pic
	// Same as pics, but with User info removed based on capability
	Pics []*schema.Pic

	NextId, PrevId int64
}

// This may return a deleted pic
func lookupStartPic(j *tab.Job, id int64, asc bool) (*schema.Pic, status.S) {
	var opts db.Opts
	if asc {
		opts = db.Opts{
			Limit: 1,
			StartInc: tab.PicsPrimary{
				Id: &id,
			},
		}
	} else {
		opts = db.Opts{
			Limit:   1,
			Reverse: true,
			StopInc: tab.PicsPrimary{
				Id: &id,
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

func getAndValidateMaxPics(conf *schema.Configuration, requestedMax int64) (
	max, overmax int64, _ status.S) {
	if requestedMax < 0 {
		return 0, 0, status.InvalidArgument(nil, "negative max pics")
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

	u, sts := requireCapability(ctx, j, schema.User_PIC_INDEX)
	if sts != nil {
		return sts
	}
	conf, sts := GetConfiguration(ctx)
	if sts != nil {
		return sts
	}

	_, overmax, sts := getAndValidateMaxPics(conf, t.MaxPics)
	if sts != nil {
		return sts
	}

	var startPic *schema.Pic
	if t.StartId != 0 {
		if startPic, sts = lookupStartPic(j, t.StartId, t.Ascending); sts != nil {
			return sts
		}
	}

	minIndexOrder := int64(defaultAscIndexId)
	maxIndexOrder := int64(defaultDescIndexId)
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
		StartInc: tab.PicsIndexOrder{
			IndexOrder: &minIndexOrder,
			Id:         &minIndexOrderPicId,
		},
		StopEx: tab.PicsIndexOrder{
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
			minIndexOrder, minIndexOrderPicId = defaultAscIndexId, 0
		} else {
			minIndexOrder, minIndexOrderPicId = maxIndexOrder, maxIndexOrderPicId
			maxIndexOrder, maxIndexOrderPicId = defaultDescIndexId, math.MaxInt64
		}
		prevOpts := db.Opts{
			Limit: 1,
			Lock:  db.LockNone,
			StartInc: tab.PicsIndexOrder{
				IndexOrder: &minIndexOrder,
				Id:         &minIndexOrderPicId,
			},
			StopEx: tab.PicsIndexOrder{
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
		t.UnfilteredPics = pics[:n-1]
		t.NextId = pics[n-1].PicId
	} else {
		t.UnfilteredPics = pics
	}
	t.PrevId = prevPicId
	t.Pics = filterPics(t.UnfilteredPics, u, conf)

	return nil
}

func filterPic(p *schema.Pic, su *schema.User, conf *schema.Configuration) *schema.Pic {
	var cs *schema.CapSet
	var subjectUserId int64
	if su != nil {
		cs = schema.CapSetOf(su.Capability...)
		subjectUserId = su.UserId
	} else {
		cs = schema.CapSetOf(conf.AnonymousCapability.Capability...)
		subjectUserId = schema.AnonymousUserId
	}

	return filterPicInternal(p, subjectUserId, cs)
}

func filterPics(ps []*schema.Pic, su *schema.User, conf *schema.Configuration) []*schema.Pic {
	var cs *schema.CapSet
	var subjectUserId int64
	if su != nil {
		cs = schema.CapSetOf(su.Capability...)
		subjectUserId = su.UserId
	} else {
		cs = schema.CapSetOf(conf.AnonymousCapability.Capability...)
		subjectUserId = schema.AnonymousUserId
	}
	dst := make([]*schema.Pic, 0, len(ps))
	for _, p := range ps {
		dst = append(dst, filterPicInternal(p, subjectUserId, cs))
	}
	return dst
}

// TODO: test
func filterPicInternal(p *schema.Pic, subjectUserId int64, cs *schema.CapSet) *schema.Pic {
	dp := *p
	if !cs.Has(schema.User_PIC_EXTENSION_READ) {
		dp.Ext = nil
	}
	switch {
	case cs.Has(schema.User_USER_READ_ALL):
	case cs.Has(schema.User_USER_READ_PUBLIC) && cs.Has(schema.User_USER_READ_PICS):
	default:
		dp.Source = nil
		for _, s := range p.Source {
			ds := *s
			switch {
			case subjectUserId == ds.UserId && cs.Has(schema.User_USER_READ_SELF):
			default:
				ds.UserId = schema.AnonymousUserId
			}
			dp.Source = append(dp.Source, &ds)
		}
	}
	return &dp
}
