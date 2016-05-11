package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/golang/protobuf/proto"
	descriptor "google/protobuf"
	plugin "google/protobuf/compiler"

	"pixur.org/pixur/schema/db"
	"pixur.org/pixur/schema/db/model"
)

type keyType string

var (
	primaryKey keyType = "PRIMARY KEY"
	uniqueKey  keyType = "UNIQUE"
	indexKey   keyType = "INDEX"
)

type Generator struct {
	args                          *tplArgs
	protoPackageMap, protoNameMap map[string]*descriptor.FileDescriptorProto
}

func New() *Generator {
	return &Generator{
		args:            new(tplArgs),
		protoPackageMap: make(map[string]*descriptor.FileDescriptorProto),
		protoNameMap:    make(map[string]*descriptor.FileDescriptorProto),
	}
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

	name, content, err := g.generateFile(req.FileToGenerate[0], req.ProtoFile)
	if err != nil {
		return &plugin.CodeGeneratorResponse{
			Error: proto.String(err.Error()),
		}
	}

	return &plugin.CodeGeneratorResponse{
		File: []*plugin.CodeGeneratorResponse_File{{
			Name:    proto.String(name),
			Content: proto.String(string(content)),
		}},
	}
}

func (g *Generator) addDefaultImports() {
	g.args.Imports = append(g.args.Imports, tplImport{
		FullName: "database/sql",
	})
	g.args.Imports = append(g.args.Imports, tplImport{
		FullName: "github.com/golang/protobuf/proto",
	})
	g.args.Imports = append(g.args.Imports, tplImport{
		FullName: "pixur.org/pixur/schema/db",
	})
}

func (g *Generator) addProtoImports(srcName string) {
	for _, dep := range g.protoNameMap[srcName].Dependency {
		fd := g.protoNameMap[dep]
		var shortName, dummyName string
		if len(fd.MessageType) > 0 {
			shortName = filepath.Base(fd.GetOptions().GetGoPackage())
			dummyName = fmt.Sprintf("%s.%s{}", shortName, *fd.MessageType[0].Name)
		} else {
			shortName = "_"
		}
		g.args.Imports = append(g.args.Imports, tplImport{
			ShortName: shortName,
			FullName:  filepath.Dir(dep),
			Dummy:     dummyName,
		})
	}
}

func (g *Generator) generateFile(
	srcName string, fds []*descriptor.FileDescriptorProto) (string, []byte, error) {
	for _, fd := range fds {
		if fd.GetOptions().GetGoPackage() == "" {
			return "", nil, fmt.Errorf("%s doesn't have go_package", *fd.Name)
		}
		g.protoNameMap[*fd.Name] = fd
		g.protoPackageMap[*fd.Package] = fd
		if *fd.Name == srcName {
			g.args.Name = fd.GetOptions().GetGoPackage()
		}

		for _, msg := range fd.MessageType {
			if msg.Options == nil || !proto.HasExtension(msg.Options, model.E_TabOpts) {
				continue
			}
			opts, err := proto.GetExtension(msg.Options, model.E_TabOpts)
			if err != nil {
				return "", nil, err
			}
			if err := g.addTable(msg, opts.(*model.TableOptions)); err != nil {
				return "", nil, err
			}
		}
	}

	g.addDefaultImports()
	g.addProtoImports(srcName)

	dstName := strings.Replace(srcName, ".proto", ".tab.go", -1)
	content, err := g.renderTables()
	return dstName, content, err
}

func (g *Generator) renderTables() ([]byte, error) {
	buf := bytes.Buffer{}
	funcs := template.FuncMap{
		"goesc": func(input string) interface{} {
			return strconv.Quote(input)
		},
		"sqlesc": func(input string) interface{} {
			return db.GetAdapter().Quote(input)
		},
		"sqlblobidxesc": func(input string) interface{} {
			return db.GetAdapter().QuoteCreateBlobIdxCol(input)
		},
	}

	if err := tpl.Funcs(funcs).Execute(&buf, g.args); err != nil {
		return nil, err
	}

	data := buf.Bytes()
	fmtData, err := format.Source(data)
	if err != nil {
		err = fmt.Errorf("%v\n%s", err, string(data))
	}
	return fmtData, err
}

type tplArgs struct {
	Name    string
	Imports []tplImport
	Tables  []tplTable
}

type tplImport struct {
	ShortName, FullName, Dummy string
}

type tplTable struct {
	Name, GoType, GoDataType string
	Columns                  []tplColumn
	Indexes                  []tplIndex
}

type tplColumn struct {
	GoName, SqlName, GoType, SqlType string
}

func (t tplColumn) IsBlobIdxCol() bool {
	return t.SqlType == db.GetAdapter().BlobType
}

