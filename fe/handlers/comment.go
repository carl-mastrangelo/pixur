package handlers

import (
	"html/template"
	"net/http"

	"pixur.org/pixur/api"
	"pixur.org/pixur/fe/server"
	ptpl "pixur.org/pixur/fe/tpl"
)

var commentTpl *template.Template

func init() {
	paneClone := template.Must(paneTpl.Clone())
	commentTpl = paneClone.New("Comment")
	commentTpl = template.Must(commentTpl.Parse(ptpl.Comment))
	commentTpl = template.Must(commentTpl.Parse(ptpl.CommentReply))
}

type commentHandler struct {
	pt paths
	c  api.PixurServiceClient
}

type commentData struct {
	baseData
	Pic        *api.Pic
	PicComment *picComment
}

func (h *commentHandler) static(w http.ResponseWriter, r *http.Request) {
	commentParentId := r.FormValue(h.pt.pr.CommentParentId())
	req := &api.LookupPicDetailsRequest{
		PicId: r.FormValue(h.pt.pr.PicId()),
	}
	ctx := r.Context()
	details, err := h.c.LookupPicDetails(ctx, req)
	if err != nil {
		httpError(w, err)
		return
	}

	xsrfToken, _ := xsrfTokenFromContext(ctx)
	bd := baseData{
		Paths:       h.pt,
		XsrfToken:   xsrfToken,
		SubjectUser: subjectUserOrNilFromCtx(r.Context()),
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

func (h *commentHandler) comment(w http.ResponseWriter, r *http.Request) {
	var pr params

	req := &api.AddPicCommentRequest{
		PicId:           r.PostFormValue(pr.PicId()),
		CommentParentId: r.PostFormValue(pr.CommentParentId()),
		Text:            r.PostFormValue(pr.CommentText()),
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
		bh := newBaseHandler(s)
		h := commentHandler{
			c:  s.Client,
			pt: paths{r: s.HTTPRoot},
		}
		s.HTTPMux.Handle(h.pt.Comment().RequestURI(), bh.static(http.HandlerFunc(h.static)))
		s.HTTPMux.Handle(h.pt.CommentAction().RequestURI(), bh.action(http.HandlerFunc(h.comment)))
		// static is initialized in root.go
		return nil
	})
}
