package generator

import (
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"

	"github.com/golang/protobuf/proto"
	descriptor "google/protobuf"
	plugin "google/protobuf/compiler"

	"pixur.org/pixur/schema/db/model"
)

type file struct {
	name    string
	pack    string
	imports deps
	packmap map[string]string
	tables  []table
}

type table struct {
	name       string
	gotype     string
	datagotype string
	columns    []*column
	indexes    []index
	msg        *descriptor.DescriptorProto
}

type coltype struct {
	sqltype string
	gotype  string
}

type column struct {
	name   string
	coltyp coltype
	field  *descriptor.FieldDescriptorProto
}

type index struct {
	name    string
	keyType keyType
	columns []*column
}

type deps []string

func (d deps) Less(i, k int) bool {
	left := strings.Split(d[i], " ")
	right := strings.Split(d[k], " ")
	return strings.Compare(left[1], right[1]) < 0
}

func (d deps) Len() int {
	return len(d)
}

func (d deps) Swap(i, k int) {
	d[k], d[i] = d[i], d[k]
}

type keyType string

var (
	primaryKey keyType = "PRIMARY KEY"
	uniqueKey  keyType = "UNIQUE"
	indexKey   keyType = "INDEX"
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
	var f file
	f.packmap = make(map[string]string)
	nameToDesc := make(map[string]*descriptor.FileDescriptorProto)
	for i, fd := range fds {
		if fd.GetOptions().GetGoPackage() == "" {
			return "", fmt.Errorf("%s doesn't have go_package", *fd.Name)
		}
		nameToDesc[*fd.Name] = fd
		if i == len(fds)-1 {
			// Make ourself the empty package
			f.packmap[*fd.Package] = ""
		} else {
			f.packmap[*fd.Package] = fd.GetOptions().GetGoPackage()
		}

		for _, msg := range fd.MessageType {
			if msg.Options == nil || !proto.HasExtension(msg.Options, model.E_TabOpts) {
				continue
			}
			opts, err := proto.GetExtension(msg.Options, model.E_TabOpts)
			if err != nil {
				return "", err
			}
			t, err := buildTable(msg, opts.(*model.TableOptions), f.packmap)
			if err != nil {
				return "", err
			}

			f.tables = append(f.tables, t)
		}
	}

	// plugin.proto says files are ordered by dependency, and we only allow one source file,
	// so it must be last.

	f.pack = fds[len(fds)-1].GetOptions().GetGoPackage()
	for _, dep := range fds[len(fds)-1].Dependency {
		fd, ok := nameToDesc[dep]
		if !ok {
			panic("Missing file " + dep)
		}
		// hack to exclude model.proto
		// TODO: check which messages are actually referenced, and import appropriately.
		if *fd.Package == "pixur.db.model" {
			continue
		}
		imp := fmt.Sprintf(`%s "%s"`,
			filepath.Base(fd.GetOptions().GetGoPackage()), filepath.Dir(dep))
		f.imports = append(f.imports, imp)
	}

	return renderTables(f), nil
}

func renderTables(f file) string {
	w := &indentWriter{}
	renderPackage(w, f)
	w.writeln("")
	renderImports(w, f)
	w.writeln("")
	renderSqlTables(w, f)
	w.writeln("")
	renderIndexes(w, f)
	w.writeln("")
	renderDefaultTypes(w, f)
	w.writeln("")
	renderScanFuncs(w, f)
	renderFindFuncs(w, f)
	renderInsertFuncs(w, f)
	renderDeleteFuncs(w, f)

	return w.String()
}

