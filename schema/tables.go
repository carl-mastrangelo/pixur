package schema

import (
	"database/sql"
)

var (
	rawSchemaTable = []string{
		"CREATE TABLE " + PicTableName + " (" +
			"  " + PicColId + " bigint(20) NOT NULL," +
			"  " + PicColData + " blob NOT NULL," +
			"  " + PicColCreatedTime + " bigint(20) NOT NULL," +
			"  " + PicColHidden + " bool NOT NULL," +
			"  PRIMARY KEY (" + PicColId + ")," +
			"  KEY " + PicColCreatedTime + " (" + PicColCreatedTime + ")," +
			"  KEY " + PicColHidden + " (" + PicColHidden + ")" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;",

		"CREATE TABLE " + TagTableName + " (" +
			"  " + TagColId + " bigint(20) NOT NULL," +
			"  " + TagColData + " blob NOT NULL," +
			"  " + TagColName + " varchar(255) NOT NULL," +
			"  PRIMARY KEY (" + TagColId + ")," +
			"  UNIQUE KEY (" + TagColName + ") " +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;",

		"CREATE TABLE " + PicTagTableName + " (" +
			"  " + PicTagColPicId + " bigint(20) NOT NULL," +
			"  " + PicTagColTagId + " bigint(20) NOT NULL," +
			"  " + PicTagColData + " blob NOT NULL," +
			"  PRIMARY KEY (" + PicTagColPicId + "," + PicTagColTagId + ")" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;",

		"CREATE TABLE " + PicIdentTableName + " (" +
			"  " + PicIdentColPicId + " bigint(20) NOT NULL," +
			"  " + PicIdentColType + " bigint(20) NOT NULL," +
			"  " + PicIdentColValue + " tinyblob NOT NULL," +
			"  " + PicIdentColData + " blob NOT NULL," +
			"  PRIMARY KEY (" + PicIdentColPicId + "," + PicIdentColType + "," + PicIdentColValue + "(255))," +
			"  KEY " + PicIdentColValue + " (" + PicIdentColType + "," + PicIdentColValue + "(255))" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;",

		"CREATE TABLE " + UserTableName + " (" +
			"  " + UserColId + " bigint(20) NOT NULL," +
			"  " + UserColEmail + " tinyblob NOT NULL," +
			"  " + UserColData + " blob NOT NULL," +
			"  PRIMARY KEY (" + UserColId + ")," +
			"  KEY " + UserColEmail + " (" + UserColEmail + "(255))" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;",

		// Special
		"CREATE TABLE " + SeqTableName + " (" +
			"  " + SeqColSeq + " bigint(20) NOT NULL" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;",

		"INSERT INTO " + SeqTableName + " (" + SeqColSeq + ") VALUES (1);",
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
