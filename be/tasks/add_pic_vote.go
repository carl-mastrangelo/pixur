package tasks

import (
	"context"
	"math"
	"time"

	anypb "github.com/golang/protobuf/ptypes/any"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

type AddPicVoteTask struct {
	// Deps
	Beg tab.JobBeginner
	Now func() time.Time

	// Inputs
	PicID int64
	Vote  schema.PicVote_Vote
	// Ext is additional extra data associated with this vote.
	Ext map[string]*anypb.Any

	// Outs
	UnfilteredPicVote *schema.PicVote
	PicVote           *schema.PicVote
}

func (t *AddPicVoteTask) Run(ctx context.Context) (stscap status.S) {
	if t.Vote != schema.PicVote_UP && t.Vote != schema.PicVote_DOWN && t.Vote != schema.PicVote_NEUTRAL {
		return status.InvalidArgument(nil, "bad vote dir", t.Vote)
	}

	j, err := tab.NewJob(ctx, t.Beg)
	if err != nil {
		return status.Internal(err, "can't create job")
	}
	defer revert(j, &stscap)

	u, sts := requireCapability(ctx, j, schema.User_PIC_VOTE_CREATE)
	if sts != nil {
		return sts
	}
	userID := schema.AnonymousUserID
	picVoteIndex := int64(0)
	if u != nil {
		userID = u.UserId
	}

	conf, sts := GetConfiguration(ctx)
	if sts != nil {
		return sts
	}

	if len(t.Ext) != 0 {
		if sts := validateCapability(u, conf, schema.User_PIC_VOTE_EXTENSION_CREATE); sts != nil {
			return sts
		}
	}

	pics, err := j.FindPics(db.Opts{
		Prefix: tab.PicsPrimary{&t.PicID},
		Lock:   db.LockWrite,
	})
	if err != nil {
		return status.Internal(err, "can't lookup pic", t.PicID)
	}
	if len(pics) != 1 {
		return status.NotFound(nil, "can't find pic", t.PicID)
	}
	p := pics[0]

	if p.HardDeleted() {
		return status.InvalidArgument(nil, "can't vote on deleted pic", t.PicID)
	}

	pvs, err := j.FindPicVotes(db.Opts{
		Prefix: tab.PicVotesPrimary{
			PicId:  &t.PicID,
			UserId: &userID,
		},
		Lock: db.LockWrite,
	})
	if err != nil {
		return status.Internal(err, "can't find pic votes")
	}
	if userID != schema.AnonymousUserID {
		if len(pvs) != 0 {
			return status.AlreadyExists(nil, "can't double vote")
		}
	} else {
		biggest := int64(-1)
		for _, pv := range pvs {
			if pv.Index > biggest {
				biggest = pv.Index
			}
		}
		if biggest == math.MaxInt64 {
			return status.Internal(nil, "overflow of pic vote index")
		}
		picVoteIndex = biggest + 1
	}

	now := t.Now()
	pv := &schema.PicVote{
		PicId:  p.PicId,
		UserId: userID,
		Index:  picVoteIndex,
		Vote:   t.Vote,
		Ext:    t.Ext,
	}
	pv.SetCreatedTime(now)
	pv.SetModifiedTime(now)

	if err := j.InsertPicVote(pv); err != nil {
		return status.Internal(err, "can't insert vote")
	}
	pic_updated := false
	switch pv.Vote {
	case schema.PicVote_UP:
		p.VoteUp++
		pic_updated = true
	case schema.PicVote_DOWN:
		p.VoteDown++
		pic_updated = true
	}
	if pic_updated {
		p.SetModifiedTime(now)
		if err := j.UpdatePic(p); err != nil {
			return status.Internal(err, "can't update pic")
		}
	}

	// only create events if the pic was updated.
	if pic_updated {
		createdTs := schema.UserEventCreatedTsCol(pv.CreatedTs)
		next := func(uid int64) (int64, status.S) {
			return nextUserEventIndex(j, uid, createdTs)
		}
		var iues []*schema.UserEvent
		var oue *schema.UserEvent
		notifications := make(map[int64]bool)
		if userID != schema.AnonymousUserID {
			idx, sts := next(userID)
			if sts != nil {
				return sts
			}
			notifications[userID] = true
			oue = &schema.UserEvent{
				UserId:     userID,
				Index:      idx,
				CreatedTs:  pv.CreatedTs,
				ModifiedTs: pv.ModifiedTs,
				Evt: &schema.UserEvent_OutgoingUpsertPicVote_{
					OutgoingUpsertPicVote: &schema.UserEvent_OutgoingUpsertPicVote{
						PicId: p.PicId,
					},
				},
			}
		}
		for _, fs := range p.Source {
			if fs.UserId != schema.AnonymousUserID && !notifications[fs.UserId] {
				idx, sts := next(fs.UserId)
				if sts != nil {
					return sts
				}
				notifications[fs.UserId] = true
				iues = append(iues, &schema.UserEvent{
					UserId:     fs.UserId,
					Index:      idx,
					CreatedTs:  pv.CreatedTs,
					ModifiedTs: pv.ModifiedTs,
					Evt: &schema.UserEvent_IncomingUpsertPicVote_{
						IncomingUpsertPicVote: &schema.UserEvent_IncomingUpsertPicVote{
							PicId:         p.PicId,
							SubjectUserId: userID,
						},
					},
				})
			}
		}

		// In the future, these notifications could be done outside of the transaction.
		if oue != nil {
			if err := j.InsertUserEvent(oue); err != nil {
				return status.Internal(err, "can't create outgoing user event")
			}
		}
		for _, iue := range iues {
			if err := j.InsertUserEvent(iue); err != nil {
				return status.Internal(err, "can't create incoming user event")
			}
		}
	}

	if err := j.Commit(); err != nil {
		return status.Internal(err, "can't commit job")
	}
	t.UnfilteredPicVote = pv
	t.PicVote = filterPicVote(t.UnfilteredPicVote, u, conf)

	// TODO: ratelimit

	return nil
}

func filterPicVote(
	pv *schema.PicVote, su *schema.User, conf *schema.Configuration) *schema.PicVote {
	var cs *schema.CapSet
	var subjectUserId int64
	if su != nil {
		cs = schema.CapSetOf(su.Capability...)
		subjectUserId = su.UserId
	} else {
		cs = schema.CapSetOf(conf.AnonymousCapability.Capability...)
		subjectUserId = schema.AnonymousUserID
	}

	return filterPicVoteInternal(pv, subjectUserId, cs)
}

func filterPicVotes(
	pvs []*schema.PicVote, su *schema.User, conf *schema.Configuration) []*schema.PicVote {
	var cs *schema.CapSet
	var subjectUserId int64
	if su != nil {
		cs = schema.CapSetOf(su.Capability...)
		subjectUserId = su.UserId
	} else {
		cs = schema.CapSetOf(conf.AnonymousCapability.Capability...)
		subjectUserId = schema.AnonymousUserID
	}
	dst := make([]*schema.PicVote, 0, len(pvs))
	for _, pv := range pvs {
		dst = append(dst, filterPicVoteInternal(pv, subjectUserId, cs))
	}
	return dst
}

func filterPicVoteInternal(
	pv *schema.PicVote, subjectUserId int64, cs *schema.CapSet) *schema.PicVote {
	dpv := *pv
	if !cs.Has(schema.User_PIC_VOTE_EXTENSION_READ) {
		dpv.Ext = nil
	}
	switch {
	case cs.Has(schema.User_USER_READ_ALL):
	case cs.Has(schema.User_USER_READ_PUBLIC) && cs.Has(schema.User_USER_READ_PIC_VOTE):
	case subjectUserId == dpv.UserId && cs.Has(schema.User_USER_READ_SELF):
	default:
		dpv.UserId = schema.AnonymousUserID
	}
	return &dpv
}
