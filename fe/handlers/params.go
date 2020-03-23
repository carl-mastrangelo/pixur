package handlers

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	"pixur.org/pixur/api"
)

type params struct{}

func (p params) Vote() string {
	return "vote"
}

func (p params) IndexPic() string {
	return "p"
}

func (p params) IndexPrev() string {
	return "prev"
}

func (p params) Ident() string {
	return "ident"
}

func (p params) Secret() string {
	return "secret"
}

func (p params) Logout() string {
	return "is_logout"
}

func (p params) PicId() string {
	return "pic_id"
}

func (p params) CommentId() string {
	return "comment_id"
}

func (p params) CommentParentId() string {
	return "comment_parent_id"
}

func (p params) UserId() string {
	return "user_id"
}

func (p params) StartUserEventId() string {
	return "start_user_event_id"
}

func (p params) UserEventsAsc() string {
	return "asc"
}

func (p params) Version() string {
	return "version"
}

func (p params) CommentText() string {
	return "text"
}

func (p params) Next() string {
	return "next"
}

func (p params) XsrfCookie() string {
	return "xt"
}

func (p params) Xsrf() string {
	return "x_xt"
}

func (p params) File() string {
	return "file"
}

func (p params) Md5Hash() string {
	return "md5"
}

func (p params) Url() string {
	return "url"
}

func (p params) Tag() string {
	return "tag"
}

func (p params) DeletePicReally() string {
	return "really"
}

func (p params) DeletePicReason() string {
	return "reason"
}

func (p params) DeletePicDetails() string {
	return "details"
}

func (p params) True() string {
	return "t"
}

func (p params) False() string {
	return "f"
}

func (p params) UserCapability(c api.Capability_Cap) string {
	return "cap" + strconv.Itoa(int(c))
}

func (p params) OldUserCapability(c api.Capability_Cap) string {
	return "oldcap" + strconv.Itoa(int(c))
}

func (p params) NewUserCapability(c api.Capability_Cap) string {
	return "newcap" + strconv.Itoa(int(c))
}

func (p params) GetOldUserCapability(vals url.Values) (yes, no []api.Capability_Cap, e error) {
	return p.parseCapsChange("oldcap", vals)
}

func (p params) GetNewUserCapability(vals url.Values) (yes, no []api.Capability_Cap, e error) {
	return p.parseCapsChange("newcap", vals)
}

func (p params) Mime(pf api.PicFile_Format) string {
	// TODO: return some unknown type?
	return picFileFormatMime[pf]
}

func (p params) parseCapsChange(prefix string, vals url.Values) (yes, no []api.Capability_Cap, e error) {
	for k, v := range vals {
		if strings.HasPrefix(k, prefix) {
			num, err := strconv.ParseInt(k[len(prefix):], 10, 32)
			if err != nil {
				return nil, nil, errors.New("can't parse " + k + "(" + err.Error() + ")")
			}
			if _, present := api.Capability_Cap_name[int32(num)]; !present {
				return nil, nil, errors.New("unknown cap " + k)
			}
			if len(v) != 1 {
				return nil, nil, errors.New("bad value(s) for " + k + ": (" + strings.Join(v, ", ") + ")")
			}
			switch v[0] {
			case p.True():
				yes = append(yes, api.Capability_Cap(num))
			case p.False():
				no = append(no, api.Capability_Cap(num))
			default:
				return nil, nil, errors.New("unknown value " + v[0] + " for " + k)
			}
		}
	}
	return yes, no, nil
}
