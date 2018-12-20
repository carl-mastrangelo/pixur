package handlers

import (
	"html/template"
	"net/http"

	"pixur.org/pixur/api"
	ptpl "pixur.org/pixur/fe/tpl"
)

type indexData struct {
	*paneData

	Pic []*api.PicAndThumbnail

	NextID, PrevID string

	CanUpload bool
}

var indexTpl = parseTpl(ptpl.Base, ptpl.Pane, ptpl.Index)

func parseTpl(tpls ...string) *template.Template {
	t := template.New("NamesAreADumbIdeaForTemplates").Option("missingkey=error")
	for i := len(tpls) - 1; i >= 0; i-- {
		template.Must(t.Parse(tpls[i]))
	}
	return t
}

type indexHandler struct {
	c  api.PixurServiceClient
	pt *paths
}

func (h *indexHandler) static(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := r.ParseForm(); err != nil {
		httpReadError(ctx, w, &HTTPErr{
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	id := r.FormValue(h.pt.pr.IndexPic())
	_, isPrev := r.Form[h.pt.pr.IndexPrev()]
	req := &api.FindIndexPicsRequest{
		StartPicId: id,
		Ascending:  isPrev,
	}

	if canViewIndex := maybeHasCap(ctx, api.Capability_PIC_INDEX); !canViewIndex {
		http.Redirect(w, r, h.pt.Login().String(), http.StatusSeeOther)
		return
	}

	res, err := h.c.FindIndexPics(ctx, req)
	if err != nil {
		httpReadError(ctx, w, err)
		return
	}
	var prevID string
	var nextID string
	if !isPrev {
		nextID = res.NextPicId
		prevID = res.PrevPicId
	} else {
		nextID = res.PrevPicId
		prevID = res.NextPicId
	}

	if isPrev {
		for i := 0; i < len(res.Pic)/2; i++ {
			res.Pic[i], res.Pic[len(res.Pic)-i-1] = res.Pic[len(res.Pic)-i-1], res.Pic[i]
		}
	}

	canupload := hasCap(ctx, api.Capability_PIC_CREATE)

	data := indexData{
		paneData:  newPaneData(ctx, "Index", h.pt),
		Pic:       res.Pic,
		NextID:    nextID,
		PrevID:    prevID,
		CanUpload: canupload,
	}
	if err := indexTpl.Execute(w, data); err != nil {
		httpCleanupError(w, err)
		return
	}
}
