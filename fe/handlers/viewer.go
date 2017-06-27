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

type Params struct{}

func (p Params) Vote() string {
	return "vote"
}

func (p Params) PicId() string {
	return "pic_id"
}

func (p Params) Next() string {
	return "next"
}

var viewerTpl = template.Must(template.ParseFiles("tpl/base.html", "tpl/viewer.html"))

type viewerHandler struct {
	p Paths
	c api.PixurServiceClient
}

type viewerData struct {
	baseData
	Paths
	Params
	Pic *api.Pic
}

func (h *viewerHandler) static(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, h.p.ViewerDir().RequestURI())
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
	data := viewerData{
		baseData: baseData{
			Title:     "",
			XsrfName:  xsrfFieldName,
			XsrfToken: xsrfToken,
		},
		Paths: h.p,
		Pic:   details.Pic,
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
	postedVote := r.PostFormValue((Params{}).Vote())
	mappedVote := api.UpsertPicVoteRequest_Vote(api.UpsertPicVoteRequest_Vote_value[postedVote])

	next := r.PostFormValue((Params{}).Next())
	nextURL, err := url.Parse(next)
	if err != nil {
		httpError(w, err)
		return
	}
	nextURL.Scheme = ""
	nextURL.Opaque = ""
	nextURL.User = nil
	nextURL.Host = ""

	req := &api.UpsertPicVoteRequest{
		PicId: r.PostFormValue((Params{}).PicId()),
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

func init() {
	register(func(s *server.Server) error {
		bh := newBaseHandler(s)
		h := viewerHandler{
			c: s.Client,
			p: Paths{s.HTTPRoot},
		}
		s.HTTPMux.Handle(h.p.VoteAction().RequestURI(), bh.action(http.HandlerFunc(h.vote)))
		s.HTTPMux.Handle(h.p.ViewerDir().RequestURI(), bh.static(http.HandlerFunc(h.static)))
		return nil
	})
}
