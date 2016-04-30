package generator

import (
	_ "bytes"
	"fmt"
	"io"
	"io/ioutil"
	_ "log"
	"strings"
	_ "text/template"

	"github.com/golang/protobuf/proto"
	descriptor "google/protobuf"
	plugin "google/protobuf/compiler"

	"pixur.org/pixur/schema/db/model"
)

type Generator struct {
}

func (g *Generator) Run(out io.Writer, in io.Reader) error {
	req, err := readRequest(in)
	if err != nil {
		return err
	}

	resp := g.run(req)
	if err := writeResponse(out, resp); err != nil {
		return err
	}

	return nil
}

func readRequest(r io.Reader) (*plugin.CodeGeneratorRequest, error) {
	raw, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	req := new(plugin.CodeGeneratorRequest)
	if err := proto.Unmarshal(raw, req); err != nil {
		return nil, err
	}
	return req, nil
}

func writeResponse(w io.Writer, resp *plugin.CodeGeneratorResponse) error {
	data, err := proto.Marshal(resp)
	if err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	return nil
}

func (g *Generator) run(req *plugin.CodeGeneratorRequest) *plugin.CodeGeneratorResponse {
	if len(req.FileToGenerate) != 1 {
		return &plugin.CodeGeneratorResponse{
			Error: proto.String("Can only generate 1 file"),
		}
	}

	content, err := g.generateFile(req.ProtoFile)
	if err != nil {
		return &plugin.CodeGeneratorResponse{
			Error: proto.String(err.Error()),
		}
	}

	return &plugin.CodeGeneratorResponse{
		File: []*plugin.CodeGeneratorResponse_File{{
			Name:    proto.String(strings.Replace(req.FileToGenerate[0], ".proto", ".tab.go", -1)),
			Content: proto.String(content),
		}},
	}
}

func (g *Generator) generateFile(fds []*descriptor.FileDescriptorProto) (string, error) {
	// plugin.proto says files are ordered by dependency, and we only allow one source file,
	// so it must be last.
	for _, fd := range fds {
		if fd.GetOptions().GetGoPackage() == "" {
			return "", fmt.Errorf("%s doesn't have go_package", *fd.Name)
		}

		for _, msg := range fd.MessageType {
			if msg.Options == nil || !proto.HasExtension(msg.Options, model.E_TabOpts) {
				continue
			}
			opts, err := proto.GetExtension(msg.Options, model.E_TabOpts)
			if err != nil {
				return "", err
			}

		}
	}

	return "", nil
}
