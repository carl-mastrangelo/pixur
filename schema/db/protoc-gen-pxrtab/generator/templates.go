package generator

import (
	"text/template"
)

var (
	noopFunc   = func(_ string) interface{} { return nil }
	dummyFuncs = template.FuncMap{
		"goesc":         noopFunc,
		"sqlesc":        noopFunc,
		"sqlblobidxesc": noopFunc}
	tpl = template.Must(template.New("").Funcs(dummyFuncs).Parse(`
{{template "package" .Name}}
{{template "imports" .Imports}}
{{template "sql" .Tables}}
{{template "defaults"}}
{{template "tables" .Tables}}
`))

	_ = template.Must(tpl.New("package").Parse(`
package {{.}}
`))
	_ = template.Must(tpl.New("imports").Parse(`
import (
  {{range .}}
    {{if .ShortName}}
      {{.ShortName}} "{{.FullName}}"
    {{else}}
      "{{.FullName}}"
    {{end}}
  {{end}}
)
var (
{{range .}}
  {{if .Dummy}}
  _ = {{.Dummy}}
  {{end}}
{{end}}
)
`))
	_ = template.Must(tpl.New("sql").Parse(`
var SqlTables = []string{
  {{range .}}
    {{template "sqltable" . }}
  {{end}}
}
`))
	_ = template.Must(tpl.New("tables").Parse(`
{{range .}}
  {{range .Indexes}}
    {{template "index" .}}
  {{end}}
  {{template "cols" .}}
  {{template "scanfunc" .}}
  {{template "findfunc" .}}
  {{template "insertfunc" .}}
  {{template "deletefunc" .}}
{{end}}
`))
	_ = template.Must(tpl.New("index").Parse(`
type {{.Name}} struct {
  {{range .Columns}}
    {{.GoName}} *{{.GoType}}
  {{end}}
}

{{if eq .KeyType "PRIMARY KEY" }}
func (_ {{.Name}}) Unique() {}
var _ db.UniqueIdx = {{.Name}}{}
{{else if eq .KeyType "UNIQUE" }}
func (_ {{.Name}}) Unique() {}
var _ db.UniqueIdx = {{.Name}}{}
{{else}}
var _ db.Idx = {{.Name}}{}
{{end}}

{{template "cols" .}}
func (idx {{.Name}}) Cols() []string {
  return cols{{.Name}}
}

func (idx {{.Name}}) Vals() (vals []interface{}) {
  var done bool
  {{range .Columns}}
  if idx.{{.GoName}} != nil {
		if done {
			panic("Extra value {{.GoName}}")
		}
		vals = append(vals, *idx.{{.GoName}})
	} else {
		done = true
	}
  {{end}}
  return
}
`))

	_ = template.Must(tpl.New("defaults").Parse(`
type Job struct {
  Tx *sql.Tx
}

func (j Job) Exec(query string, args ...interface{}) (db.Result, error) {
  res, err := j.Tx.Exec(query, args...)
  return db.Result(res), err
}

func (j Job) Query(query string, args ...interface{}) (db.Rows, error) {
  rows, err := j.Tx.Query(query, args...)
  return db.Rows(rows), err
}
`))
	_ = template.Must(tpl.New("scanfunc").Parse(`
func (j Job) Scan{{.Name}}(opts db.Opts, cb func({{.GoDataType}}) error) error {
	return db.Scan(j, {{goesc .Name}}, opts, func(data []byte) error {
		var pb {{.GoDataType}}
		if err := proto.Unmarshal(data, &pb); err != nil {
			return err
		}
		return cb(pb)
	}, cols{{.Name}})
}
`))
	_ = template.Must(tpl.New("cols").Parse(`
var cols{{.Name}} = []string{ {{- range .Columns}} {{goesc .SqlName}}, {{end -}} }
`))
	_ = template.Must(tpl.New("findfunc").Parse(`
func (j Job) Find{{.Name}}(opts db.Opts) (rows []{{.GoDataType}}, err error) {
	err = j.Scan{{.Name}}(opts, func(data {{.GoDataType}}) error {
		rows = append(rows, data)
		return nil
	})
	return
}
`))
	_ = template.Must(tpl.New("insertfunc").Parse(`
func (j Job) Insert{{.Name}}(row {{.GoType}}) error {
	vals := []interface{}{ {{- range .Columns}} row.{{.GoName}}, {{end -}} }
	return db.Insert(j, {{goesc .Name}}, cols{{.Name}}, vals)
}
`))
	_ = template.Must(tpl.New("deletefunc").Parse(`
func (j Job) Delete{{.Name}}(key {{.Name}}Primary) error {
	return db.Delete(j, {{goesc .Name}}, key)
}
`))

	_ = template.Must(tpl.New("sqltable").Parse(`
  {{$tableName := .Name}}
  "CREATE TABLE {{sqlesc .Name}} (" +
{{range .Columns}}
    "{{sqlesc .SqlName}} {{.SqlType}} NOT NULL, " +
{{end}}
{{range .Indexes}}
  {{if eq .KeyType "UNIQUE"}}
    "UNIQUE( {{- range $i, $_ := .Columns -}} 
      {{if ne $i 0}},{{end -}}
        {{- if .IsBlobIdxCol}}{{sqlblobidxesc .SqlName}}{{else}}{{sqlesc .SqlName}}{{end -}}
      {{- end -}} ), " +
  {{end}}
{{end}}
{{range .Indexes}}
  {{if eq .KeyType "PRIMARY KEY"}}
    "PRIMARY KEY( {{- range $i, $_ := .Columns -}} 
      {{if ne $i 0}},{{end}}
      {{- if .IsBlobIdxCol}}{{sqlblobidxesc .SqlName}}{{else}}{{sqlesc .SqlName}}{{end -}}
      {{- end -}} )" +
  {{end}}
{{end}}
  ");",
{{range .Indexes}}
  {{if eq .KeyType "INDEX"}}
    "CREATE INDEX {{sqlesc .Name}} ON {{sqlesc $tableName}} ( {{- range $i, $_ := .Columns -}} 
      {{if ne $i 0}},{{end -}}
        {{- if .IsBlobIdxCol}}{{sqlblobidxesc .SqlName}}{{else}}{{sqlesc .SqlName}}{{end -}}
      {{- end -}} );",
  {{end}}
{{end}}
`))
)
