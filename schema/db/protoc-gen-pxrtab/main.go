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
{{end -}}
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
	fmt.Fprintf(&buf, `func (j Job) Find%s(opts Opts) ([]%s, error) {`, t.Name, t.GoType)
	buf.WriteRune('\n')
	fmt.Fprintf(&buf, "\t"+`return db.Scan(j.Tx, "%s", opts, func(data []byte) error {`, t.Name)
	buf.WriteRune('\n')
	fmt.Fprintf(&buf, "\t\t"+`var pb %s`, t.GoType)
	buf.WriteRune('\n')
	buf.WriteString("\t\tif err := proto.Unmarshal(data, &pb); err != nil {\n")
	buf.WriteString("\t\t\treturn err\n")
	buf.WriteString("\t\t}\n")
	buf.WriteString("\treturn cb(pb)\n")
	buf.WriteRune('}')
	return buf.String()
}

func (t table) ScanString() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, `func (j Job) Scan%s(opts Opts, cb func(%s) error) error {`, t.Name, t.GoType)
	buf.WriteRune('\n')
	fmt.Fprintf(&buf, "\t"+`return db.Scan(j.Tx, "%s", opts, func(data []byte) error {`, t.Name)
	buf.WriteRune('\n')
	fmt.Fprintf(&buf, "\t\t"+`var pb %s`, t.GoType)
	buf.WriteRune('\n')
	buf.WriteString("\t\tif err := proto.Unmarshal(data, &pb); err != nil {\n")
	buf.WriteString("\t\t\treturn err\n")
	buf.WriteString("\t\t}\n")
	buf.WriteString("\treturn cb(pb)\n")
	buf.WriteRune('}')
	return buf.String()
}

func run() error {
	fds, err := getRequest()
	if err != nil {
		return err
	}

	log.Println(fds.String())

	var tables []table
	for _, fd := range fds.ProtoFile {
		for _, msg := range fd.MessageType {
			if msg.Options == nil || !proto.HasExtension(msg.Options, model.E_TabOpts) {
				continue
			}
			opts, err := proto.GetExtension(msg.Options, model.E_TabOpts)
			if err != nil {
				return err
			}
			t, err := buildTable(msg, opts.(*model.TableOptions))
			if err != nil {
				return err
			}
			tables = append(tables, t)
		}
	}

	temp := template.Must(template.New("inline").Parse(tpl))
	temp.Execute(os.Stderr, tables)
	return nil
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

	log.Fatal(*msg)

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
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}
