// Code generated by pixur-gen-tpl.
// DO NOT EDIT!

package tpl

const (
	Base = "<!doctype html>\n<html>\n  <head>\n    <meta charset=\"utf-8\">\n    <meta name=viewport content=\"width=device-width, initial-scale=1\">\n    <title>{{.Title}}</title>\n    <style>\n      body {\n        background-color: #FFFFEE;\n        margin: 0;\n      }\n    </style>\n    {{block \"basestyle\" .}}{{end}}\n  </head>\n  <body>\n    {{template \"body\" .}}\n  </body>\n</html>\n"

	Comment = "{{- define \"pane\" -}}\n{{- template \"commentreply\" . -}}\n{{- end -}}\n"

	CommentReply = "{{define \"commentreply\" }}\n{{if .PicComment.CommentId }}\n<div>\n  {{.PicComment.CommentId}}: {{.PicComment.Text}}\n</div>\n{{end}}\n{{$pt := .Paths}}\n{{$pr := $pt.Params}}\n<form action=\"{{$pt.CommentReply .PicComment.PicId .PicComment.CommentId}}\" method=\"post\">\n  <textarea name=\"{{$pr.CommentText}}\">{{.CommentText}}</textarea>\n  <input type=\"hidden\" name=\"{{$pr.Xsrf}}\" value=\"{{.XsrfToken}}\" />\n  <input type=\"hidden\" name=\"{{$pr.PicId}}\" value=\"{{.PicComment.PicId}}\" />\n  <input type=\"hidden\" name=\"{{$pr.CommentParentId}}\" value=\"{{.PicComment.CommentId}}\" />\n  <input type=\"submit\" value=\"Reply\" />\n</form>\n{{end}}\n"

	Index = "{{define \"panestyle\"}}\n<style>\n  .index {\n    text-align: center;\n  }\n\n  .index ul.thumbnail-list {\n    list-style-type: none;\n    padding: 0;\n  }\n  \n  .index ul.thumbnail-list li {\n    display: inline;\n  }\n  \n  .index .thumbnail-cntr {\n    background-color: #FFFFEE;\n    border-style: solid;\n    border-width: 2px;\n    border-color: #2c1fc0;\n    border-radius: 10px;\n    display: inline-block;\n    height: 192px;\n    margin: 6px;\n    padding: 0;\n    text-align: center;\n    width: 192px;\n  }\n  \n  .index .thumbnail-cntr:hover {\n    border-color: #9c99bf;\n  }\n  \n  .index img.thumbnail {\n    width: 192px;\n    height: 192px;\n    border-radius: 8px;\n  }\n\n  .index img.deleted {\n    filter: blur(5px) grayscale(5%);\n    -webkit-filter: blur(5px) grayscale(5%);\n  }\n  \n  .index .nav-home {\n    text-align: center;\n  }\n  .index .nav-prev {\n    float: left;\n  }\n  .index .nav-next {\n    float: right;\n  }\n  .index .nav:after {\n    clear: both;\n    content: \"\";\n    display: block;\n  }\n</style>\n{{end}}\n{{define \"nav\"}}\n  {{- $pt := .Paths -}}\n  {{- $pr := $pt.Params -}}\n  <div class=\"nav\">\n    {{if .PrevID}}<span class=\"nav-prev\"><a href=\"{{$pt.IndexPrev .PrevID}}\">Previous</a></span>{{end}}\n    {{if .NextID}}<span class=\"nav-next\"><a href=\"{{$pt.Index .NextID}}\">Next</a></span>{{end}}\n  </div>\n{{end}}\n{{define \"pane\"}}\n<div class=\"index\">\n  {{ $pt := .Paths}}\n  {{- $pr := $pt.Params -}}\n  {{- template \"nav\" . -}}\n  {{if .Pic}}\n  <ul class=\"thumbnail-list\">\n    {{- range .Pic -}}\n    <li>{{- /**/ -}}\n      <div class=\"thumbnail-cntr\">{{- /**/ -}}\n        <a href=\"{{$pt.Viewer .Id}}\">{{- /**/ -}}\n          <img {{/**/ -}}\n\t          class=\"thumbnail{{if .PendingDeletion}} deleted{{end}}\" {{/**/ -}}\n\t          src=\"{{$pt.PicFileFirst .Thumbnail}}\" />{{- /**/ -}}\n\t      </a>{{- /**/ -}}\n      </div>{{- /**/ -}}\n    </li>{{- /**/ -}}\n    {{- end -}}\n  </ul>\n  {{end}}\n  {{- template \"nav\" . -}}\n</div>\n{{if .CanUpload}}\n<div style=\"margin-bottom: 2em; margin-top: 2em;\">\n  <fieldset>\n    <legend>Pic Upload</legend>\n    <form action=\"{{$pt.UpsertPicAction}}\" method=\"post\" enctype=\"multipart/form-data\">\n      <input type=\"hidden\" name=\"{{$pr.Xsrf}}\" value=\"{{.XsrfToken}}\" />\n      <dl>\n        <dt style=\"display:inline-block\">File Upload (option 1)</dt>\n        <dd style=\"display:inline-block\"><input type=\"file\" name=\"{{$pr.File}}\" /></dd>\n      </dl>\n      <dl>\n        <dt style=\"display:inline-block\">URL Upload (option 2)</dt>\n        <dd style=\"display:inline-block\"><input placeholder=\"File URL\" name=\"{{$pr.Url}}\" /></dd>\n      </dl>\n      <input type=\"submit\" value=\"Submit\" />\n    </form>\n  </fieldset>\n</div>\n{{end}}\n{{end}}\n"

	Login = "{{define \"pane\"}}\n{{ $pt := .Paths}}\n{{ $pr := $pt.Params}}\n<form action=\"{{$pt.LoginAction}}\" method=\"post\">\n  <input type=\"text\" name=\"{{$pr.Ident}}\" placeholder=\"Ident\" /><br />\n  <input type=\"password\" name=\"{{$pr.Secret}}\" /><br />\n  <input type=\"hidden\" name=\"{{$pr.Xsrf}}\" value=\"{{.XsrfToken}}\" />\n  <input type=\"hidden\" name=\"{{$pr.Next}}\" value=\"{{.Next}}\" />\n  <input type=\"submit\" value=\"Login\" />\n</form>\n<hr>\n<form action=\"{{$pt.CreateUserAction}}\" method=\"post\">\n  <input type=\"text\" name=\"{{$pr.Ident}}\" placeholder=\"Ident\" /><br />\n  <input type=\"password\" name=\"{{$pr.Secret}}\" /><br />\n  <input type=\"hidden\" name=\"{{$pr.Xsrf}}\" value=\"{{.XsrfToken}}\" />\n  <input type=\"hidden\" name=\"{{$pr.Next}}\" value=\"{{.Next}}\" />\n  <input type=\"submit\" value=\"CreateUser\" />\n</form>\n<hr>\n{{/*login action is same as log out*/}}\n<form action=\"{{$pt.LoginAction}}\" method=\"post\">\n  <input type=\"hidden\" name=\"{{$pr.Logout}}\" value=\"non-empty string\" />\n  <input type=\"hidden\" name=\"{{$pr.Xsrf}}\" value=\"{{.XsrfToken}}\" />\n  <input type=\"hidden\" name=\"{{$pr.Next}}\" value=\"{{.Next}}\" />\n  <input type=\"submit\" value=\"Logout\" />\n</form>\n{{end}}\n"

	Pane = "{{block \"basestyle\" .}}\n<style>\n  .pane {\n    margin-left: auto;\n    margin-right: auto;\n    max-width: 950px;\n  }\n  .pane-header {\n    margin-top: 6px;\n    position: relative;\n  }\n  .pane-header a {\n    text-decoration: none;\n  }\n  .pane-user-box {\n    background-color: white;\n    border: 1px solid black;\n    float: right;\n    padding: 4px;\n    position: absolute;\n    right: 0%;\n  }\n  .pane-header .site-name {\n    text-align: center;\n  }\n  .pane-header .site-name h1 {\n    display: inline;\n  }\n  .pane-header .site-name a:link {\n    color: #000;\n  }\n  .pane-header .site-name a:visited {\n    color: #000;\n  }\n  .pane-header .site-name a:hover {\n    color: #000;\n  }\n  .pane .error {\n    color: red;\n    margin: 1em;\n    text-align: center;\n  }\n</style>  \n{{block \"panestyle\" .}}{{end}}\n{{end}}\n{{define \"body\"}}\n{{- $pt := .Paths -}}\n{{- $pr := $pt.Params -}}\n<div class=\"pane\">\n  <div class=\"pane-header\">\n    <div class=\"pane-user-box\">\n      {{if .SubjectUser}}\n        <a href=\"{{\"\" | $pt.UserEdit}}\">{{.SubjectUser.Ident}}</a>\n      {{else}}\n        <a href=\"{{$pt.Login}}\">Login</a>\n      {{end}}\n    </div>\n    <div class=\"site-name\">\n      <a href=\"{{$pt.Root}}\"><h1>{{.SiteName}}</h1></a>\n    </div>\n  </div>\n  {{if .Err}}\n    {{- template \"error\" . -}}\n  {{end}}\n  {{template \"pane\" .}}\n</div>\n{{end}}\n{{define \"error\"}}\n<div class=\"error\">\n  <strong style=\"\">{{.Err.Error}}{{if .ErrShouldLogin}} - <a href=\"{{.Paths.Login}}\">Login</a>?{{end}}</strong>\n</div>\n{{end}}\n\n"

	UserEdit = "{{- define \"panestyle\" -}}\n<style>\n\n</style>\n{{- end -}}\n{{define \"pane\"}}\n{{- $pt := .Paths -}}\n{{- $pr := $pt.Params -}}\n<div>\n  <h1>User Edit</h1>\n  <form method=\"post\" action=\"{{$pt.UpdateUserAction}}\">\n    <input type=\"hidden\" name=\"{{$pr.Xsrf}}\" value=\"{{.XsrfToken}}\">\n    <input type=\"hidden\" name=\"{{$pr.UserId}}\" value=\"{{.ObjectUser.UserId}}\">\n    <input type=\"hidden\" name=\"{{$pr.Version}}\" value=\"{{.ObjectUser.Version}}\">\n  <dl>\n    <dt>Ident</dt>\n    <dd>{{ .ObjectUser.Ident }}</dd>\n    <dt>Created</dt>\n    <dd>{{ .ObjectUser.CreatedTime }}</dd>\n    <fieldset>\n      <legend>Capabilities</legend>\n      <div style=\"display: table\">\n      {{- $canedit := .CanEditCap -}}\n      {{ range .Cap }}\n        <input \n            type=\"hidden\" \n            name=\"{{.Cap | $pr.OldUserCapability}}\" \n            value=\"{{if .Has}}{{$pr.True}}{{else}}{{$pr.False}}{{end}}\"\n            {{if not $canedit}}disabled{{end}}>\n        <div style=\"display:table-row\">\n          <div style=\"display:table-cell\">{{.Cap}}</div>\n          <div style=\"display:table-cell\">\n            <label for=\"{{.Cap | $pr.NewUserCapability}}-yes\">Yes<label>\n            <input \n                type=\"radio\"\n                id=\"{{.Cap | $pr.NewUserCapability}}-yes\" \n                name=\"{{.Cap | $pr.NewUserCapability}}\" \n                value=\"{{$pr.True}}\"\n                {{if not $canedit}}disabled{{end}}\n                {{if .Has}}checked{{end}}>\n          </div>\n          <div style=\"display:table-cell\">\n            <label for=\"{{.Cap | $pr.NewUserCapability}}-no\">No<label>\n              <input \n                  type=\"radio\"\n                  id=\"{{.Cap | $pr.NewUserCapability}}-no\" \n                  name=\"{{.Cap | $pr.NewUserCapability}}\" \n                  value=\"{{$pr.False}}\"\n                  {{if not $canedit}}disabled{{end}}\n                  {{if not .Has}}checked{{end}}>\n          </div>\n        </div>\n      {{ end }}\n      </div>\n    </fieldset>\n  </dl>\n  <input type=\"submit\" {{if not $canedit}}disabled{{end}}>\n  </form>\n</div>\n{{end}}\n"

	Viewer = "{{- define \"panestyle\" -}}\n<style>\n  .viewer .thepic, .viewer .thevideo {\n    height: auto;\n    max-width: 100%;\n    width: auto;\n  }\n  .viewer .votebutton {\n    padding: 0 .5cm 0 .5cm;\n    border: none;\n    margin: 10px;\n  }\n  .viewer .pic {\n    text-align: center;\n    max-height: 768px;\n     \n  }\n  \n  .viewer .pic img {\n    max-height: inherit;\n  }\n  \n    .viewer .pic video {\n    max-height: inherit;\n  }\n\n  .actions {\n    float: right;\n    display: inline;\n  }\n  \n  .votebar {\n    float: left;\n  }\n  .actionbar:after {\n    clear: both;\n    content: \"\";\n    display: block;\n  }\n  \n  .votebar form {\n    display: inline;\n  }\n  .votebar .up {\n    color: white;\n    background-color: hsl(120, 75%, 50%);\n  }\n  .votebar .down {\n    color: black;\n    background-color: hsl(0, 75%, 50%);\n  }\n  .votebar .neutral {\n    color: white;\n    background-color: hsl(0, 0%, 50%);\n  }\n  .votebar .up.unpicked {\n    color: white;\n    background-color: hsl(120, 25%, 75%);\n  }\n  .votebar .down.unpicked {\n    color: black;\n    background-color: hsl(0, 25%, 75%);\n  }\n  .votebar .neutral.unpicked {\n    color: white;\n    background-color: hsl(0, 0%, 75%);;\n  }\n  .votebutton.unvoted {\n    cursor: pointer;\n  }\n</style>\n{{- end -}}\n{{define \"pane\"}}\n{{- $pt := .Paths -}}\n{{- $pr := $pt.Params -}}\n<div class=\"viewer\">\n  <div class=\"pic\">\n    {{if ne .Pic.File.Format.String `WEBM`}}\n    <img class=\"thepic\" src=\"{{$pt.PicFile .Pic.File}}\" />\n    {{else}}\n    <video\n        class=\"thevideo\"\n        src=\"{{$pt.PicFile .Pic.File}}\"\n        loop\n        muted\n        autoplay\n        controls>\n      Your browser does not support the video tag.\n    </video>\n    {{end}}\n  </div>\n  <div class=\"actionbar\">\n    <div class=\"votebar\">\n      <form action=\"{{$pt.VoteAction}}\" method=\"post\">\n        <input type=\"hidden\" name=\"{{$pr.Xsrf}}\" value=\"{{.XsrfToken}}\" />\n        <input type=\"hidden\" name=\"{{$pr.Vote}}\" value=\"DOWN\" />\n        <input type=\"hidden\" name=\"{{$pr.PicId}}\" value=\"{{.Pic.Id}}\" />\n        <input type=\"hidden\" name=\"{{$pr.Next}}\" value=\"{{$pt.Viewer .Pic.Id}}\" />\n        <input \n            type=\"submit\" \n            value=\"▼\" \n            {{if .PicVote}}\n              class=\"votebutton down\n                {{- if ne .PicVote.Vote.String `DOWN`}} unpicked{{end}}\"\n              disabled\n            {{else}}\n              class=\"votebutton down unvoted\"\n            {{end}}\n        />\n      </form>\n      <form action=\"{{$pt.VoteAction}}\" method=\"post\">\n        <input type=\"hidden\" name=\"{{$pr.Xsrf}}\" value=\"{{.XsrfToken}}\" />\n        <input type=\"hidden\" name=\"{{$pr.Vote}}\" value=\"NEUTRAL\" />\n        <input type=\"hidden\" name=\"{{$pr.PicId}}\" value=\"{{.Pic.Id}}\" />\n        <input type=\"hidden\" name=\"{{$pr.Next}}\" value=\"{{$pt.Viewer .Pic.Id}}\" />\n        <input\n            type=\"submit\" \n            value=\"meh\" \n            {{if .PicVote}}\n              class=\"votebutton neutral\n                {{- if ne .PicVote.Vote.String `NEUTRAL`}} unpicked{{end}}\"\n              disabled\n            {{else}}\n              class=\"votebutton neutral unvoted\"\n            {{end}}\n        />\n      </form>\n      <form action=\"{{$pt.VoteAction}}\" method=\"post\">\n        <input type=\"hidden\" name=\"{{$pr.Xsrf}}\" value=\"{{.XsrfToken}}\" />\n        <input type=\"hidden\" name=\"{{$pr.Vote}}\" value=\"UP\" />\n        <input type=\"hidden\" name=\"{{$pr.PicId}}\" value=\"{{.Pic.Id}}\" />\n        <input type=\"hidden\" name=\"{{$pr.Next}}\" value=\"{{$pt.Viewer .Pic.Id}}\" />\n        <input \n            type=\"submit\" \n            value=\"▲\"\n            {{if .PicVote}}\n              class=\"votebutton up\n                {{- if ne .PicVote.Vote.String `UP`}} unpicked{{end}}\"\n              disabled\n            {{else}}\n              class=\"votebutton up unvoted\"\n            {{end}}\n        />\n      </form>\n    </div>\n    <div class=\"actions\">\n      <a href=\"{{$pt.PicFile .Pic.File}}\">View Full</a>\n    </div>\n  </div>\n  {{ .Pic }}\n  <br>\n  {{template \"commentreply\" .PicComment}}\n  {{template \"comment\" .PicComment.Child}}\n  \n  <hr>\n    {{if .Pic.PendingDeletion }}\n    <div>This pic is pending deletion</div>\n    {{else}}\n    <form action=\"{{$pt.SoftDeletePicAction}}\" method=\"post\">\n      <input type=\"hidden\" name=\"{{$pr.Xsrf}}\" value=\"{{.XsrfToken}}\" />\n      <input type=\"hidden\" name=\"{{$pr.PicId}}\" value=\"{{.Pic.Id}}\" />\n      <input type=\"hidden\" name=\"{{$pr.Next}}\" value=\"{{$pt.Viewer .Pic.Id}}\" />\n      <input type=\"details\" name=\"{{$pr.DeletePicDetails}}\" placeholder=\"Details why this pic is deleted\" /><br />\n      <select name=\"{{$pr.DeletePicReason}}\">\n        {{ range .DeletionReason }}\n          <option value=\"{{.Value}}\" {{/*1 is NONE*/}}{{if eq .Value 1}}selected=\"selected\"{{end}}>{{.Name}}</option>\n        {{end}}\n      </select>\n      <br/>\n      <input type=\"submit\" value=\"Delete\"/>\n      <label>\n        <input type=\"checkbox\" name=\"{{$pr.DeletePicReally}}\" value=\"non-empty text\" />\n        Really?\n      </label>\n    </form>\n    {{end}}\n<div>\n{{end}}\n{{define \"comment\"}}\n  <ul>\n\t    {{- range . -}}\n\t    {{- $pt := .Paths -}}\n\t    {{- $pr := $pt.Params -}}\n\t    <li>\n\t      <div id=\"{{($pt.ViewerComment .PicId .CommentId).Fragment}}\">\n\t      \t{{.CommentId}}: {{.Text}}\n\t      </div>\n\t      <a href=\"{{$pt.CommentReply .PicId .CommentId}}\">Reply</a>\n\t      {{if .Child}}{{template \"comment\" .Child}}{{end}}\n\t    </li>\n\t    {{- end -}}\n  </ul>\n{{end}}\n"
)
