package handlers

import (
	"context"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"pixur.org/pixur/api"
)

type pixHandler struct {
	ctx context.Context
	c   api.PixurServiceClient
}

var _ os.FileInfo = &pixurFI{}

type pixurFI struct {
	picFileID string
	format    api.PicFile_Format
	name      string
	size      int64
	modtime   time.Time
}

func (fi *pixurFI) Name() string {
	return fi.name
}

func (fi *pixurFI) Size() int64 {
	return fi.size
}

func (fi *pixurFI) Mode() os.FileMode {
	return os.FileMode(0775)
}

func (fi *pixurFI) ModTime() time.Time {
	return fi.modtime
}

func (fi *pixurFI) IsDir() bool {
	return false
}

func (fi *pixurFI) Sys() interface{} {
	return nil
}

var _ http.FileSystem = &pixurFS{}

type pixurFS struct {
	ctx context.Context
	c   api.PixurServiceClient
}

func (fs *pixurFS) Open(name string) (http.File, error) {
	p := path.Base(name)
	picFileID := strings.TrimSuffix(p, path.Ext(p))

	format := api.PicFile_UNKNOWN
	switch path.Ext(p) {
	case ".jpg":
		format = api.PicFile_JPEG
	case ".gif":
		format = api.PicFile_GIF
	case ".webm":
		format = api.PicFile_WEBM
	case ".png":
		format = api.PicFile_PNG
	}

	resp, err := fs.c.LookupPicFile(fs.ctx, &api.LookupPicFileRequest{
		PicFileId: picFileID,
		Format:    format,
	})
	if err != nil {
		glog.Info(err)
		if s, ok := status.FromError(err); ok {
			switch s.Code() {
			case codes.NotFound:
				return nil, os.ErrNotExist
			case codes.PermissionDenied:
				return nil, os.ErrPermission
			case codes.Unauthenticated:
				return nil, os.ErrPermission
			}
		}

		return nil, err
	}

	mtime, err := ptypes.Timestamp(resp.PicFile.ModifiedTime)
	if err != nil {
		glog.Info(err)
		return nil, err
	}

	rpfc, err := fs.c.ReadPicFile(fs.ctx)
	if err != nil {
		glog.Info(err)
		return nil, err
	}

	return &pixurFile{
		rpfc: rpfc,
		fi: &pixurFI{
			picFileID: resp.PicFile.Id,
			format:    resp.PicFile.Format,
			name:      p,
			size:      resp.PicFile.Size,
			modtime:   mtime,
		},
	}, nil
}

var _ http.File = &pixurFile{}

type pixurFile struct {
	rpfc   api.PixurService_ReadPicFileClient
	offset int64
	fi     *pixurFI
}

func (f *pixurFile) Close() error {
	if err := f.rpfc.CloseSend(); err != nil {
		glog.Info(err)
		return err
	}
	_, err := f.rpfc.Recv()
	if err == nil || err == io.EOF {
		return nil
	}
	glog.Info(err)
	return err
}

func (f *pixurFile) Stat() (os.FileInfo, error) {
	return f.fi, nil
}

func (f *pixurFile) Seek(offset int64, whence int) (int64, error) {
	if whence == os.SEEK_SET {
		f.offset = offset
	} else if whence == os.SEEK_CUR {
		f.offset += offset
	} else if whence == os.SEEK_END {
		f.offset = offset + f.fi.Size()
	} else {
		return f.offset, os.ErrInvalid
	}
	return f.offset, nil
}

func (f *pixurFile) Read(data []byte) (int, error) {
	err := f.rpfc.Send(&api.ReadPicFileRequest{
		PicFileId: f.fi.picFileID,
		Format:    f.fi.format,
		Offset:    f.offset,
		Limit:     int64(len(data)),
	})
	if err != nil {
		glog.Info(err)
		return 0, err
	}
	resp, err := f.rpfc.Recv()
	if err != nil {
		glog.Info(err)
		return 0, err
	}
	n := copy(data, resp.Data)
	f.offset += int64(n)
	if resp.Eof {
		return n, io.EOF
	}
	return n, nil
}

func (f *pixurFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, os.ErrInvalid
}

func (h *pixHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var md metadata.MD
	if c, err := r.Cookie(pixPwtCookieName); err == nil {
		md = metadata.Pairs(pixPwtHeaderKey, c.Value)
	}
	ctx := metadata.NewOutgoingContext(r.Context(), md)

	http.FileServer(&pixurFS{
		ctx: ctx,
		c:   h.c,
	}).ServeHTTP(w, r)
}