func renderDefaultTypes(w *indentWriter, f file) {
	w.writeln("type Job struct {")
	w.in()
	w.writeln("Tx *sql.Tx")
	w.out()
	w.writeln("}")
	w.writeln("")

	w.writeln("func (j Job) Exec(query string, args ...interface{}) (db.Result, error) {")
	w.in()
	w.writeln("res, err := j.Tx.Exec(query, args...)")
	w.writeln("return db.Result(res), err")
	w.out()
	w.writeln("}")
	w.writeln("")

	w.writeln("func (j Job) Query(query string, args ...interface{}) (db.Rows, error) {")
	w.in()
	w.writeln("rows, err := j.Tx.Query(query, args...)")
	w.writeln("return db.Rows(rows), err")
	w.out()
	w.writeln("}")
	w.writeln("")
}

func renderScanFuncs(w *indentWriter, f file) {
	for _, t := range f.tables {
		w.writefln(`func (j Job) Scan%s(opts db.Opts, cb func(%s) error) error {`, t.name, t.datagotype)
		w.in()
		var cols []string
		for _, col := range t.columns {
			cols = append(cols, `"`+col.name+`"`)
		}
		w.writefln(`cols := []string{%s}`, strings.Join(cols, ", "))
		w.writefln(`return db.Scan(j, "%s", opts, func(data []byte) error {`, t.name)
		w.in()
		w.writefln(`var pb %s`, t.datagotype)
		w.writeln("if err := proto.Unmarshal(data, &pb); err != nil {")
		w.in()
		w.writeln("return err")
		w.out()
		w.writeln("}")
		w.writeln("return cb(pb)")
		w.out()
		w.writeln("}, cols)")
		w.out()
		w.writeln("}")
		w.writeln("")
	}
}

func renderDeleteFuncs(w *indentWriter, f file) {
	for _, t := range f.tables {
		w.writefln(`func (j Job) Delete%s(key %sPrimary) error {`, t.gotype, t.name)
		w.in()
		w.writefln(`return db.Delete(j, "%s", key)`, t.name)
		w.out()
		w.writeln("}")
		w.writeln("")
	}
}

func renderInsertFuncs(w *indentWriter, f file) {
	for _, t := range f.tables {
		w.writefln(`func (j Job) Insert%s(row %s) error {`, t.gotype, t.gotype)
		w.in()
		var vals []string
		var cols []string
		for _, col := range t.columns {
			cols = append(cols, `"`+col.name+`"`)
			vals = append(vals, "row."+colNameToGoName(col.name))
		}
		w.writefln(`cols := []string{%s}`, strings.Join(cols, ", "))
		w.writefln(`vals := []interface{}{%s}`, strings.Join(vals, ", "))
		w.writefln(`return db.Insert(j, "%s", cols, vals)`, t.name)
		w.out()
		w.writeln("}")
		w.writeln("")
	}
}

func renderFindFuncs(w *indentWriter, f file) {
	for _, t := range f.tables {
		w.writefln(`func (j Job) Find%s(opts db.Opts) (rows []%s, err error) {`, t.name, t.datagotype)
		w.in()
		w.writefln(`err = j.Scan%s(opts, func(data %s) error {`, t.name, t.datagotype)
		w.in()
		w.writeln(`rows = append(rows, data)`)
		w.writeln(`return nil`)
		w.out()
		w.writeln(`})`)
		w.writeln(`return`)
		w.out()
		w.writeln("}")
		w.writeln("")
	}
}

func renderPackage(w *indentWriter, f file) {
	w.writeln("package " + f.pack)
}

