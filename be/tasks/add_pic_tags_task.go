package tasks

import (
	"context"
	"math"
	"strings"
	"time"

	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/be/text"
)

type AddPicTagsTask struct {
	// Deps
	Beg tab.JobBeginner
	Now func() time.Time

	// Inputs
	PicID    int64
	TagNames []string
}

// TODO: add tests
func (t *AddPicTagsTask) Run(ctx context.Context) (stscap status.S) {
	j, err := tab.NewJob(ctx, t.Beg)
	if err != nil {
		return status.Internal(err, "can't create job")
	}
	defer revert(j, &stscap)

	u, sts := requireCapability(ctx, j, schema.User_PIC_TAG_CREATE)
	if sts != nil {
		return sts
	}

	pics, err := j.FindPics(db.Opts{
		Prefix: tab.PicsPrimary{&t.PicID},
		Lock:   db.LockWrite,
	})
	if err != nil {
		return status.Internal(err, "can't lookup pic")
	}
	if len(pics) != 1 {
		return status.NotFound(nil, "can't find pic")
	}
	p := pics[0]

	if p.HardDeleted() {
		return status.InvalidArgument(nil, "can't tag deleted pic")
	}

	conf, sts := GetConfiguration(ctx)
	if sts != nil {
		return sts
	}

	var minTagLen, maxTagLen int64
	if conf.MinTagLength != nil {
		minTagLen = conf.MinTagLength.Value
	} else {
		minTagLen = math.MinInt64
	}
	if conf.MaxTagLength != nil {
		maxTagLen = conf.MaxTagLength.Value
	} else {
		maxTagLen = math.MaxInt64
	}

	if sts := upsertTags(j, t.TagNames, p.PicId, t.Now(), u.UserId, minTagLen, maxTagLen); sts != nil {
		return sts
	}

	if err := j.Commit(); err != nil {
		return status.Internal(err, "can't commit job")
	}
	return nil
}

type tagNameAndUniq struct {
	name, uniq string
}

func upsertTags(j *tab.Job, rawTags []string, picID int64, now time.Time,
	userID, minTagLen, maxTagLen int64) status.S {
	newTagNames, sts := cleanTagNames(rawTags, minTagLen, maxTagLen)
	if sts != nil {
		return sts
	}

	attachedTags, _, sts := findAttachedPicTags(j, picID)
	if sts != nil {
		return sts
	}

	unattachedTagNames := findUnattachedTagNames(attachedTags, newTagNames)
	unattachedExistingTags, unknownNames, sts := findExistingTagsByName(j, unattachedTagNames)
	if sts != nil {
		return sts
	}

	if sts := updateExistingTags(j, unattachedExistingTags, now); sts != nil {
		return sts
	}
	newTags, sts := createNewTags(j, unknownNames, now)
	if sts != nil {
		return sts
	}

	unattachedExistingTags = append(unattachedExistingTags, newTags...)
	if _, sts := createPicTags(j, unattachedExistingTags, picID, now, userID); sts != nil {
		return sts
	}

	return nil
}

func findAttachedPicTags(j *tab.Job, picID int64) ([]*schema.Tag, []*schema.PicTag, status.S) {
	pts, err := j.FindPicTags(db.Opts{
		Prefix: tab.PicTagsPrimary{PicId: &picID},
		Lock:   db.LockWrite,
	})
	if err != nil {
		return nil, nil, status.Internal(err, "cant't find pic tags")
	}

	var tags []*schema.Tag
	// TODO: maybe do something with lock ordering?
	for _, pt := range pts {
		ts, err := j.FindTags(db.Opts{
			Prefix: tab.TagsPrimary{&pt.TagId},
			Limit:  1,
			Lock:   db.LockWrite,
		})
		if err != nil {
			return nil, nil, status.Internal(err, "can't find tags")
		}
		if len(ts) != 1 {
			return nil, nil, status.Internal(nil, "can't lookup tag", len(ts))
		}
		tags = append(tags, ts[0])
	}
	return tags, pts, nil
}

