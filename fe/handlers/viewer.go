package handlers

import (
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes"
	"golang.org/x/sync/errgroup"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"

	"pixur.org/pixur/api"
	"pixur.org/pixur/fe/server"
	ptpl "pixur.org/pixur/fe/tpl"
)

var viewerTpl = parseTpl(ptpl.Base, ptpl.Pane, ptpl.Viewer, ptpl.CommentReply)

type viewerHandler struct {
	pt *paths
	c  api.PixurServiceClient
}

type picComment struct {
	*api.PicComment
	UserId, Ident string
	Child         []*picComment
	Paths         *paths
	XsrfToken     string
	// CommentText is the initial comment after a failed write
	CommentText string
}

func (pc *picComment) CreatedTime() time.Time {
	ts, err := ptypes.Timestamp(pc.PicComment.CreatedTime)
	if err != nil {
		panic(err)
	}
	return ts
}

type viewerDataDeletionReason struct {
	Name  string
	Value int32
}

type viewerData struct {
	*paneData
	Pic            *api.Pic
	PicComment     *picComment
	PicVote        *api.PicVote
	PicTag         []*api.PicTag
	Derived        []*api.PicFile
	DeletionReason []viewerDataDeletionReason
}

var _ collate.Lister = (*picTagsSortable)(nil)

type picTagsSortable []*api.PicTag

func (pts picTagsSortable) Len() int {
	return len(pts)
}

func (pts picTagsSortable) Swap(i, k int) {
	pts[i], pts[k] = pts[k], pts[i]
}

func (pts picTagsSortable) Bytes(i int) []byte {
	return []byte(pts[i].Name)
}

func (h *viewerHandler) static(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if canViewPic := maybeHasCap(ctx, api.Capability_PIC_INDEX); !canViewPic {
		http.Redirect(w, r, h.pt.Login().String(), http.StatusSeeOther)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, h.pt.ViewerDir().Path)
	req := &api.LookupPicDetailsRequest{
		PicId: id,
	}

	details, err := h.c.LookupPicDetails(ctx, req)
	if err != nil {
		httpReadError(ctx, w, err)
		return
	}

	var puismu sync.Mutex
	puis := make(map[string]*api.PublicUserInfo)
	if hasCap(ctx, api.Capability_USER_READ_PUBLIC) || hasCap(ctx, api.Capability_USER_READ_ALL) {
		eg, egctx := errgroup.WithContext(ctx)
		for _, pc := range details.PicCommentTree.Comment {
			if pc.UserId != nil {
				uid := pc.UserId.Value
				eg.Go(func() error {
					resp, err := h.c.LookupPublicUserInfo(egctx, &api.LookupPublicUserInfoRequest{
						UserId: uid,
					})
					if err != nil {
						return err
					}
					puismu.Lock()
					defer puismu.Unlock()
					puis[resp.UserInfo.UserId] = resp.UserInfo
					return nil
				})
			}
		}

		if err := eg.Wait(); err != nil {
			// discard the error, since it's probably not fatal
			puis = nil
		}
	}

	u := subjectUserOrNilFromCtx(ctx)
	var pv *api.PicVote
	if u != nil {
		resp, err := h.c.LookupPicVote(ctx, &api.LookupPicVoteRequest{
			PicId: id,
		})
		if err != nil {
			httpReadError(ctx, w, err)
			return
		}
		pv = resp.Vote
	}
	pd := newPaneData(ctx, "Viewer", h.pt)

	root := &picComment{
		XsrfToken: pd.XsrfToken,
		Paths:     pd.Paths,
		PicComment: &api.PicComment{
			PicId: id,
		},
	}
	if details.PicCommentTree != nil && len(details.PicCommentTree.Comment) > 0 {
		m := make(map[string][]*picComment)
		for _, c := range details.PicCommentTree.Comment {
			var userId string
			var ident string
			if c.UserId != nil {
				userId = puis[c.UserId.Value].UserId
				ident = puis[c.UserId.Value].Ident
			}
			m[c.CommentParentId] = append(m[c.CommentParentId], &picComment{
				PicComment: c,
				UserId:     userId,
				Ident:      ident,
				Child:      m[c.CommentId],
				XsrfToken:  pd.XsrfToken,
				Paths:      pd.Paths,
			})
		}
		root.Child = m["0"]
	}
	var timesort func(children []*picComment)
	timesort = func(children []*picComment) {
		if len(children) == 0 {
			return
		}
		sort.Slice(children, func(i, k int) bool {
			itime, ktime := children[i].CreatedTime(), children[k].CreatedTime()
			if itime.Before(ktime) {
				return true
			} else if itime.After(ktime) {
				return false
			} else {
				return children[i].PicComment.CommentId < children[k].PicComment.CommentId
			}
		})
		for _, c := range children {
			timesort(c.Child)
		}
	}
	timesort(root.Child)

	pts := picTagsSortable(details.PicTag)
	collate.New(language.English, collate.Loose).Sort(pts)

	data := viewerData{
		paneData:   pd,
		Pic:        details.Pic,
		Derived:    details.Derived,
		PicComment: root,
		PicTag:     ([]*api.PicTag)(pts),
		PicVote:    pv,
	}
	for k, v := range api.DeletionReason_name {
		if k == int32(api.DeletionReason_UNKNOWN) {
			continue
		}
		data.DeletionReason = append(data.DeletionReason, viewerDataDeletionReason{
			Name:  v,
			Value: k,
		})
	}

	if err := viewerTpl.Execute(w, data); err != nil {
		httpCleanupError(w, err)
		return
	}
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	// This happens after
	eg, egctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		_, err := h.c.IncrementViewCount(egctx, &api.IncrementViewCountRequest{
			PicId: id,
		})
		return err
	})
	if err := eg.Wait(); err != nil {
		httpCleanupError(w, err)
		return
	}
}

