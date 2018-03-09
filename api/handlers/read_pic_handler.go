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

func (s *serv) handleReadPicFileInfoRequest(ctx context.Context, req *api.ReadPicFileInfoRequest) (
	*api.ReadPicFileInfoResponse, status.S) {
	if sts := authReadPicRequest(ctx); sts != nil {
		return nil, sts
	}

	path, sts := getPathForReadPic(s.pixpath, req.PicId, req.Type, req.Thumbnail)
	if sts != nil {
		return nil, sts
	}
	fi, err := os.Stat(path)
	if err != nil {
		return nil, status.NotFound(err, "can't open pic")
	}

	ts, err := ptypes.TimestampProto(fi.ModTime())
	if err != nil {
		return nil, status.InternalError(err, "bad ts")
	}

	return &api.ReadPicFileInfoResponse{
		Name:         fi.Name(),
		Size:         fi.Size(),
		ModifiedTime: ts,
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
func (s *serv) handleReadPic(rps api.PixurService_ReadPicServer) status.S {
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
			path, sts := getPathForReadPic(s.pixpath, req.PicId, req.Type, req.Thumbnail)
			if sts != nil {
				return sts
			}
			f, err = os.Open(path)
			if err != nil {
				return status.NotFound(err, "can't open pic")
			}
			defer f.Close()
		}

		resp := &api.ReadPicResponse{}
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

func getPathForReadPic(pixPath, rawPicID, typ string, thumbnail bool) (string, status.S) {
	var picID schema.Varint
	if err := picID.DecodeAll(rawPicID); err != nil {
		return "", status.NotFound(err, "Unable to decode pic id")
	}
	mime, err := schema.FromImageFormat(typ)
	if err != nil {
		return "", status.NotFound(err, "Unable to decode pic type")
	}
	p := &schema.Pic{
		PicId: int64(picID),
		Mime:  mime,
	}
	if thumbnail {
		return p.ThumbnailPath(pixPath), nil
	}
	return p.Path(pixPath), nil
}
