package pixur

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"html/template"
)

type indexParams struct {
	Pics []*Pic
}

func FetchPics(db *sql.DB) ([]*Pic, error) {
  var columnNameMap = (&Pic{}).PointerMap()
  
  var columnNames = make([]string, 0, len(columnNameMap))
	for name, _ := range columnNameMap {
    columnNames = append(columnNames, name)
  }

  stmt := fmt.Sprintf("SELECT %s FROM pix;", strings.Join(columnNames, ","))
	rows, err := db.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pics []*Pic
	for rows.Next() {
		var p = new(Pic)
    pmap := p.PointerMap()
    
		var rawRowValues = make([]interface{}, 0, len(columnNames))
		for _, columnName := range columnNames {
			rawRowValues = append(rawRowValues, pmap[columnName])
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
