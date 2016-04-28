package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/golang/protobuf/proto"
	descriptor "google/protobuf"
	plugin "google/protobuf/compiler"

	"pixur.org/pixur/schema/db/model"
)

const (
	tpl = `
package tables

import (
  "database/sql"
  
  "github.com/golang/protobuf/proto"
  
  "pixur.org/pixur/schema/db"
  "pixur.org/pixur/schema"
)

type Job struct {
  Tx *sql.Tx
}

var SqlTables = []string{
{{range .}}
  {{- .SqlString -}}
{{end -}}
}

{{range .}}
{{.ScanString}}
{{.FindString}}
{{end}}
`
)

type columnType string

var (
	int16Column columnType = "smallint"
	int32Column columnType = "integer"
	int64Column columnType = "bigint"
	bytesColumn columnType = "bytea"
)

type keyType string

var (
	primaryKey keyType = "PRIMARY KEY"
	uniqueKey  keyType = "UNIQUE"
	indexKey   keyType = "INDEX"
)

type column struct {
	Name       string
	ColumnType columnType
	field      *descriptor.FieldDescriptorProto
}

type index struct {
	Name    string
	KeyType keyType
	Columns []string
}

type table struct {
	Name    string
	GoType  string
	Columns []column
	Indexes []index
	msg     *descriptor.DescriptorProto
}

func (t table) SqlString() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, `  "CREATE TABLE \"%s\" (" +`, t.Name)
	buf.WriteRune('\n')
	for _, col := range t.Columns {
		fmt.Fprintf(&buf, `    "\"%s\" %s NOT NULL, " +`, col.Name, col.ColumnType)
		buf.WriteRune('\n')
	}
	var inlineIndexes []index
	var indexes []index
	for _, idx := range t.Indexes {
		if idx.KeyType == indexKey {
			indexes = append(indexes, idx)
		} else {
			inlineIndexes = append(inlineIndexes, idx)
		}
	}

	for i, idx := range inlineIndexes {
		switch idx.KeyType {
		case indexKey:
			continue
		}
		var cols []string
		for _, col := range idx.Columns {
			cols = append(cols, fmt.Sprintf(`\"%s\"`, col))
		}
		last := ", "
		if i == len(inlineIndexes)-1 {
			last = ""
		}
		fmt.Fprintf(&buf, `    "%s(%s)%s" +`, idx.KeyType, strings.Join(cols, ", "), last)
		buf.WriteRune('\n')
	}
	buf.WriteString("  \");\",\n")
	for _, idx := range indexes {
		var cols []string
		for _, col := range idx.Columns {
			cols = append(cols, fmt.Sprintf(`\"%s\"`, col))
		}
		fmt.Fprintf(&buf, `  "CREATE INDEX \"%s\" ON \"%s\" (%s);",`,
			idx.Name, t.Name, strings.Join(cols, ", "))
		buf.WriteRune('\n')
	}
	return buf.String()
}

func (t table) FindString() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, `func (j Job) Find%s(opts db.Opts) (rows []%s, err error) {`, t.Name, t.GoType)
	buf.WriteRune('\n')
	fmt.Fprintf(&buf, "\terr = j.Scan%s(opts, func(data %s) error {\n", t.Name, t.GoType)
	buf.WriteString("\t\trows = append(rows, data)\n")
	buf.WriteString("\t\treturn nil\n")
	buf.WriteString("\t})\n")
	buf.WriteString("return\n")
	buf.WriteString("}")

	return buf.String()
}

func (t table) ScanString() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, `func (j Job) Scan%s(opts db.Opts, cb func(%s) error) error {`, t.Name, t.GoType)
	buf.WriteRune('\n')
	fmt.Fprintf(&buf, "\t"+`return db.Scan(j.Tx, "%s", opts, func(data []byte) error {`, t.Name)
	buf.WriteRune('\n')
	fmt.Fprintf(&buf, "\t\t"+`var pb %s`, t.GoType)
	buf.WriteRune('\n')
	buf.WriteString("\t\tif err := proto.Unmarshal(data, &pb); err != nil {\n")
	buf.WriteString("\t\t\treturn err\n")
	buf.WriteString("\t\t}\n")
	buf.WriteString("\treturn cb(pb)\n")
	buf.WriteString("})\n")
	buf.WriteString("}\n")
	return buf.String()
}

