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
	PicId int64
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

	now := t.Now()
	j, u, sts := authedJob(ctx, t.Beg, now)
	if sts != nil {
		return sts
	}
	defer revert(j, &stscap)

	conf, sts := GetConfiguration(ctx)
	if sts != nil {
		return sts
	}
	if sts := validateCapability(u, conf, schema.User_PIC_VOTE_CREATE); sts != nil {
		return sts
	}
	userId := schema.AnonymousUserId
	picVoteIndex := int64(0)
	if u != nil {
		userId = u.UserId
	}

	if len(t.Ext) != 0 {
		if sts := validateCapability(u, conf, schema.User_PIC_VOTE_EXTENSION_CREATE); sts != nil {
			return sts
		}
	}

	pics, err := j.FindPics(db.Opts{
		Prefix: tab.PicsPrimary{&t.PicId},
		Lock:   db.LockWrite,
	})
	if err != nil {
		return status.Internal(err, "can't lookup pic", t.PicId)
	}
	if len(pics) != 1 {
		return status.NotFound(nil, "can't find pic", t.PicId)
	}
	p := pics[0]

	if p.HardDeleted() {
		return status.InvalidArgument(nil, "can't vote on deleted pic", t.PicId)
	}

	pvs, err := j.FindPicVotes(db.Opts{
		Prefix: tab.PicVotesPrimary{
			PicId:  &t.PicId,
			UserId: &userId,
		},
		Lock: db.LockWrite,
	})
	if err != nil {
		return status.Internal(err, "can't find pic votes")
	}
	if userId != schema.AnonymousUserId {
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

	pv := &schema.PicVote{
		PicId:  p.PicId,
		UserId: userId,
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
		if userId != schema.AnonymousUserId {
			idx, sts := next(userId)
			if sts != nil {
				return sts
			}
			notifications[userId] = true
			oue = &schema.UserEvent{
				UserId:     userId,
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
			if fs.UserId != schema.AnonymousUserId && !notifications[fs.UserId] {
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
							SubjectUserId: userId,
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
	uc := userCredOf(su, conf)
	dpv, _ := filterPicVoteInternal(pv, uc)
	return dpv
}

func filterPicVotes(
	pvs []*schema.PicVote, su *schema.User, conf *schema.Configuration) []*schema.PicVote {
	uc := userCredOf(su, conf)
	dst := make([]*schema.PicVote, 0, len(pvs))
	for _, pv := range pvs {
		if dpv, uf := filterPicVoteInternal(pv, uc); !uf {
			dst = append(dst, dpv)
		}
	}
	return dst
}

func filterPicVoteInternal(pv *schema.PicVote, uc *userCred) (
	filteredVote *schema.PicVote, userFiltered bool) {
	dpv := *pv
	if !uc.cs.Has(schema.User_PIC_VOTE_EXTENSION_READ) {
		dpv.Ext = nil
	}
	uf := false
	switch {
	case uc.cs.Has(schema.User_USER_READ_ALL):
	case uc.cs.Has(schema.User_USER_READ_PUBLIC) && uc.cs.Has(schema.User_USER_READ_PIC_VOTE):
	case uc.subjectUserId == dpv.UserId && uc.cs.Has(schema.User_USER_READ_SELF):
	default:
		uf = true
		dpv.UserId = schema.AnonymousUserId
	}
	return &dpv, uf
}