func (h *viewerHandler) vote(w http.ResponseWriter, r *http.Request) {
	postedVote := r.PostFormValue(h.pt.pr.Vote())
	mappedVote := api.PicVote_Vote(api.PicVote_Vote_value[postedVote])

	next := r.PostFormValue(h.pt.pr.Next())
	nextURL, err := url.Parse(next)
	if err != nil {
		httpError(w, err)
		return
	}

	req := &api.UpsertPicVoteRequest{
		PicId: r.PostFormValue(h.pt.pr.PicId()),
		Vote:  mappedVote,
	}

	ctx := r.Context()

	_, err = h.c.UpsertPicVote(ctx, req)
	if err != nil {
		httpError(w, err)
		return
	}

	http.Redirect(w, r, nextURL.String(), http.StatusSeeOther)
}

// stored here until there is some sort of admin panel
func (h *viewerHandler) softdelete(w http.ResponseWriter, r *http.Request) {
	if r.PostFormValue(h.pt.pr.DeletePicReally()) == "" {
		httpError(w, &HTTPErr{
			Message: "\"Really\" box not checked",
			Code:    http.StatusBadRequest,
		})
		return
	}

	rawDeletionReason := r.PostFormValue(h.pt.pr.DeletePicReason())
	deletionReasonNum, err := strconv.ParseInt(rawDeletionReason, 10, 32)
	if err != nil {
		httpError(w, err)
		return
	}

	req := &api.SoftDeletePicRequest{
		PicId:        r.PostFormValue(h.pt.pr.PicId()),
		Details:      r.PostFormValue(h.pt.pr.DeletePicDetails()),
		Reason:       api.DeletionReason(deletionReasonNum),
		DeletionTime: nil,
	}

	ctx := r.Context()
	_, err = h.c.SoftDeletePic(ctx, req)
	if err != nil {
		httpError(w, err)
		return
	}

	http.Redirect(w, r, h.pt.Viewer(req.PicId).String(), http.StatusSeeOther)
}

func init() {
	register(func(s *server.Server) error {
		h := viewerHandler{
			c:  s.Client,
			pt: &paths{r: s.HTTPRoot},
		}
		s.HTTPMux.Handle(h.pt.VoteAction().Path, newActionHandler(s, http.HandlerFunc(h.vote)))
		// static is initialized in root.go
		s.HTTPMux.Handle(h.pt.SoftDeletePicAction().Path, newActionHandler(s, http.HandlerFunc(h.softdelete)))
		return nil
	})
}
