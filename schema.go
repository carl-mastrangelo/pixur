package pixur

import (
	"database/sql"
)

var (
	rawSchemaTable = []string{
		"CREATE TABLE IF NOT EXISTS `pics` (" +
			"  `id` int(11) NOT NULL AUTO_INCREMENT," +
			"  `file_size` int(11) NOT NULL," +
			"  `mime` tinyint(4) NOT NULL," +
			"  `width` int(11) NOT NULL," +
			"  `height` int(11) NOT NULL," +
			"  `created_time` bigint(20) NOT NULL," +
			"  `modified_time` bigint(20) NOT NULL," +
			"  PRIMARY KEY (`id`)" +
			") ENGINE=InnoDB  DEFAULT CHARSET=utf8;",

		"CREATE TABLE IF NOT EXISTS `tags` (" +
			"  `id` int(11) NOT NULL AUTO_INCREMENT," +
			"  `name` varchar(255) COLLATE utf8_bin NOT NULL," +
			"  `created_time` bigint(20) NOT NULL," +
			"  `modified_time` bigint(20) NOT NULL," +
			"  PRIMARY KEY (`id`)," +
			"  UNIQUE KEY `name` (`name`)" +
			"  ) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin AUTO_INCREMENT=1 ;",

		"CREATE TABLE IF NOT EXISTS `pictags` (" +
			"  `pic_id` int(11) NOT NULL," +
			"  `tag_id` int(11) NOT NULL," +
			"  `name` varchar(255) COLLATE utf8_bin NOT NULL," +
			"  `created_time` bigint(20) NOT NULL," +
			"  `modified_time` bigint(20) NOT NULL," +
			"  PRIMARY KEY (`pic_id`,`tag_id`)," +
			"  KEY `tag_id` (`tag_id`)" +
			"  ) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;",
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
