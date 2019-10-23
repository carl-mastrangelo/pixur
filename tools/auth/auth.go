// Package auth manages auth tokens for command line usage.
package auth

import (
	"bufio"
	"context"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"log"
	"os"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	tspb "github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	gstatus "google.golang.org/grpc/status"

	"pixur.org/pixur/api"
	"pixur.org/pixur/be/status"
	"pixur.org/pixur/tools/config"
)

const (
	authPwtHeaderKey = "pixur-auth-token"
	pixPwtHeaderKey  = "pixur-pix-token"
	httpHeaderKey    = "pixur-http-header-bin"
)

// cc is optional
func Auth(ctx context.Context, cc **grpc.ClientConn) (context.Context, status.S) {
	conf, sts := getAndUpdateConfig(ctx, cc)
	if sts != nil {
		return nil, sts
	}

	md := make(metadata.MD, 2)
	if conf.AuthToken != "" {
		md.Set(authPwtHeaderKey, conf.AuthToken)
	}
	if conf.PixToken != "" {
		md.Set(pixPwtHeaderKey, conf.PixToken)
	}
	return metadata.NewOutgoingContext(ctx, md), nil
}

func getAndUpdateConfig(ctx context.Context, cc **grpc.ClientConn) (
	_ *config.Config, stscap status.S) {
	oldconf, sts := config.GetConfig()
	if sts != nil {
		if sts.Code() == codes.NotFound {
			oldconf = &config.Config{}
		} else {
			return nil, sts
		}
	}

	newconf := proto.Clone(oldconf).(*config.Config)
	if sts := updateConfig(ctx, cc, newconf); sts != nil {
		return nil, sts
	}
	if !proto.Equal(oldconf, newconf) {
		if sts := config.SetConfig(newconf); sts != nil {
			status.ReplaceOrSuppress(&stscap, sts)
		}
	}

	return newconf, nil
}

// shouldRefresh determines if a token refresh is needed.  notAfter indicates when the token is
// expired and must be non-nil.  softNotAfter indicates that the token is still valid, but should be
// refreshed.  softNotAfter may be nil.  The return value useToken indicates that the auth token
// can be reusued.  useCreds indicates if a ident/secret is needed.  At most one of useToken or
// useCreds can be true.
func shouldRefresh(now func() time.Time, notAfter, softNotAfter *tspb.Timestamp) (
	useToken, useCreds bool, _ status.S) {
	if notAfter == nil {
		return false, false, status.Internal(nil, "missing not after")
	}
	na, err := ptypes.Timestamp(notAfter)
	if err != nil {
		return false, false, status.Internal(err, "bad not after")
	}
	ts := now()
	if ts.After(na) {
		return false, true, nil
	}
	if softNotAfter != nil {
		sna, err := ptypes.Timestamp(softNotAfter)
		if err != nil {
			return false, false, status.Internal(err, "bad soft not after")
		}
		if ts.After(sna) {
			return true, false, nil
		}
	}
	return false, false, nil
}

func updateConfig(ctx context.Context, cc **grpc.ClientConn, conf *config.Config) (stscap status.S) {
	var (
		useToken bool
		useCreds bool = true
	)
	if conf.AuthPayload != nil {
		var sts status.S
		useToken, useCreds, sts =
			shouldRefresh(time.Now, conf.AuthPayload.NotAfter, conf.AuthPayload.SoftNotAfter)
		if sts != nil {
			return sts
		}
		if !useToken && !useCreds {
			return nil
		}
	}

	var conn *grpc.ClientConn
	if cc == nil || *cc == nil {
		if conf.PixurTarget == "" {
			target, sts := getTargetFromTerm(os.Stderr, os.Stdin)
			if sts != nil {
				return sts
			}
			conf.PixurTarget = target
		}
		var err error
		conn, err = grpc.DialContext(ctx, conf.PixurTarget, grpc.WithInsecure())
		if err != nil {
			return status.From(err)
		}
		if cc != nil {
			*cc = conn
		} else {
			defer func() {

				if err := conn.Close(); err != nil {
					status.ReplaceOrSuppress(&stscap, status.From(err))

				}
			}()
		}

	}

	client := api.NewPixurServiceClient(conn)

	if useToken && conf.AuthToken != "" {
		res, err := client.GetRefreshToken(ctx, &api.GetRefreshTokenRequest{
			PreviousAuthToken: conf.AuthToken,
		})
		if err != nil {
			if gstatus.Code(err) == codes.Unauthenticated {
				log.Println("Previous Auth token denied, falling back to creds", err)
				useCreds = true
			} else {
				return status.From(err)
			}
		} else {
			conf.AuthToken, conf.PixToken = res.AuthToken, res.PixToken
			conf.AuthPayload, conf.PixPayload = res.AuthPayload, res.PixPayload
			return nil
		}
	}

	if useCreds {
		ident, secret, sts := getIdentSecretFromTerm(os.Stderr, os.Stdin)
		if sts != nil {
			return sts
		}
		res, err := client.GetRefreshToken(ctx, &api.GetRefreshTokenRequest{
			Ident:  ident,
			Secret: secret,
		})
		if err != nil {
			return status.From(err)
		} else {
			conf.AuthToken, conf.PixToken = res.AuthToken, res.PixToken
			conf.AuthPayload, conf.PixPayload = res.AuthPayload, res.PixPayload
			return nil
		}
	}

	panic("unreachable")
}

func getTargetFromTerm(w io.Writer, r io.Reader) (string, status.S) {
	if _, err := w.Write([]byte("gRPC Target of Pixur Backend (e.g. dns:///localhost:8079): \n")); err != nil {
		return "", status.From(err)
	}
	br := bufio.NewReader(r)
	target, err := br.ReadString('\n')
	if err != nil {
		return "", status.From(err)
	}

	return target[:len(target)-1], nil
}

func getIdentSecretFromTerm(w io.Writer, r *os.File) (string, string, status.S) {
	if _, err := w.Write([]byte("User Ident (e.g. admin): \n")); err != nil {
		return "", "", status.From(err)
	}
	br := bufio.NewReader(r)
	ident, err := br.ReadString('\n')
	if err != nil {
		return "", "", status.From(err)
	}

	if _, err := w.Write([]byte("Secret (e.g. 12345): \n")); err != nil {
		return "", "", status.From(err)
	}
	secret, err := terminal.ReadPassword(int(r.Fd()))
	if err != nil {
		return "", "", status.From(err)
	}

	return ident[:len(ident)-1], string(secret), nil
}