// findUnattachedTagNames finds tag names that are not part of a pic's tags.
// While pic tags are the SoT for attachment, only the Tag is the SoT for the name.
func findUnattachedTagNames(attachedTags []*schema.Tag, newTagNames []tagNameAndUniq) []tagNameAndUniq {
	attachedTagNames := make(map[string]struct{}, len(attachedTags))

	for _, tag := range attachedTags {
		attachedTagNames[schema.TagUniqueName(tag.Name)] = struct{}{}
	}
	var unattachedTagNames []tagNameAndUniq
	for _, newTagName := range newTagNames {
		if _, attached := attachedTagNames[newTagName.uniq]; !attached {
			unattachedTagNames = append(unattachedTagNames, newTagName)
		}
	}

	return unattachedTagNames
}

func findExistingTagsByName(j *tab.Job, names []tagNameAndUniq) (
	tags []*schema.Tag, unknownNames []tagNameAndUniq, _ status.S) {
	for _, name := range names {
		ts, err := j.FindTags(db.Opts{
			Prefix: tab.TagsName{&name.uniq},
			Limit:  1,
			Lock:   db.LockWrite,
		})
		if err != nil {
			return nil, nil, status.Internal(err, "can't find tags")
		}
		if len(ts) == 1 {
			tags = append(tags, ts[0])
		} else {
			unknownNames = append(unknownNames, name)
		}
	}

	return
}

func updateExistingTags(j *tab.Job, tags []*schema.Tag, now time.Time) status.S {
	for _, tag := range tags {
		tag.SetModifiedTime(now)
		tag.UsageCount++
		if err := j.UpdateTag(tag); err != nil {
			return status.Internal(err, "can't update tag")
		}
	}
	return nil
}

func createNewTags(j *tab.Job, names []tagNameAndUniq, now time.Time) ([]*schema.Tag, status.S) {
	var tags []*schema.Tag
	for _, name := range names {
		tagID, err := j.AllocID()
		if err != nil {
			return nil, status.Internal(err, "can't allocate id")
		}
		tag := &schema.Tag{
			TagId:      tagID,
			Name:       name.name,
			UsageCount: 1,
		}
		tag.SetCreatedTime(now)
		tag.SetModifiedTime(now)
		if err := j.InsertTag(tag); err != nil {
			return nil, status.Internal(err, "can't create tag")
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func createPicTags(j *tab.Job, tags []*schema.Tag, picID int64, now time.Time, userID int64) (
	[]*schema.PicTag, status.S) {
	var picTags []*schema.PicTag
	for _, tag := range tags {
		pt := &schema.PicTag{
			PicId:  picID,
			TagId:  tag.TagId,
			Name:   tag.Name,
			UserId: userID,
		}
		pt.SetCreatedTime(now)
		pt.SetModifiedTime(now)
		if err := j.InsertPicTag(pt); err != nil {
			return nil, status.Internal(err, "can't create pic tag")
		}
		picTags = append(picTags, pt)
	}
	return picTags, nil
}

func cleanTagNames(rawTagNames []string, minTagLen, maxTagLen int64) ([]tagNameAndUniq, status.S) {
	validtagnames := make([]tagNameAndUniq, 0, len(rawTagNames))
	defaultValidator := text.DefaultValidator(minTagLen, maxTagLen)
	var norm text.TextNormalizer = func(txt, fieldname string) (string, error) {
		if newtext, err := text.ToNFC(txt, fieldname); err != nil {
			return "", err
		} else {
			return strings.TrimSpace(newtext), nil
		}
	}
	for _, rawtagname := range rawTagNames {
		trimmednormaltagname, err :=
			text.ValidateAndNormalize(rawtagname, "tag", norm, defaultValidator, text.ValidateNoNewlines)
		if err != nil {
			return nil, status.From(err)
		}
		validtagnames = append(validtagnames, tagNameAndUniq{
			name: trimmednormaltagname,
			uniq: schema.TagUniqueName(trimmednormaltagname),
		})
	}

	if sts := validateNoDuplicateTags(validtagnames); sts != nil {
		return nil, sts
	}

	return validtagnames, nil
}

func validateNoDuplicateTags(tagNames []tagNameAndUniq) status.S {
	var seen = make(map[string]int, len(tagNames))
	for i, tn := range tagNames {
		if pos, present := seen[tn.uniq]; present {
			return status.InvalidArgumentf(nil, "duplicate tag '%s' at position %d and %d", tn.name, pos, i)
		}
		seen[tn.uniq] = i
	}
	return nil
}
