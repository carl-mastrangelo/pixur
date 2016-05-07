package tables

import (
	"database/sql"

	"github.com/golang/protobuf/proto"

	"pixur.org/pixur/schema/db"

	schema "pixur.org/pixur/schema"
)

var SqlTables = []string{
	"CREATE TABLE `Pics` (" +
		"`id` bigint(20) NOT NULL, " +
		"`created_time` bigint(20) NOT NULL, " +
		"`is_hidden` bool NOT NULL, " +
		"`data` blob NOT NULL, " +
		"PRIMARY KEY(`id`)" +
		");",
	"CREATE INDEX `PicsBumpOrder` ON `Pics` (`created_time`);",
	"CREATE INDEX `PicsHidden` ON `Pics` (`is_hidden`);",
	"CREATE TABLE `Tags` (" +
		"`id` bigint(20) NOT NULL, " +
		"`name` blob NOT NULL, " +
		"`data` blob NOT NULL, " +
		"PRIMARY KEY(`id`), " +
		"UNIQUE(`name`(255))" +
		");",
	"CREATE TABLE `PicTags` (" +
		"`pic_id` bigint(20) NOT NULL, " +
		"`tag_id` bigint(20) NOT NULL, " +
		"`data` blob NOT NULL, " +
		"PRIMARY KEY(`pic_id`, `tag_id`)" +
		");",
	"CREATE TABLE `PicIdents` (" +
		"`pic_id` bigint(20) NOT NULL, " +
		"`type` int NOT NULL, " +
		"`value` blob NOT NULL, " +
		"`data` blob NOT NULL, " +
		"PRIMARY KEY(`pic_id`, `type`, `value`(255))" +
		");",
	"CREATE INDEX `PicIdentsIdent` ON `PicIdents` (`type`, `value`(255));",
	"CREATE TABLE `Users` (" +
		"`id` bigint(20) NOT NULL, " +
		"`ident` blob NOT NULL, " +
		"`data` blob NOT NULL, " +
		"PRIMARY KEY(`id`), " +
		"UNIQUE(`ident`(255))" +
		");",
}

var _ db.UniqueIdx = PicsPrimary{}

type PicsPrimary struct {
	Id *int64
}

func (_ PicsPrimary) Unique() {}

func (idx PicsPrimary) Cols() []string {
	return []string{"id"}
}

func (idx PicsPrimary) Vals() (vals []interface{}) {
	var done bool
	if idx.Id != nil {
		if done {
			panic("Extra value Id")
		}
		vals = append(vals, *idx.Id)
	} else {
		done = true
	}
	return
}

var _ db.Idx = PicsBumpOrder{}

type PicsBumpOrder struct {
	CreatedTime *int64
}

func (idx PicsBumpOrder) Cols() []string {
	return []string{"created_time"}
}

func (idx PicsBumpOrder) Vals() (vals []interface{}) {
	var done bool
	if idx.CreatedTime != nil {
		if done {
			panic("Extra value CreatedTime")
		}
		vals = append(vals, *idx.CreatedTime)
	} else {
		done = true
	}
	return
}

var _ db.Idx = PicsHidden{}

type PicsHidden struct {
	IsHidden *bool
}

func (idx PicsHidden) Cols() []string {
	return []string{"is_hidden"}
}

func (idx PicsHidden) Vals() (vals []interface{}) {
	var done bool
	if idx.IsHidden != nil {
		if done {
			panic("Extra value IsHidden")
		}
		vals = append(vals, *idx.IsHidden)
	} else {
		done = true
	}
	return
}

var _ db.UniqueIdx = TagsPrimary{}

type TagsPrimary struct {
	Id *int64
}

func (_ TagsPrimary) Unique() {}

func (idx TagsPrimary) Cols() []string {
	return []string{"id"}
}

func (idx TagsPrimary) Vals() (vals []interface{}) {
	var done bool
	if idx.Id != nil {
		if done {
			panic("Extra value Id")
		}
		vals = append(vals, *idx.Id)
	} else {
		done = true
	}
	return
}

var _ db.UniqueIdx = TagsName{}

type TagsName struct {
	Name *string
}

func (_ TagsName) Unique() {}

func (idx TagsName) Cols() []string {
	return []string{"name"}
}

