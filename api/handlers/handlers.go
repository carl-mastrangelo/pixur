//go:generate protoc api.proto --go_out=.
package handlers

import (
	"compress/gzip"
	"crypto/rsa"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"

	"pixur.org/pixur/schema/db"
	"pixur.org/pixur/status"
)

type registerFunc func(mux *http.ServeMux, c *ServerConfig)

var (
	handlerFuncs []registerFunc
)

type ServerConfig struct {
	DB          db.DB
	PixPath     string
	TokenSecret []byte
	PrivateKey  *rsa.PrivateKey
	PublicKey   *rsa.PublicKey
	Secure      bool
}

func register(rf registerFunc) {
	handlerFuncs = append(handlerFuncs, rf)
}

func AddAllHandlers(mux *http.ServeMux, c *ServerConfig) {
	for _, rf := range handlerFuncs {
		rf(mux, c)
	}
}

var errorLog = log.New(os.Stderr, "", log.LstdFlags)

func httpError(w http.ResponseWriter, sts status.S) {
	w.Header().Set("Pixur-Status", strconv.Itoa(int(sts.Code())))
	w.Header().Set("Pixur-Message", sts.Message())

	code := sts.Code()
	http.Error(w, code.String()+": "+sts.Message(), code.HttpStatus())

	errorLog.Println(sts.String())
}

var protoJSONMarshaller = &jsonpb.Marshaler{}

func returnProtoJSON(w http.ResponseWriter, r *http.Request, pb proto.Message) {
	var writer io.Writer = w

	if encs := r.Header.Get("Accept-Encoding"); encs != "" {
		for _, enc := range strings.Split(encs, ",") {
			if strings.TrimSpace(enc) == "gzip" {
				if gw, err := gzip.NewWriterLevel(writer, gzip.BestSpeed); err != nil {
					panic(err)
				} else {
					defer gw.Close()
					// TODO: log this

					writer = gw
				}
				w.Header().Set("Content-Encoding", "gzip")
				break
			}
		}
	}
	if accept := r.Header.Get("Accept"); accept != "" {
		for _, acc := range strings.Split(accept, ",") {
			switch strings.TrimSpace(acc) {
			case "application/json":
				w.Header().Set("Content-Type", "application/json")
				if err := protoJSONMarshaller.Marshal(writer, pb); err != nil {
					httpError(w, status.InternalError(err, "error writing json"))
					return
				}
				return
			case "application/proto":
				w.Header().Set("Content-Type", "application/proto")
				raw, err := proto.Marshal(pb)
				if err != nil {
					httpError(w, status.InternalError(err, "error building proto"))
					return
				}
				if _, err := writer.Write(raw); err != nil {
					httpError(w, status.InternalError(err, "error writing proto"))
					return
				}
				return
			}
		}
	}
	// default
	w.Header().Set("Content-Type", "application/json")
	if err := protoJSONMarshaller.Marshal(writer, pb); err != nil {
		httpError(w, status.InternalError(err, "error writing json"))
	}
}