func run(req *plugin.CodeGeneratorRequest) (*plugin.CodeGeneratorResponse, error) {
	if len(req.FileToGenerate) != 1 {
		return nil, errors.New("Can only generate 1 file.")
	}
	var tables []table
	for _, fd := range req.ProtoFile {
		if fd.GetOptions().GetGoPackage() == "" {
			return nil, errors.New("Must have a Go package")
		}
		for _, msg := range fd.MessageType {
			if msg.Options == nil || !proto.HasExtension(msg.Options, model.E_TabOpts) {
				continue
			}
			opts, err := proto.GetExtension(msg.Options, model.E_TabOpts)
			if err != nil {
				return nil, err
			}
			t, err := buildTable(msg, opts.(*model.TableOptions))
			if err != nil {
				return nil, err
			}
			tables = append(tables, t)
		}
	}

	temp := template.Must(template.New("inline").Parse(tpl))
	var buf bytes.Buffer
	temp.Execute(&buf, tables)
	resp := new(plugin.CodeGeneratorResponse)
	resp.File = []*plugin.CodeGeneratorResponse_File{{
		Name:    proto.String(strings.Replace(req.FileToGenerate[0], ".proto", ".tab.go", -1)),
		Content: proto.String(buf.String()),
	}}

	return resp, nil
}

func getRequest() (*plugin.CodeGeneratorRequest, error) {
	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return nil, err
	}

	req := new(plugin.CodeGeneratorRequest)
	if err := proto.Unmarshal(data, req); err != nil {
		return nil, err
	}

	return req, nil
}

func buildTable(msg *descriptor.DescriptorProto, opts *model.TableOptions) (table, error) {
	t := table{
		msg: msg,
	}

	if opts.Name != "" {
		t.Name = opts.Name
	} else {
		t.Name = *msg.Name
	}
	if strings.ContainsAny(t.Name, "\"\\") {
		return t, errors.New("Invalid characters in table name " + t.Name)
	}
	fieldNames := make(map[string]bool, len(msg.Field))
	for _, f := range msg.Field {
		var typ columnType
		switch *f.Type {
		case descriptor.FieldDescriptorProto_TYPE_FIXED32:
			fallthrough
		case descriptor.FieldDescriptorProto_TYPE_SFIXED32:
			fallthrough
		case descriptor.FieldDescriptorProto_TYPE_SINT32:
			fallthrough
		case descriptor.FieldDescriptorProto_TYPE_ENUM:
			fallthrough
		case descriptor.FieldDescriptorProto_TYPE_INT32:
			typ = int32Column
		case descriptor.FieldDescriptorProto_TYPE_FIXED64:
			fallthrough
		case descriptor.FieldDescriptorProto_TYPE_SFIXED64:
			fallthrough
		case descriptor.FieldDescriptorProto_TYPE_SINT64:
			fallthrough
		case descriptor.FieldDescriptorProto_TYPE_INT64:
			typ = int64Column
		case descriptor.FieldDescriptorProto_TYPE_BOOL:
			typ = int16Column
		case descriptor.FieldDescriptorProto_TYPE_STRING:
			fallthrough
		case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
			fallthrough
		case descriptor.FieldDescriptorProto_TYPE_BYTES:
			typ = bytesColumn
		default:
			return t, errors.New("No type for  " + f.Type.String())
		}
		fieldNames[*f.Name] = true
		t.Columns = append(t.Columns, column{
			Name:       *f.Name,
			ColumnType: typ,
			field:      f,
		})

		if *f.Name == "data" {
			parts := strings.Split(*f.TypeName, ".")
			t.GoType = "schema." + parts[len(parts)-1]
		}
	}
	if !fieldNames["data"] {
		return t, errors.New("Missing data col on table " + t.Name)
	}
	for _, k := range opts.Key {
		if len(k.Col) == 0 {
			return t, errors.New("No cols in key on table " + t.Name)
		}
		for _, c := range k.Col {
			if !fieldNames[c] {
				return t, errors.New("Unknown col on table " + t.Name)
			}
		}
		name := k.Name
		var typ keyType
		switch k.KeyType {
		case model.KeyType_PRIMARY:
			typ = primaryKey
			if k.Name == "" {
				name = "Primary"
			}
		case model.KeyType_UNIQUE:
			typ = uniqueKey
			if k.Name == "" {
				return t, errors.New("Missing name for key on table " + t.Name)
			}
		case model.KeyType_INDEX:
			typ = indexKey
			if k.Name == "" {
				return t, errors.New("Missing name for key on table " + t.Name)
			}
		default:
			return t, errors.New("Unknown key type on table " + t.Name)
		}
		t.Indexes = append(t.Indexes, index{
			Name:    name,
			KeyType: typ,
			Columns: k.Col,
		})
	}

	return t, nil
}

func main() {
	req, err := getRequest()
	if err != nil {
		log.Fatalln(err)
	}

	resp, err := run(req)
	if err != nil {
		resp = new(plugin.CodeGeneratorResponse)
		resp.Error = proto.String(err.Error())
	}
	if data, err := proto.Marshal(resp); err != nil {
		log.Fatalln(err)
	} else {
		if _, err := os.Stdout.Write(data); err != nil {
			log.Fatalln(err)
		}
	}
}