func (idx TagsName) Vals() (vals []interface{}) {
	var done bool
	if idx.Name != nil {
		if done {
			panic("Extra value Name")
		}
		vals = append(vals, *idx.Name)
	} else {
		done = true
	}
	return
}

var _ db.UniqueIdx = PicTagsPrimary{}

type PicTagsPrimary struct {
	PicId *int64
	TagId *int64
}

func (_ PicTagsPrimary) Unique() {}

func (idx PicTagsPrimary) Cols() []string {
	return []string{"pic_id", "tag_id"}
}

func (idx PicTagsPrimary) Vals() (vals []interface{}) {
	var done bool
	if idx.PicId != nil {
		if done {
			panic("Extra value PicId")
		}
		vals = append(vals, *idx.PicId)
	} else {
		done = true
	}
	if idx.TagId != nil {
		if done {
			panic("Extra value TagId")
		}
		vals = append(vals, *idx.TagId)
	} else {
		done = true
	}
	return
}

var _ db.UniqueIdx = PicIdentsPrimary{}

type PicIdentsPrimary struct {
	PicId *int64
	Type  *schema.PicIdent_Type
	Value *[]byte
}

func (_ PicIdentsPrimary) Unique() {}

func (idx PicIdentsPrimary) Cols() []string {
	return []string{"pic_id", "type", "value"}
}

func (idx PicIdentsPrimary) Vals() (vals []interface{}) {
	var done bool
	if idx.PicId != nil {
		if done {
			panic("Extra value PicId")
		}
		vals = append(vals, *idx.PicId)
	} else {
		done = true
	}
	if idx.Type != nil {
		if done {
			panic("Extra value Type")
		}
		vals = append(vals, *idx.Type)
	} else {
		done = true
	}
	if idx.Value != nil {
		if done {
			panic("Extra value Value")
		}
		vals = append(vals, *idx.Value)
	} else {
		done = true
	}
	return
}

var _ db.Idx = PicIdentsIdent{}

type PicIdentsIdent struct {
	Type  *schema.PicIdent_Type
	Value *[]byte
}

func (idx PicIdentsIdent) Cols() []string {
	return []string{"type", "value"}
}

func (idx PicIdentsIdent) Vals() (vals []interface{}) {
	var done bool
	if idx.Type != nil {
		if done {
			panic("Extra value Type")
		}
		vals = append(vals, *idx.Type)
	} else {
		done = true
	}
	if idx.Value != nil {
		if done {
			panic("Extra value Value")
		}
		vals = append(vals, *idx.Value)
	} else {
		done = true
	}
	return
}

var _ db.UniqueIdx = UsersPrimary{}

type UsersPrimary struct {
	Id *int64
}

func (_ UsersPrimary) Unique() {}

func (idx UsersPrimary) Cols() []string {
	return []string{"id"}
}

func (idx UsersPrimary) Vals() (vals []interface{}) {
	var done bool
	if idx.Id != nil {
		if done {
			panic("Extra value Id")
		}
		vals = append(vals, *idx.Id)
	} else {
		done = true
	}
	return
}

var _ db.UniqueIdx = UsersIdent{}

type UsersIdent struct {
	Ident *string
}

func (_ UsersIdent) Unique() {}

func (idx UsersIdent) Cols() []string {
	return []string{"ident"}
}

func (idx UsersIdent) Vals() (vals []interface{}) {
	var done bool
	if idx.Ident != nil {
		if done {
			panic("Extra value Ident")
		}
		vals = append(vals, *idx.Ident)
	} else {
		done = true
	}
	return
}

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

func (j Job) ScanPics(opts db.Opts, cb func(schema.Pic) error) error {
	cols := []string{"id", "created_time", "is_hidden", "data"}
	return db.Scan(j, "Pics", opts, func(data []byte) error {
		var pb schema.Pic
		if err := proto.Unmarshal(data, &pb); err != nil {
			return err
		}
		return cb(pb)
	}, cols)
}

func (j Job) ScanTags(opts db.Opts, cb func(schema.Tag) error) error {
	cols := []string{"id", "name", "data"}
	return db.Scan(j, "Tags", opts, func(data []byte) error {
		var pb schema.Tag
		if err := proto.Unmarshal(data, &pb); err != nil {
			return err
		}
		return cb(pb)
	}, cols)
}

