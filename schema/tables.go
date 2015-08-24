package schema

import (
	"database/sql"
)

var (
	rawSchemaTable = []string{
		"CREATE TABLE IF NOT EXISTS " + PicTableName + " (" +
			"  " + PicColId + " bigint(11) NOT NULL AUTO_INCREMENT," +
			"  " + PicColData + " blob NOT NULL," +
			"  " + PicColCreatedTime + " bigint(20) NOT NULL," +
			"  " + PicColHidden + " bool NOT NULL," +
			"  PRIMARY KEY (" + PicColId + ")," +
			"  KEY " + PicColCreatedTime + " (" + PicColCreatedTime + ")," +
			"  KEY " + PicColHidden + " (" + PicColHidden + ")" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin AUTO_INCREMENT=1;",

		"CREATE TABLE IF NOT EXISTS " + TagTableName + " (" +
			"  " + TagColId + " bigint(11) NOT NULL AUTO_INCREMENT," +
			"  " + TagColData + " blob NOT NULL," +
			"  " + TagColName + " varchar(255) NOT NULL," +
			"  PRIMARY KEY (" + TagColId + ")," +
			"  UNIQUE KEY (" + TagColName + ") " +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin AUTO_INCREMENT=1;",

		"CREATE TABLE IF NOT EXISTS " + PicTagTableName + " (" +
			"  " + PicTagColPicId + " bigint(11) NOT NULL," +
			"  " + PicTagColTagId + " bigint(11) NOT NULL," +
			"  " + PicTagColData + " blob NOT NULL," +
			"  PRIMARY KEY (" + PicTagColPicId + "," + PicTagColTagId + ")" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;",

		"CREATE TABLE IF NOT EXISTS " + PicIdentTableName + " (" +
			"  " + PicIdentColPicId + " int(11) NOT NULL," +
			"  " + PicIdentColType + " int(11) NOT NULL," +
			"  " + PicIdentColValue + " tinyblob NOT NULL," +
			"  " + PicIdentColData + " blob NOT NULL," +
			"  PRIMARY KEY (" + PicIdentColPicId + "," + PicIdentColType + "," + PicIdentColValue + "(255))," +
			"  KEY " + PicIdentColValue + " (" + PicIdentColType + "," + PicIdentColValue + "(255))" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;",
	}
)

func CreateTables(db *sql.DB) error {
	for _, schemaTable := range rawSchemaTable {
		if _, err := db.Exec(schemaTable); err != nil {
			return err
		}
	}
	return nil
}