type tplIndex struct {
	Name    string
	KeyType keyType
	Columns []tplColumn
}

func colNameToGoName(name string) string {
	parts := strings.Split(name, "_")
	for i := 0; i < len(parts); i++ {
		parts[i] = strings.Title(parts[i])
	}
	return strings.Join(parts, "")
}

func (g *Generator) addTable(msg *descriptor.DescriptorProto, opts *model.TableOptions) error {
	t := tplTable{
		Name:   opts.Name,
		GoType: *msg.Name,
	}
	if t.Name == "" {
		t.Name = *msg.Name
	}

	colNames := make(map[string]tplColumn)
	for _, f := range msg.Field {
		col := tplColumn{
			SqlName: *f.Name,
			GoName:  colNameToGoName(*f.Name),
		}
		switch *f.Type {
		case descriptor.FieldDescriptorProto_TYPE_FIXED32:
			fallthrough
		case descriptor.FieldDescriptorProto_TYPE_SFIXED32:
			fallthrough
		case descriptor.FieldDescriptorProto_TYPE_SINT32:
			fallthrough
		case descriptor.FieldDescriptorProto_TYPE_INT32:
			col.GoType = "int32"
			col.SqlType = db.GetAdapter().IntType
		case descriptor.FieldDescriptorProto_TYPE_ENUM:
			col.GoType = g.typeNameToGoName(*f.TypeName)
			col.SqlType = db.GetAdapter().IntType
		case descriptor.FieldDescriptorProto_TYPE_FIXED64:
			fallthrough
		case descriptor.FieldDescriptorProto_TYPE_SFIXED64:
			fallthrough
		case descriptor.FieldDescriptorProto_TYPE_SINT64:
			fallthrough
		case descriptor.FieldDescriptorProto_TYPE_INT64:
			col.GoType = "int64"
			col.SqlType = db.GetAdapter().BigIntType
		case descriptor.FieldDescriptorProto_TYPE_BOOL:
			col.GoType = "bool"
			col.SqlType = db.GetAdapter().BoolType
		case descriptor.FieldDescriptorProto_TYPE_STRING:
			col.GoType = "string"
			col.SqlType = db.GetAdapter().BlobType
		case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
			col.GoType = g.typeNameToGoName(*f.TypeName)
			col.SqlType = db.GetAdapter().BlobType
		case descriptor.FieldDescriptorProto_TYPE_BYTES:
			col.GoType = "[]byte"
			col.SqlType = db.GetAdapter().BlobType
		default:
			return fmt.Errorf("No type for %s", f.Type)
		}
		colNames[*f.Name] = col
		t.Columns = append(t.Columns, col)
		if *f.Name == "data" {
			t.GoDataType = g.typeNameToGoName(*f.TypeName)
		}
	}
	if _, present := colNames["data"]; !present {
		return fmt.Errorf("Missing data col on table %s", t.Name)
	}

	for _, k := range opts.Key {
		if len(k.Col) == 0 {
			return fmt.Errorf("No cols in key on table %s", t.Name)
		}
		idx := tplIndex{
			Name: k.Name,
		}
		for _, c := range k.Col {
			if _, present := colNames[c]; !present {
				return fmt.Errorf("Unknown col on table %s", t.Name)
			}
			idx.Columns = append(idx.Columns, colNames[c])
		}

		switch k.KeyType {
		case model.KeyType_PRIMARY:
			idx.KeyType = primaryKey
			if idx.Name == "" {
				idx.Name = "Primary"
			}
		case model.KeyType_UNIQUE:
			idx.KeyType = uniqueKey
			if idx.Name == "" {
				return fmt.Errorf("Missing name for key on table %s", t.Name)
			}
		case model.KeyType_INDEX:
			idx.KeyType = indexKey
			if idx.Name == "" {
				return fmt.Errorf("Missing name for key on table %s", t.Name)
			}
		default:
			return fmt.Errorf("Unknown key type on table %s", t.Name)
		}
		idx.Name = t.Name + idx.Name
		t.Indexes = append(t.Indexes, idx)
	}
	g.args.Tables = append(g.args.Tables, t)

	return nil
}

func (g *Generator) typeNameToGoName(fqProtoName string) string {
	var best string
	for pack := range g.protoPackageMap {
		if strings.HasPrefix(fqProtoName, "."+pack+".") {
			if len(pack) > len(best) {
				best = pack
			}
		}
	}
	if best != "" {
		msg := strings.TrimPrefix(fqProtoName, "."+best+".")
		if pack := g.protoPackageMap[best].GetOptions().GetGoPackage(); pack != g.args.Name {
			return pack + "." + strings.Join(strings.Split(msg, "."), "_")
		}
		return strings.Join(strings.Split(msg, "."), "_")
	}
	panic("Could not find type!" + fqProtoName)
}
