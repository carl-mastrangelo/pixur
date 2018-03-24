package handlers

import (
	"context"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/schema"
	"pixur.org/pixur/be/status"
)

var picCacheTimeSeconds = 7 * 24 * time.Hour / time.Second

func apiFormatToSchemaMime(format api.PicFile_Format) (schema.Pic_Mime, status.S) {
	apiName, present := api.PicFile_Format_name[int32(format)]
	if !present {
		return schema.Pic_UNKNOWN, status.InvalidArgument(nil, "unknown format")
	}
	schemaValue, present := schema.Pic_Mime_value[apiName]
	if !present {
		return schema.Pic_UNKNOWN, status.InvalidArgument(nil, "unknown name")
	}
	return schema.Pic_Mime(schemaValue), nil
}

// TODO: add tests
func (s *serv) handleLookupPicFile(ctx context.Context, req *api.LookupPicFileRequest) (
	*api.LookupPicFileResponse, status.S) {
	if sts := authReadPicRequest(ctx); sts != nil {
		return nil, sts
	}

	mime, sts := apiFormatToSchemaMime(req.Format)
	if sts != nil {
		return nil, sts
	}

	path, sts := schema.PicFilePath(s.pixpath, req.PicFileId, mime)
	if sts != nil {
		return nil, sts
	}
	fi, err := os.Stat(path)
	if err != nil {
		return nil, status.NotFound(err, "can't open pic")
	}

	mts, err := ptypes.TimestampProto(fi.ModTime())
	if err != nil {
		return nil, status.InternalError(err, "bad ts")
	}

	md, sts := readPicHeaders()
	if sts != nil {
		return nil, sts
	}
	grpc.SendHeader(ctx, md)

	return &api.LookupPicFileResponse{
		PicFile: &api.PicFile{
			Id:     req.PicFileId,
			Format: req.Format,
			// TODO: return right value
			CreatedTime:  mts,
			ModifiedTime: mts,
			Size:         fi.Size(),
			// TODO: include the rest of the values
		},
	}, nil
}

func authReadPicRequest(ctx context.Context) status.S {
	if md, present := metadata.FromIncomingContext(ctx); present {
		if tokens, ok := md[pixPwtHeaderKey]; !ok || len(tokens) == 0 {
			if !schema.UserHasPerm(schema.AnonymousUser, schema.User_PIC_READ) {
				return status.Unauthenticated(nil, "missing pix token")
			}
		} else if len(tokens) > 1 {
			return status.Unauthenticated(nil, "too many tokens")
		} else {
			pixPayload, err := defaultPwtCoder.decode([]byte(tokens[0]))
			if err != nil {
				return status.Unauthenticated(err, err.Error())
			}
			if pixPayload.Type != api.PwtPayload_PIX {
				return status.Unauthenticated(nil, "not pix token")
			}
		}
	} else {
		return status.InternalError(nil, "missing MD")
	}
	return nil
}

func readPicHeaders() (metadata.MD, status.S) {
	h1 := &api.HttpHeader{
		Key: "Cache-Control",
	}
	if schema.UserHasPerm(schema.AnonymousUser, schema.User_PIC_READ) {
		h1.Value = "public"
	} else {
		h1.Value = "private"
	}
	h1data, err := proto.Marshal(h1)
	if err != nil {
		return nil, status.InternalError(err, "can't encode headers")
	}
	h2 := &api.HttpHeader{
		Key:   "Cache-Control",
		Value: "max-age=" + strconv.Itoa(int(picCacheTimeSeconds)),
	}
	h2data, err := proto.Marshal(h2)
	if err != nil {
		return nil, status.InternalError(err, "can't encode headers")
	}
	return metadata.Pairs(httpHeaderKey, string(h1data), httpHeaderKey, string(h2data)), nil
}

// TODO: add tests
func (s *serv) handleReadPicFile(rps api.PixurService_ReadPicFileServer) status.S {
	if sts := authReadPicRequest(rps.Context()); sts != nil {
		return sts
	}

	// ok, authed!

	var f *os.File
	for {
		req, err := rps.Recv()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return status.InternalError(err, "can't recv")
		}
		if f == nil {
			mime, sts := apiFormatToSchemaMime(req.Format)
			if sts != nil {
				return sts
			}
			path, sts := schema.PicFilePath(s.pixpath, req.PicFileId, mime)
			if sts != nil {
				return sts
			}
			f, err = os.Open(path)
			if err != nil {
				return status.NotFound(err, "can't open pic")
			}
			defer f.Close()
		}

		resp := &api.ReadPicFileResponse{}
		if req.Limit > 1048576 || req.Limit == 0 {
			resp.Data = make([]byte, 1048576)
		} else if req.Limit < 0 {
			return status.InvalidArgument(nil, "bad limit")
		} else {
			resp.Data = make([]byte, int(req.Limit))
		}
		n, err := f.ReadAt(resp.Data, req.Offset)
		if err == io.EOF {
			resp.Eof = true
		} else if err != nil {
			return status.InternalError(err, "can't read")
		}
		resp.Data = resp.Data[:n]
		if err := rps.Send(resp); err != nil {
			return status.InternalError(err, "can't send")
		}
	}
}