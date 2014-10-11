package pixur

import (
	"database/sql"
)

var (
	rawSchemaTable = []string{
		"CREATE TABLE IF NOT EXISTS `pix` (" +
			"  `id` int(11) NOT NULL AUTO_INCREMENT," +
			"  `file_size` int(11) NOT NULL," +
			"  `mime` tinyint(4) NOT NULL," +
			"  `width` int(11) NOT NULL," +
			"  `height` int(11) NOT NULL," +
			"  `created_time_msec` bigint(20) NOT NULL," +
			"  `modified_time_msec` bigint(20) NOT NULL," +
			"  PRIMARY KEY (`id`)" +
			") ENGINE=InnoDB  DEFAULT CHARSET=utf8;",
	}
)

func createTables(db *sql.DB) error {
	for _, schemaTable := range rawSchemaTable {
		if _, err := db.Exec(schemaTable); err != nil {
			return err
		}
	}
	return nil
}
