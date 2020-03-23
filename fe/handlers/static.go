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

var picFileFormatMime = map[api.PicFile_Format]string{
	api.PicFile_JPEG: "image/jpeg",
	api.PicFile_GIF:  "image/gif",
	api.PicFile_PNG:  "image/png",
	api.PicFile_WEBM: "video/webm",
	api.PicFile_MP4:  "video/mp4",
}

var picFileFormatExt = map[api.PicFile_Format]string{
	api.PicFile_JPEG: ".jpg",
	api.PicFile_GIF:  ".gif",
	api.PicFile_PNG:  ".png",
	api.PicFile_WEBM: ".webm",
	api.PicFile_MP4:  ".mp4",
}

var picFileFormatTypes = map[string]api.PicFile_Format{
	".jpg":  api.PicFile_JPEG,
	".gif":  api.PicFile_GIF,
	".png":  api.PicFile_PNG,
	".webm": api.PicFile_WEBM,
	".mp4":  api.PicFile_MP4,
}

func init() {
	if len(picFileFormatExt) != len(api.PicFile_Format_name)-1 {
		panic("format map wrong")
	}
	if len(picFileFormatTypes) != len(api.PicFile_Format_name)-1 {
		panic("format map wrong")
	}
	for k, _ := range api.PicFile_Format_name {
		if api.PicFile_Format(k) == api.PicFile_UNKNOWN {
			continue
		}
		if _, present := picFileFormatExt[api.PicFile_Format(k)]; !present {
			panic("missing value in format map")
		}
	}
}

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

	format, ok := picFileFormatTypes[path.Ext(p)]
	if !ok {
		httpError(w, &HTTPErr{
			Code:    http.StatusNotFound,
			Message: "Unknown file extension",
		})
		return
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
