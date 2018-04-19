package handlers

import (
	"net/http"

	"pixur.org/pixur/api"
	"pixur.org/pixur/fe/server"
	ptpl "pixur.org/pixur/fe/tpl"
)

var commentTpl = parseTpl(ptpl.Base, ptpl.Pane, ptpl.Comment, ptpl.CommentReply)

type commentHandler struct {
	pt paths
	c  api.PixurServiceClient
}

type commentData struct {
	baseData
	Pic        *api.Pic
	PicComment *picComment
}

func (h *commentHandler) get(w http.ResponseWriter, r *http.Request) {
	commentParentId := r.FormValue(h.pt.pr.CommentParentId())
	picId := r.FormValue(h.pt.pr.PicId())
	req := &api.LookupPicDetailsRequest{
		PicId: picId,
	}
	ctx := r.Context()
	details, err := h.c.LookupPicDetails(ctx, req)
	if err != nil {
		httpError(w, err)
		return
	}

	bd := baseData{
		Paths:       h.pt,
		XsrfToken:   outgoingXsrfTokenOrEmptyFromCtx(ctx),
		SubjectUser: subjectUserOrNilFromCtx(ctx),
	}

	var root *picComment
	if details.PicCommentTree != nil && len(details.PicCommentTree.Comment) > 0 {
		m := make(map[string][]*picComment)
		for _, c := range details.PicCommentTree.Comment {
			pc := &picComment{
				PicComment: c,
				Child:      m[c.CommentId],
				baseData:   bd,
			}
			if pc.CommentId == commentParentId {
				root = pc
				break
			}
			m[c.CommentParentId] = append(m[c.CommentParentId], pc)
		}
	}
	if root == nil {
		httpError(w, &HTTPErr{
			Message: "Can't find comment id",
			Code:    http.StatusBadRequest,
		})
		return
	}

	data := commentData{
		baseData:   bd,
		Pic:        details.Pic,
		PicComment: root,
	}
	if err := commentTpl.Execute(w, data); err != nil {
		httpError(w, err)
		return
	}
}

type addPicCommentHandler struct {
	pt          paths
	c           api.PixurServiceClient
	readHandler http.Handler
}

func (h *commentHandler) comment(w http.ResponseWriter, r *http.Request) {
	req := &api.AddPicCommentRequest{
		PicId:           r.PostFormValue(h.pt.pr.PicId()),
		CommentParentId: r.PostFormValue(h.pt.pr.CommentParentId()),
		Text:            r.PostFormValue(h.pt.pr.CommentText()),
	}

	res, err := h.c.AddPicComment(r.Context(), req)
	if err != nil {
		httpError(w, err)
		return
	}

	http.Redirect(w, r, h.pt.ViewerComment(res.Comment.PicId, res.Comment.CommentId).RequestURI(), http.StatusSeeOther)
}

func init() {
	register(func(s *server.Server) error {
		ch := &commentHandler{
			c:  s.Client,
			pt: paths{r: s.HTTPRoot},
		}
		get := newReadHandler(s, http.HandlerFunc(ch.get))
		post := newActionHandler(s, http.HandlerFunc(ch.comment))

		// todo: remove CommentAction
		h := &compressionHandler{
			next: &htmlHandler{
				next: &methodHandler{
					Get:  get,
					Post: post,
				},
			},
		}
		s.HTTPMux.Handle(ch.pt.Comment().Path, h)

		return nil
	})
}
