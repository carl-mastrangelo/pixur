package handlers

import (
	"html/template"
	"net/http"
	"strings"

	"golang.org/x/sync/errgroup"

	"pixur.org/pixur/api"
	"pixur.org/pixur/fe/server"
)

var viewerTpl = template.Must(template.ParseFiles("tpl/base.html", "tpl/viewer.html"))

func (p Paths) ViewerDir() string {
	return p.Root() + "p/"
}

func (p Paths) Viewer(id string) string {
	return p.ViewerDir() + id
}

type viewerHandler struct {
	Paths
	c api.PixurServiceClient
}

type viewerData struct {
	baseData
	Paths
	Pic *api.Pic
}

func (h *viewerHandler) static(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, h.ViewerDir())
	req := &api.LookupPicDetailsRequest{
		PicId: id,
	}
	ctx := r.Context()
	details, err := h.c.LookupPicDetails(ctx, req)
	if err != nil {
		httpError(w, err)
		return
	}
	data := viewerData{
		Paths: h.Paths,
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

}

func init() {
	register(func(s *server.Server) error {
		bh := newBaseHandler(s)
		h := viewerHandler{
			c: s.Client,
		}

		s.HTTPMux.Handle(defaultPaths.ViewerDir(), bh.static(http.HandlerFunc(h.static)))
		return nil
	})
}
