package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/net/webdav"

	"pixur.org/pixur/schema"
	_ "pixur.org/pixur/tasks"
)

type PixFS struct {
	// embeds
	webdav.FileSystem

	// deps
	DB *sql.DB
}

func (fs *PixFS) Mkdir(name string, perm os.FileMode) error {
	return os.ErrPermission
}

func (fs *PixFS) RemoveAll(name string) error {
	return os.ErrPermission
}

func (fs *PixFS) Rename(oldName, newName string) error {
	return os.ErrPermission
}

func (fs *PixFS) Stat(name string) (os.FileInfo, error) {
	if len(filepath.Ext(name)) != 0 {
		base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
		fmt.Println(name, "  ", base)
		var v schema.Varint
		if consumed, err := v.Decode(base); err != nil {
			return nil, os.ErrNotExist
		} else if len(base) > consumed {
			// Thumbnails have this property
			return nil, os.ErrNotExist
		}
	}

	return fs.FileSystem.Stat(name)
}

func (fs *PixFS) OpenFile(name string, flag int, perm os.FileMode) (webdav.File, error) {
	if len(filepath.Ext(name)) != 0 {
		fmt.Println(name + "zzz")
		// Exclude thumbnails from showing up, which are noisy.
		base := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
		var v schema.Varint
		if consumed, err := v.Decode(base); err != nil {
			return nil, os.ErrNotExist
		} else if len(base) > consumed {
			// Thumbnails have this property
			return nil, os.ErrNotExist
		}
	}
	if flag != os.O_RDONLY || perm != 0 {
		return nil, os.ErrPermission
	}

	f, err := fs.FileSystem.OpenFile(name, flag, perm)
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
		if strings.ContainsRune(path.Base(info.Name()), 'u') {
			continue
		}

		goodInfos = append(goodInfos, info)
	}

	return goodInfos, nil
}

func init() {
	register(func(mux *http.ServeMux, c *ServerConfig) {
		mux.Handle("/api/dav/", &webdav.Handler{
			Prefix: "/api/dav",
			FileSystem: &PixFS{
				FileSystem: webdav.Dir(c.PixPath),
				DB:         c.DB,
			},
			LockSystem: webdav.NewMemLS(),
		})
	})
}
