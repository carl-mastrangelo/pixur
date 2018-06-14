package handlers

import (
	"net/http"

	"pixur.org/pixur/api"
	"pixur.org/pixur/fe/server"
	ptpl "pixur.org/pixur/fe/tpl"
)

var commentDisplayTpl = parseTpl(ptpl.Base, ptpl.Pane, ptpl.Comment, ptpl.CommentReply)

type commentDisplayData struct {
	*paneData
	Pic        *api.Pic
	PicComment *picComment
	// CommentText is the initial comment after a failed write
	CommentText string
}

type commentDisplayHandler struct {
	pt *paths
	c  api.PixurServiceClient
}

func (h *commentDisplayHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	commentParentId := r.FormValue(h.pt.pr.CommentParentId())
	picId := r.FormValue(h.pt.pr.PicId())
	req := &api.LookupPicDetailsRequest{
		PicId: picId,
	}

	details, err := h.c.LookupPicDetails(ctx, req)
	if err != nil {
		httpReadError(ctx, w, err)
		return
	}

	pd := newPaneData(ctx, "Add Comment", h.pt)

	var root *picComment
	if details.PicCommentTree != nil && len(details.PicCommentTree.Comment) > 0 {
		m := make(map[string][]*picComment)
		for _, c := range details.PicCommentTree.Comment {
			pc := &picComment{
				PicComment: c,
				Child:      m[c.CommentId],
				Paths:      pd.Paths,
				XsrfToken:  pd.XsrfToken,
			}
			if pc.CommentId == commentParentId {
				root = pc
				break
			}
			m[c.CommentParentId] = append(m[c.CommentParentId], pc)
		}
	}
	if root == nil {
		httpReadError(ctx, w, &HTTPErr{
			Message: "Can't find comment id",
			Code:    http.StatusBadRequest,
		})
		return
	}

	data := commentDisplayData{
		paneData:    pd,
		Pic:         details.Pic,
		PicComment:  root,
		CommentText: r.PostFormValue(h.pt.pr.CommentText()),
	}
	if err := commentDisplayTpl.Execute(w, data); err != nil {
		httpCleanupError(w, err)
		return
	}
}

type addPicCommentHandler struct {
	pt      *paths
	c       api.PixurServiceClient
	display http.Handler
}

func (h *addPicCommentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req := &api.AddPicCommentRequest{
		PicId:           r.PostFormValue(h.pt.pr.PicId()),
		CommentParentId: r.PostFormValue(h.pt.pr.CommentParentId()),
		Text:            r.PostFormValue(h.pt.pr.CommentText()),
	}
	ctx := r.Context()

	res, err := h.c.AddPicComment(ctx, req)
	if err != nil {
		httpWriteError(w, err)
		ctx = ctxFromWriteErr(ctx, err)
		r = r.WithContext(ctx)
		h.display.ServeHTTP(w, r)
		return
	}

	http.Redirect(w, r, h.pt.ViewerComment(res.Comment.PicId, res.Comment.CommentId).String(), http.StatusSeeOther)
}

func init() {
	register(func(s *server.Server) error {
		pt := &paths{r: s.HTTPRoot}
		cdh := readWrapper(s)(&commentDisplayHandler{
			c:  s.Client,
			pt: pt,
		})
		apch := writeWrapper(s)(&addPicCommentHandler{
			c:       s.Client,
			pt:      pt,
			display: cdh,
		})
		h := compressHtmlHandler(&methodHandler{
			Get:  cdh,
			Post: apch,
		})
		s.HTTPMux.Handle(pt.Comment().Path, h)

		return nil
	})
}
