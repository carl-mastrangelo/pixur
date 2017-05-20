package handlers

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	oldcontext "golang.org/x/net/context"
	"golang.org/x/net/webdav"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/schema/db"
)

type PixFS struct {
	// embeds
	webdav.FileSystem

	// deps
	DB db.DB
}

func (fs *PixFS) Mkdir(ctx oldcontext.Context, name string, perm os.FileMode) error {
	return os.ErrPermission
}

func (fs *PixFS) RemoveAll(ctx oldcontext.Context, name string) error {
	return os.ErrPermission
}

func (fs *PixFS) Rename(ctx oldcontext.Context, oldName, newName string) error {
	return os.ErrPermission
}

func (fs *PixFS) Stat(ctx oldcontext.Context, name string) (os.FileInfo, error) {
	if isThumbnail(name) {
		return nil, os.ErrNotExist
	}

	return fs.FileSystem.Stat(context.TODO(), name)
}

func (fs *PixFS) OpenFile(ctx oldcontext.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	if isThumbnail(name) {
		return nil, os.ErrNotExist
	}
	if flag != os.O_RDONLY || perm != 0 {
		return nil, os.ErrPermission
	}

	f, err := fs.FileSystem.OpenFile(ctx, name, flag, perm)
	if err != nil {
		return nil, err
	}

	return &wrapper{File: f}, nil
}

type wrapper struct {
	webdav.File
}

func (w *wrapper) Readdir(count int) ([]os.FileInfo, error) {
	infos, err := w.File.Readdir(count)
	if err != nil {
		return nil, err
	}
	var goodInfos []os.FileInfo

	for _, info := range infos {
		if isThumbnail(info.Name()) {
			continue
		}

		goodInfos = append(goodInfos, info)
	}

	return goodInfos, nil
}

func isThumbnail(name string) bool {
	base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
	var v schema.Varint
	consumed, err := v.Decode(base)
	if err != nil {
		return false
	}
	return len(base) > consumed
}

type davAuthHandler struct {
	Handler http.Handler
	Now     func() time.Time
}

func (h *davAuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Currently blocks dav from working, mostly.
	rc := &requestChecker{r: r, now: h.Now}
	rc.checkPixAuth()
	if rc.sts != nil {
		httpError(w, rc.sts)
		return
	}

	h.Handler.ServeHTTP(w, r)
}

/*
func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/x/dav/", &davAuthHandler{
			Handler: &webdav.Handler{
				Prefix: "/x/dav",
				FileSystem: &PixFS{
					FileSystem: webdav.Dir(c.PixPath),
					DB:         c.DB,
				},
				LockSystem: webdav.NewMemLS(),
			},
			Now: time.Now,
		})
	})
}
*/