func (j Job) ScanPicTags(opts db.Opts, cb func(schema.PicTag) error) error {
	cols := []string{"pic_id", "tag_id", "data"}
	return db.Scan(j, "PicTags", opts, func(data []byte) error {
		var pb schema.PicTag
		if err := proto.Unmarshal(data, &pb); err != nil {
			return err
		}
		return cb(pb)
	}, cols)
}

func (j Job) ScanPicIdents(opts db.Opts, cb func(schema.PicIdent) error) error {
	cols := []string{"pic_id", "type", "value", "data"}
	return db.Scan(j, "PicIdents", opts, func(data []byte) error {
		var pb schema.PicIdent
		if err := proto.Unmarshal(data, &pb); err != nil {
			return err
		}
		return cb(pb)
	}, cols)
}

func (j Job) ScanUsers(opts db.Opts, cb func(schema.User) error) error {
	cols := []string{"id", "ident", "data"}
	return db.Scan(j, "Users", opts, func(data []byte) error {
		var pb schema.User
		if err := proto.Unmarshal(data, &pb); err != nil {
			return err
		}
		return cb(pb)
	}, cols)
}

func (j Job) FindPics(opts db.Opts) (rows []schema.Pic, err error) {
	err = j.ScanPics(opts, func(data schema.Pic) error {
		rows = append(rows, data)
		return nil
	})
	return
}

func (j Job) FindTags(opts db.Opts) (rows []schema.Tag, err error) {
	err = j.ScanTags(opts, func(data schema.Tag) error {
		rows = append(rows, data)
		return nil
	})
	return
}

func (j Job) FindPicTags(opts db.Opts) (rows []schema.PicTag, err error) {
	err = j.ScanPicTags(opts, func(data schema.PicTag) error {
		rows = append(rows, data)
		return nil
	})
	return
}

func (j Job) FindPicIdents(opts db.Opts) (rows []schema.PicIdent, err error) {
	err = j.ScanPicIdents(opts, func(data schema.PicIdent) error {
		rows = append(rows, data)
		return nil
	})
	return
}

func (j Job) FindUsers(opts db.Opts) (rows []schema.User, err error) {
	err = j.ScanUsers(opts, func(data schema.User) error {
		rows = append(rows, data)
		return nil
	})
	return
}

func (j Job) InsertPicRow(row PicRow) error {
	cols := []string{"id", "created_time", "is_hidden", "data"}
	vals := []interface{}{row.Id, row.CreatedTime, row.IsHidden, row.Data}
	return db.Insert(j, "Pics", cols, vals)
}

func (j Job) InsertTagRow(row TagRow) error {
	cols := []string{"id", "name", "data"}
	vals := []interface{}{row.Id, row.Name, row.Data}
	return db.Insert(j, "Tags", cols, vals)
}

func (j Job) InsertPicTagRow(row PicTagRow) error {
	cols := []string{"pic_id", "tag_id", "data"}
	vals := []interface{}{row.PicId, row.TagId, row.Data}
	return db.Insert(j, "PicTags", cols, vals)
}

func (j Job) InsertPicIdentRow(row PicIdentRow) error {
	cols := []string{"pic_id", "type", "value", "data"}
	vals := []interface{}{row.PicId, row.Type, row.Value, row.Data}
	return db.Insert(j, "PicIdents", cols, vals)
}

func (j Job) InsertUserRow(row UserRow) error {
	cols := []string{"id", "ident", "data"}
	vals := []interface{}{row.Id, row.Ident, row.Data}
	return db.Insert(j, "Users", cols, vals)
}

func (j Job) DeletePicRow(key PicsPrimary) error {
	return db.Delete(j, "Pics", key)
}

func (j Job) DeleteTagRow(key TagsPrimary) error {
	return db.Delete(j, "Tags", key)
}

func (j Job) DeletePicTagRow(key PicTagsPrimary) error {
	return db.Delete(j, "PicTags", key)
}

func (j Job) DeletePicIdentRow(key PicIdentsPrimary) error {
	return db.Delete(j, "PicIdents", key)
}

func (j Job) DeleteUserRow(key UsersPrimary) error {
	return db.Delete(j, "Users", key)
}
