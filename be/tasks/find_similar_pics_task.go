package tasks

import (
	"context"
	"encoding/binary"
	"time"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
)

// TODO: add tests

type FindSimilarPicsTask struct {
	// Deps
	Beg tab.JobBeginner
	Now func() time.Time

	// Inputs
	PicId int64

	// Results
	SimilarPicIds []int64
}

func (t *FindSimilarPicsTask) Run(ctx context.Context) (stscap status.S) {
	now := t.Now()
	j, u, sts := authedReadonlyJob(ctx, t.Beg, now)
	if sts != nil {
		return sts
	}
	defer revert(j, &stscap)

	conf, sts := GetConfiguration(ctx)
	if sts != nil {
		return sts
	}
	if sts := validateCapability(u, conf, schema.User_PIC_INDEX); sts != nil {
		return sts
	}

	pics, err := j.FindPics(db.Opts{
		Prefix: tab.PicsPrimary{&t.PicId},
		Limit:  1,
	})
	if err != nil {
		return status.Internal(err, "can't lookup pic")
	}
	if len(pics) != 1 {
		return status.NotFound(nil, "can't lookup pic", len(pics))
	}
	pic := pics[0]

	dctIdentType := schema.PicIdent_DCT_0

	picIdents, err := j.FindPicIdents(db.Opts{
		Prefix: tab.PicIdentsPrimary{PicId: &t.PicId, Type: &dctIdentType},
		Limit:  1,
	})
	if err != nil {
		return status.Internal(err, "can't lookup pic ident")
	}
	if len(picIdents) != 1 {
		return status.InvalidArgument(nil, "can't lookup pic ident", len(picIdents))
	}
	targetIdent := picIdents[0]
	match := binary.BigEndian.Uint64(targetIdent.Value)

	scanOpts := db.Opts{
		StartInc: tab.PicIdentsIdent{Type: &dctIdentType},
	}
	var similarPicIds []int64

	err = j.ScanPicIdents(scanOpts, func(pi *schema.PicIdent) error {
		if pi.PicId == pic.PicId {
			return nil
		}
		guess := binary.BigEndian.Uint64(pi.Value)
		bits := guess ^ match
		bitCount := 0
		// replace this with something that isn't hideously slow.  Hamming distance would be
		// better served by a look up table or some 64 bit specific bit magic.  Cosine similarity
		// on the attached floats would also work.
		for i := uint(0); i < 64; i++ {
			if ((1 << i) & bits) > 0 {
				bitCount++
			}
		}
		if bitCount <= 10 {
			similarPicIds = append(similarPicIds, pi.PicId)
		}

		return nil
	})

	if err != nil {
		return status.Internal(err, "can't scan pic idents")
	}
	if err := j.Rollback(); err != nil {
		return status.Internal(err, "can't rollback job")
	}
	// Only set results on success
	t.SimilarPicIds = similarPicIds

	return nil
}
