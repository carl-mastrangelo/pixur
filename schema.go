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
			"  `tag` varchar(255) COLLATE utf8_unicode_ci NOT NULL," +
			"  `created_time` bigint(20) NOT NULL," +
			"  `modified_time` bigint(20) NOT NULL," +
			"  PRIMARY KEY (`id`)," +
			"  UNIQUE KEY `tag` (`tag`)" +
			"  ) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci AUTO_INCREMENT=1 ;",

		"CREATE TABLE IF NOT EXISTS `pictags` (" +
			"  `pic_id` int(11) NOT NULL," +
			"  `tag_id` int(11) NOT NULL," +
			"  `tag` varchar(255) COLLATE utf8_unicode_ci NOT NULL," +
			"  `created_time` bigint(20) NOT NULL," +
			"  `modified_time` bigint(20) NOT NULL," +
			"  PRIMARY KEY (`pic_id`,`tag_id`)," +
			"  KEY `tag_id` (`tag_id`)" +
			"  ) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci;",

		"ALTER TABLE `pictags`" +
			"  ADD CONSTRAINT `pictags_ibfk_2` FOREIGN KEY (`tag_id`) REFERENCES `tags` (`id`)," +
			"  ADD CONSTRAINT `pictags_ibfk_1` FOREIGN KEY (`pic_id`) REFERENCES `pics` (`id`);",
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
