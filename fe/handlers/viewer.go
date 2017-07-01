package handlers

import (
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/sync/errgroup"

	"pixur.org/pixur/api"
	"pixur.org/pixur/fe/server"
)

var viewerTpl = template.Must(template.Must(rootTpl.Clone()).ParseFiles("tpl/viewer.html", "tpl/comment_reply.html"))

type viewerHandler struct {
	pt paths
	c  api.PixurServiceClient
}

type picComment struct {
	*api.PicComment
	Child []*picComment
	baseData
}

type viewerData struct {
	baseData
	Pic        *api.Pic
	PicComment *picComment
}

func (h *viewerHandler) static(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, h.pt.ViewerDir().RequestURI())
	req := &api.LookupPicDetailsRequest{
		PicId: id,
	}
	ctx := r.Context()
	details, err := h.c.LookupPicDetails(ctx, req)
	if err != nil {
		httpError(w, err)
		return
	}

	xsrfToken, _ := xsrfTokenFromContext(ctx)
	bd := baseData{
		Paths:     h.pt,
		Params:    h.pt.pr,
		XsrfToken: xsrfToken,
	}

	root := &picComment{
		baseData: bd,
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
				baseData:   bd,
			})
		}
		root.Child = m["0"]
	}

	data := viewerData{
		baseData:   bd,
		Pic:        details.Pic,
		PicComment: root,
	}
	if err := viewerTpl.Execute(w, data); err != nil {
		httpError(w, err)
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
		httpError(w, err)
		return
	}
}

func (h *viewerHandler) vote(w http.ResponseWriter, r *http.Request) {
	postedVote := r.PostFormValue(h.pt.pr.Vote())
	mappedVote := api.UpsertPicVoteRequest_Vote(api.UpsertPicVoteRequest_Vote_value[postedVote])

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

func init() {
	register(func(s *server.Server) error {
		bh := newBaseHandler(s)
		h := viewerHandler{
			c:  s.Client,
			pt: paths{r: s.HTTPRoot},
		}
		s.HTTPMux.Handle(h.pt.VoteAction().RequestURI(), bh.action(http.HandlerFunc(h.vote)))
		// static is initialized in root.go
		return nil
	})
}
