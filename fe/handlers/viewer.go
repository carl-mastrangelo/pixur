package handlers

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"

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
	Child     []*picComment
	Paths     *paths
	XsrfToken string
	// CommentText is the initial comment after a failed write
	CommentText string
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
	DeletionReason []viewerDataDeletionReason
}

func (h *viewerHandler) static(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, h.pt.ViewerDir().RequestURI())
	req := &api.LookupPicDetailsRequest{
		PicId: id,
	}
	ctx := r.Context()
	details, err := h.c.LookupPicDetails(ctx, req)
	if err != nil {
		httpReadError(ctx, w, err)
		return
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
			m[c.CommentParentId] = append(m[c.CommentParentId], &picComment{
				PicComment: c,
				Child:      m[c.CommentId],
				XsrfToken:  pd.XsrfToken,
				Paths:      pd.Paths,
			})
		}
		root.Child = m["0"]
	}

	data := viewerData{
		paneData:   pd,
		Pic:        details.Pic,
		PicComment: root,
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

	http.Redirect(w, r, nextURL.RequestURI(), http.StatusSeeOther)
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

	http.Redirect(w, r, h.pt.Viewer(req.PicId).RequestURI(), http.StatusSeeOther)
}

func init() {
	register(func(s *server.Server) error {
		h := viewerHandler{
			c:  s.Client,
			pt: &paths{r: s.HTTPRoot},
		}
		s.HTTPMux.Handle(h.pt.VoteAction().RequestURI(), newActionHandler(s, http.HandlerFunc(h.vote)))
		// static is initialized in root.go
		s.HTTPMux.Handle(h.pt.SoftDeletePicAction().RequestURI(), newActionHandler(s, http.HandlerFunc(h.softdelete)))
		return nil
	})
}