func renderIndexes(w *indentWriter, f file) {
	for _, t := range f.tables {
		for _, idx := range t.indexes {
			if idx.keyType == uniqueKey || idx.keyType == primaryKey {
				w.writefln(`var _ db.UniqueIdx = %s{}`, idx.name)
			} else {
				w.writefln(`var _ db.Idx = %s{}`, idx.name)
			}
			w.writeln("")
			w.writefln(`type %s struct {`, idx.name)
			w.in()
			for _, c := range idx.columns {
				w.writefln(`%s *%s`, colNameToGoName(c.name), c.coltyp.gotype)
			}
			w.out()
			w.writeln("}")
			w.writeln("")
			if idx.keyType == uniqueKey || idx.keyType == primaryKey {
				w.writefln(`func (_ %s) Unique() {}`, idx.name)
				w.writeln("")
			}
			w.writefln(`func (idx %s) Cols() []string {`, idx.name)
			var escaped []string
			for _, c := range idx.columns {
				escaped = append(escaped, `"`+c.name+`"`)
			}
			w.in()
			w.writefln(`return []string{%s}`, strings.Join(escaped, ", "))
			w.out()
			w.writeln("}")
			w.writeln("")
			w.writefln(`func (idx %s) Vals() (vals []interface{}) {`, idx.name)
			w.in()
			w.writeln("var done bool")
			for _, c := range idx.columns {
				w.writefln("if idx.%s != nil {", colNameToGoName(c.name))
				w.in()
				w.writeln("if done {")
				w.in()
				w.writefln(`panic("Extra value %s")`, colNameToGoName(c.name))
				w.out()
				w.writeln("}")
				w.writefln("vals = append(vals, *idx.%s)", colNameToGoName(c.name))
				w.out()
				w.writeln("} else {")
				w.in()
				w.writeln("done = true")
				w.out()
				w.writeln("}")
			}
			w.writeln("return")
			w.out()
			w.writeln("}")
			w.writeln("")
		}
	}
}

func colNameToGoName(name string) string {
	parts := strings.Split(name, "_")
	for i := 0; i < len(parts); i++ {
		parts[i] = strings.Title(parts[i])
	}
	return strings.Join(parts, "")
}

func renderImports(w *indentWriter, f file) {
	sort.Sort(deps(f.imports))
	w.writeln("import (")
	w.in()
	w.writeln(`"database/sql"`)
	w.out()
	w.writeln("")
	w.in()
	w.writeln(`"github.com/golang/protobuf/proto"`)
	w.out()
	w.writeln("")
	w.in()
	w.writeln(`"pixur.org/pixur/schema/db"`)
	w.out()
	w.writeln("")
	w.in()
	for _, imp := range f.imports {
		w.writeln(imp)
	}
	w.out()
	w.writeln(")")
}

func renderSqlTables(w *indentWriter, f file) {
	w.writeln("var SqlTables = []string{")
	w.in()
	for _, t := range f.tables {
		w.writefln(`"CREATE TABLE \"%s\" (" +`, t.name)
		w.in()
		for _, col := range t.columns {
			w.writefln(`"\"%s\" %s NOT NULL, " +`, col.name, col.coltyp.sqltype)
		}
		var inlineIndexes []index
		var indexes []index
		for _, idx := range t.indexes {
			if idx.keyType == indexKey {
				indexes = append(indexes, idx)
			} else {
				inlineIndexes = append(inlineIndexes, idx)
			}
		}

		for i, idx := range inlineIndexes {
			switch idx.keyType {
			case indexKey:
				continue
			}
			var cols []string
			for _, col := range idx.columns {
				cols = append(cols, fmt.Sprintf(`\"%s\"`, col.name))
			}
			last := ", "
			if i == len(inlineIndexes)-1 {
				last = ""
			}
			w.writefln(`"%s(%s)%s" +`, idx.keyType, strings.Join(cols, ", "), last)
		}
		w.writeln(`");",`)
		w.out()
		for _, idx := range indexes {
			var cols []string
			for _, col := range idx.columns {
				cols = append(cols, fmt.Sprintf(`\"%s\"`, col.name))
			}
			w.writefln(`"CREATE INDEX \"%s\" ON \"%s\" (%s);",`,
				idx.name, t.name, strings.Join(cols, ", "))
		}
	}

	w.out()
	w.writeln("}")
}

