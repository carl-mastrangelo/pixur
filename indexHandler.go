package pixur

import (
  "net/http"
  _ "fmt"
  "reflect"
  "strings"
  "database/sql"
  
  
  "html/template"
)

type indexParams struct {
  Pics []*Pic
}

func FetchPics(db *sql.DB) ([]*Pic, error) {
  typ := reflect.TypeOf(Pic{})
  var columnNames = make([]string, 0, typ.NumField())
  var columnIndicies = make([]int, 0, typ.NumField())
  for i := 0; i < typ.NumField(); i++ {
    if columnName := typ.Field(i).Tag.Get("mysql"); columnName != "" {
      columnNames = append(columnNames, "`" + columnName + "`")
      columnIndicies = append(columnIndicies, i)
    }
  }

  flatColumnNames := strings.Join(columnNames, ",")

  rows, err := db.Query("SELECT " + flatColumnNames + " FROM pix;")
  if err != nil {
    return nil, err
  }
  defer rows.Close()

  var pics []*Pic
  for rows.Next() {
    var p = new(Pic)
    val := reflect.Indirect(reflect.ValueOf(p))
    var rawRowValues = make([]interface{}, 0, len(columnIndicies))
    for _, columnIndex := range columnIndicies {
      rawRowValues = append(rawRowValues, val.Field(columnIndex).Addr().Interface())
    }
    if err := rows.Scan(rawRowValues...); err != nil {
      return nil, err
    }
    pics = append(pics, p)
  }
  
  if err := rows.Err(); err != nil {
    return nil, err
  }
  
  return pics, nil
}

func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) error {
  pics, err := FetchPics(s.db)
  if err != nil {
    return err
  }
  
  var params indexParams
    params.Pics = pics

  tpl, err := template.ParseFiles("tpl/index.html")
  if err != nil {
    return err
  }
  if err := tpl.Execute(w, params); err != nil {
    return err
  }
  return nil
}
