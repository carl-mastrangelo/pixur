package handlers

import (
	"context"
	"io"
	"os"

	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc/metadata"

	"pixur.org/pixur/api"
	"pixur.org/pixur/schema"
	"pixur.org/pixur/status"
)

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
		if tokens, ok := md[pixPwtCookieName]; !ok || len(tokens) == 0 {
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
