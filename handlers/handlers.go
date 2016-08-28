//go:generate protoc api.proto --go_out=.
package handlers

import (
	"compress/gzip"
	"crypto/rsa"
	"database/sql"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"

	"pixur.org/pixur/status"
)

type registerFunc func(mux *http.ServeMux, c *ServerConfig)

var (
	handlerFuncs []registerFunc
)

type ServerConfig struct {
	DB          *sql.DB
	PixPath     string
	TokenSecret []byte
	PrivateKey  *rsa.PrivateKey
	PublicKey   *rsa.PublicKey
}

func register(rf registerFunc) {
	handlerFuncs = append(handlerFuncs, rf)
}

func AddAllHandlers(mux *http.ServeMux, c *ServerConfig) {
	for _, rf := range handlerFuncs {
		rf(mux, c)
	}
}

func returnTaskError(w http.ResponseWriter, sts status.S) {
	log.Println("Error in task: ", sts)
	w.Header().Set("Content-Type", "text/plain")
	code := sts.Code()
	http.Error(w, code.String()+": "+sts.Message(), code.HttpStatus())
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
					log.Println("Error writing JSON", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				return
			case "application/proto":
				w.Header().Set("Content-Type", "application/proto")
				raw, err := proto.Marshal(pb)
				if err != nil {
					log.Println("Error building Proto", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if _, err := writer.Write(raw); err != nil {
					log.Println("Error writing Proto", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				return
			}
		}
	}
	// default
	w.Header().Set("Content-Type", "application/json")
	if err := protoJSONMarshaller.Marshal(writer, pb); err != nil {
		log.Println("Error writing JSON", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
