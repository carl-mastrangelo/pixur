package handlers

import (
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"pixur.org/pixur/api"
)

type pixHandler struct {
	c api.PixurServiceClient
}

var _ io.ReadSeeker = &pixurFile{}

type pixurFile struct {
	rpfc api.PixurService_ReadPicFileClient

	picFileID string
	format    api.PicFile_Format
	size      int64

	// state
	head      http.Header
	headadded bool
	offset    int64
}

func (f *pixurFile) Seek(offset int64, whence int) (int64, error) {
	if whence == os.SEEK_SET {
		f.offset = offset
	} else if whence == os.SEEK_CUR {
		f.offset += offset
	} else if whence == os.SEEK_END {
		f.offset = offset + f.size
	} else {
		return f.offset, os.ErrInvalid
	}
	return f.offset, nil
}

func (f *pixurFile) Read(data []byte) (int, error) {
	err := f.rpfc.Send(&api.ReadPicFileRequest{
		PicFileId: f.picFileID,
		Format:    f.format,
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

func (h *pixHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var md metadata.MD
	if c, err := r.Cookie(pixPwtCookieName); err == nil {
		md = metadata.Pairs(pixPwtHeaderKey, c.Value)
	}
	ctx := metadata.NewOutgoingContext(r.Context(), md)

	p := path.Base(r.URL.Path)
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

	var header metadata.MD
	resp, err := h.c.LookupPicFile(ctx, &api.LookupPicFileRequest{
		PicFileId: picFileID,
		Format:    format,
	}, grpc.Header(&header))
	if err != nil {
		httpError(w, err)
		return
	}

	if hdrs, ok := header[httpHeaderKey]; ok {
		for _, h := range hdrs {
			var hh api.HttpHeader
			if err := proto.Unmarshal([]byte(h), &hh); err != nil {
				httpError(w, err)
				return
			}
			w.Header().Add(hh.Key, hh.Value)
		}
	}

	mtime, err := ptypes.Timestamp(resp.PicFile.ModifiedTime)
	if err != nil {
		httpError(w, err)
		return
	}
	rpfc, err := h.c.ReadPicFile(ctx)
	if err != nil {
		httpError(w, err)
		return
	}
	defer func() {
		rpfc.CloseSend()
		if _, err := rpfc.Recv(); err != nil && err != io.EOF {
			glog.Info(err)
		}
	}()

	pf := &pixurFile{
		rpfc: rpfc,

		picFileID: resp.PicFile.Id,
		format:    resp.PicFile.Format,
		size:      resp.PicFile.Size,
	}

	http.ServeContent(w, r, p, mtime, pf)

}