func buildTable(msg *descriptor.DescriptorProto, opts *model.TableOptions,
	packmap map[string]string) (table, error) {
	t := table{}
	if opts.Name != "" {
		t.name = opts.Name
	} else {
		t.name = *msg.Name
	}
	t.gotype = *msg.Name
	if strings.ContainsAny(t.name, `\"`) {
		return t, fmt.Errorf("Invalid characters in table name %s", t.name)
	}
	fieldNames := make(map[string]*column, len(msg.Field))
	for _, f := range msg.Field {
		var coltyp coltype
		switch *f.Type {
		case descriptor.FieldDescriptorProto_TYPE_FIXED32:
			fallthrough
		case descriptor.FieldDescriptorProto_TYPE_SFIXED32:
			fallthrough
		case descriptor.FieldDescriptorProto_TYPE_SINT32:
			fallthrough
		case descriptor.FieldDescriptorProto_TYPE_ENUM:
			coltyp = coltype{
				sqltype: "integer",
				gotype:  typeNameToGoName(*f.TypeName, packmap),
			}
		case descriptor.FieldDescriptorProto_TYPE_INT32:
			coltyp = coltype{
				sqltype: "integer",
				gotype:  "int32",
			}
		case descriptor.FieldDescriptorProto_TYPE_FIXED64:
			fallthrough
		case descriptor.FieldDescriptorProto_TYPE_SFIXED64:
			fallthrough
		case descriptor.FieldDescriptorProto_TYPE_SINT64:
			fallthrough
		case descriptor.FieldDescriptorProto_TYPE_INT64:
			coltyp = coltype{
				sqltype: "bigint",
				gotype:  "int64",
			}
		case descriptor.FieldDescriptorProto_TYPE_BOOL:
			coltyp = coltype{
				sqltype: "boolean",
				gotype:  "bool",
			}
		case descriptor.FieldDescriptorProto_TYPE_STRING:
			coltyp = coltype{
				sqltype: "bytea",
				gotype:  "string",
			}
		case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
			coltyp = coltype{
				sqltype: "bytea",
				gotype:  typeNameToGoName(*f.TypeName, packmap),
			}
		case descriptor.FieldDescriptorProto_TYPE_BYTES:
			coltyp = coltype{
				sqltype: "bytea",
				gotype:  "[]byte",
			}
		default:
			return t, fmt.Errorf("No type for %s", f.Type)
		}
		col := &column{
			name:   *f.Name,
			coltyp: coltyp,
			field:  f,
		}
		fieldNames[*f.Name] = col
		t.columns = append(t.columns, col)
		if *f.Name == "data" {
			t.datagotype = typeNameToGoName(*f.TypeName, packmap)
		}
	}
	if fieldNames["data"] == nil {
		return t, fmt.Errorf("Missing data col on table %s", t.name)
	}

	for _, k := range opts.Key {
		if len(k.Col) == 0 {
			return t, fmt.Errorf("No cols in key on table %s", t.name)
		}
		var cols []*column
		for _, c := range k.Col {
			if fieldNames[c] == nil {
				return t, fmt.Errorf("Unknown col on table %s", t.name)
			}
			cols = append(cols, fieldNames[c])
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
				return t, fmt.Errorf("Missing name for key on table %s", t.name)
			}
		case model.KeyType_INDEX:
			typ = indexKey
			if k.Name == "" {
				return t, fmt.Errorf("Missing name for key on table %s", t.name)
			}
		default:
			return t, fmt.Errorf("Unknown key type on table %s", t.name)
		}
		t.indexes = append(t.indexes, index{
			name:    t.name + name,
			keyType: typ,
			columns: cols,
		})
	}

	return t, nil
}

func typeNameToGoName(tn string, packmap map[string]string) string {
	var best string
	for k := range packmap {
		if strings.HasPrefix(tn, "."+k+".") {
			if len(k) > len(best) {
				best = k
			}
		}
	}
	if best != "" {
		msg := strings.TrimPrefix(tn, "."+best+".")
		if gopackage := packmap[best]; gopackage != "" {
			return gopackage + "." + strings.Join(strings.Split(msg, "."), "_")
		}
		return strings.Join(strings.Split(msg, "."), "_")
	}
	panic("Could not find type!" + tn)
}
