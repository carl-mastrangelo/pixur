package generator

import (
	"text/template"
)

var (
	noopFunc   = func(_ string) interface{} { return nil }
	dummyFuncs = template.FuncMap{
		"goesc":    noopFunc,
		"gostresc": noopFunc}
	tpl = template.Must(template.New("").Funcs(dummyFuncs).Parse(`
{{template "package" .Name}}
{{template "imports" .Imports}}
{{template "sql" .}}
{{template "defaults"}}
{{template "tables" .Tables}}
`))

	_ = template.Must(tpl.New("package").Parse(`
package {{.}}
`))
	_ = template.Must(tpl.New("imports").Parse(`
import (
  "runtime"
  "log"
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
	{{$seqTabName := .SequenceTableName}}
	{{$seqColName := .SequenceColName}}
	var SqlTables = map[string][]string {
	{{$tabs := .Tables}}
	{{range .Adapters}}
	  {{$adapter := .}}
		"{{.Name}}": {
			{{range $tabs}}
				{{/***********************************************************************/}}

					{{$tableName := $adapter.Quote .Name}}
					"CREATE TABLE {{gostresc $tableName}} (" +
				{{range .Columns}}
					{{$name := $adapter.Quote .SqlName}}
					"{{gostresc $name}} {{.SqlTypeString $adapter}} NOT NULL, " +
				{{end}}
				{{range .Indexes}}
					{{if eq .KeyType "UNIQUE"}}
						"UNIQUE( {{- range $i, $_ := .Columns -}} 
							{{if ne $i 0}},{{end -}}
								{{- if .IsBlobIdxCol -}}
									{{- $name := $adapter.BlobIdxQuote .SqlName -}}
									{{- gostresc $name -}}
								{{- else -}}
									{{- $name := $adapter.Quote .SqlName -}}
									{{- gostresc $name -}}
								{{- end -}}
							{{- end -}} ), " +
					{{end}}
				{{end}}
				{{range .Indexes}}
					{{if eq .KeyType "PRIMARY KEY"}}
						"PRIMARY KEY( {{- range $i, $_ := .Columns -}} 
							{{if ne $i 0}},{{end}}
								{{- if .IsBlobIdxCol -}}
									{{- $name := $adapter.BlobIdxQuote .SqlName -}}
									{{- gostresc $name -}}
								{{- else -}}
									{{- $name := $adapter.Quote .SqlName -}}
									{{- gostresc $name -}}
								{{- end -}}
							{{- end -}} )" +
					{{end}}
				{{end}}
					");",
				{{range .Indexes}}
					{{if eq .KeyType "INDEX"}}
						{{$indexName := $adapter.Quote .Name}}
						"CREATE INDEX {{gostresc $indexName}} ON {{gostresc $tableName}} ( {{- range $i, $_ := .Columns -}} 
							{{if ne $i 0}},{{end -}}
								{{- if .IsBlobIdxCol -}}
									{{- $name := $adapter.BlobIdxQuote .SqlName -}}
									{{- gostresc $name -}}
								{{- else -}}
									{{- $name := $adapter.Quote .SqlName -}}
									{{- gostresc $name -}}
								{{- end -}}
							{{- end -}} );",
					{{end}}
				{{end}}
				
				{{/***********************************************************************/}}

			{{end}}
			{{$seqTabNameEsc := .Quote $seqTabName}}
			{{$seqColNameEsc := .Quote $seqColName}}
			"CREATE TABLE {{gostresc $seqTabNameEsc}} ({{gostresc $seqColNameEsc}} {{.BigIntType}} NOT NULL);",
		},
	{{end}}
	}
	
	var SqlInitTables = map[string][]string {
	{{range .Adapters}}
		{{$seqTabNameEsc := .Quote $seqTabName}}
		{{$seqColNameEsc := .Quote $seqColName}}
		"{{.Name}}": {
			"INSERT INTO {{gostresc $seqTabNameEsc}} ({{gostresc $seqColNameEsc}}) VALUES (1);",
		},
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
  {{template "insertrowfunc" .}}
  {{template "updatefunc" .}}
  {{template "updaterowfunc" .}}
  {{template "deletefunc" .}}
{{end}}
`))
	_ = template.Must(tpl.New("index").Parse(`
type {{.Name}} struct {
  {{range .Columns}}
    {{.GoName}} *{{.GoType}}
  {{end}}
}

{{if eq .KeyType "PRIMARY KEY"}}
func (_ {{.Name}}) Unique() {}
var _ db.UniqueIdx = {{.Name}}{}
{{else if eq .KeyType "UNIQUE"}}
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
func NewJob(DB *sql.DB) (*Job, error) {
  tx, err := DB.Begin()
  if err != nil {
    return nil, err
  }
  j := &Job{
    db: dbWrapper{DB},
    tx: txWrapper{tx},
  }
  runtime.SetFinalizer(j, jobCloser)
  return j, nil
}
  
type Job struct {
  db db.QuerierExecutorBeginner
  tx db.QuerierExecutorCommitter
}

func (j *Job) Commit() error {
  defer runtime.SetFinalizer(j, nil)
  return j.tx.Commit()
}

func (j *Job) Rollback() error {
  defer runtime.SetFinalizer(j, nil)
  return j.tx.Rollback()
}

var jobCloser = func(j *Job) {
  log.Println("warning: found orphaned job")
  if err := j.Rollback(); err != nil {
    log.Println("error rolling back orphaned job", err)
  }
}

type dbWrapper struct {
  db *sql.DB
}

func (w dbWrapper) Begin() (db.QuerierExecutorCommitter, error) {
  tx, err := w.db.Begin()
  return txWrapper{tx}, err
}

type txWrapper struct {
  tx *sql.Tx
}

func (w txWrapper) Exec(query string, args ...interface{}) (db.Result, error) {
  res, err := w.tx.Exec(query, args...)
  return db.Result(res), err
}

func (w txWrapper) Query(query string, args ...interface{}) (db.Rows, error) {
  rows, err := w.tx.Query(query, args...)
  return db.Rows(rows), err
}

func (w txWrapper) Commit() error {
  return w.tx.Commit()
}

func (w txWrapper) Rollback() error {
  return w.tx.Rollback()
}

var alloc db.IDAlloc

func (j *Job) AllocID() (int64, error) {
  return db.AllocID(j.db, &alloc)
}
`))
	_ = template.Must(tpl.New("scanfunc").Parse(`
func (j *Job) Scan{{.Name}}(opts db.Opts, cb func(*{{.GoDataType}}) error) error {
	return db.Scan(j.tx, {{goesc .Name}}, opts, func(data []byte) error {
		var pb {{.GoDataType}}
		if err := proto.Unmarshal(data, &pb); err != nil {
			return err
		}
		return cb(&pb)
	})
}
`))
	_ = template.Must(tpl.New("cols").Parse(`
var cols{{.Name}} = []string{ {{- range .Columns}} {{goesc .SqlName}}, {{end -}} }
`))
	_ = template.Must(tpl.New("findfunc").Parse(`
func (j *Job) Find{{.Name}}(opts db.Opts) (rows []*{{.GoDataType}}, err error) {
	err = j.Scan{{.Name}}(opts, func(data *{{.GoDataType}}) error {
		rows = append(rows, data)
		return nil
	})
	return
}
`))
	_ = template.Must(tpl.New("insertfunc").Parse(`
{{if .HasColFns}}
  {{$goDataType := .GoDataType}}
  {{range .Columns}}
    {{if .IsProto}}{{else}}
      var _ interface{ {{- .ColFn}}() {{.GoType -}} } = (*{{$goDataType}})(nil)
    {{end}}
  {{end}}

  func (j *Job) Insert{{.GoDataTypeShort}}(pb *{{$goDataType}}) error {
    return j.Insert{{.GoType}}(&{{.GoType}} {
      Data: pb,
      {{range .Columns}}
        {{if .IsProto}}{{else}}
          {{.GoName}}: pb.{{- .ColFn}}(),
        {{end}}
      {{end}}
    })
  }
{{end}}
`))
	_ = template.Must(tpl.New("insertrowfunc").Parse(`
func (j *Job) Insert{{.GoType}}(row *{{.GoType}}) error {
  var vals []interface{}
  {{range .Columns}}
    {{if .IsProto}}
      if val, err := proto.Marshal(row.{{.GoName}}); err != nil {
        return err
      } else {
        vals = append(vals, val)
      }
    {{else}}
      vals = append(vals, row.{{.GoName}})
    {{end}}
  {{end}}

	return db.Insert(j.tx, {{goesc .Name}}, cols{{.Name}}, vals)
}
`))
	_ = template.Must(tpl.New("deletefunc").Parse(`
func (j *Job) Delete{{.GoDataTypeShort}}(key {{.Name}}Primary) error {
	return db.Delete(j.tx, {{goesc .Name}}, key)
}
`))
	_ = template.Must(tpl.New("updatefunc").Parse(`
{{if .HasColFns}}
  {{$goDataType := .GoDataType}}
  {{range .Columns}}
    {{if .IsProto}}{{else}}
      var _ interface{ {{- .ColFn}}() {{.GoType -}} } = (*{{$goDataType}})(nil)
    {{end}}
  {{end}}

  func (j *Job) Update{{.GoDataTypeShort}}(pb *{{$goDataType}}) error {
    return j.Update{{.GoType}}(&{{.GoType}} {
      Data: pb,
      {{range .Columns}}
        {{if .IsProto}}{{else}}
          {{.GoName}}: pb.{{- .ColFn}}(),
        {{end}}
      {{end}}
    })
  }
{{end}}
`))
	_ = template.Must(tpl.New("updaterowfunc").Parse(`
func (j *Job) Update{{.GoType}}(row *{{.GoType}}) error {
  {{range .Indexes}}
    {{if eq .KeyType "PRIMARY KEY"}}
      key := {{.Name}}{
        {{range .Columns}}
          {{.GoName}}: &row.{{.GoName}},
        {{end}}
      }
    {{end}}
  {{end}}

  var vals []interface{}
  {{range .Columns}}
    {{if .IsProto}}
      if val, err := proto.Marshal(row.{{.GoName}}); err != nil {
        return err
      } else {
        vals = append(vals, val)
      }
    {{else}}
      vals = append(vals, row.{{.GoName}})
    {{end}}
  {{end}}

	return db.Update(j.tx, {{goesc .Name}}, cols{{.Name}}, vals, key)
}
`))
)
