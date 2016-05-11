package tables

import (
	"database/sql"

	"github.com/golang/protobuf/proto"

	"pixur.org/pixur/schema/db"

	schema "pixur.org/pixur/schema"

	model "pixur.org/pixur/schema/db/model"
)

var (
	_ = schema.Pic{}

	_ = model.TableOptions{}
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

		"UNIQUE(`name`(255)), " +

		"PRIMARY KEY(`id`)" +

		");",

	"CREATE TABLE `PicTags` (" +

		"`pic_id` bigint(20) NOT NULL, " +

		"`tag_id` bigint(20) NOT NULL, " +

		"`data` blob NOT NULL, " +

		"PRIMARY KEY(`pic_id`,`tag_id`)" +

		");",

	"CREATE TABLE `PicIdents` (" +

		"`pic_id` bigint(20) NOT NULL, " +

		"`type` int NOT NULL, " +

		"`value` blob NOT NULL, " +

		"`data` blob NOT NULL, " +

		"PRIMARY KEY(`pic_id`,`type`,`value`(255))" +

		");",

	"CREATE INDEX `PicIdentsIdent` ON `PicIdents` (`type`,`value`(255));",

	"CREATE TABLE `Users` (" +

		"`id` bigint(20) NOT NULL, " +

		"`ident` blob NOT NULL, " +

		"`data` blob NOT NULL, " +

		"UNIQUE(`ident`(255)), " +

		"PRIMARY KEY(`id`)" +

		");",
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

type PicsPrimary struct {
	Id *int64
}

func (_ PicsPrimary) Unique() {}

var _ db.UniqueIdx = PicsPrimary{}

var colsPicsPrimary = []string{"id"}

func (idx PicsPrimary) Cols() []string {
	return colsPicsPrimary
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

type PicsBumpOrder struct {
	CreatedTime *int64
}

var _ db.Idx = PicsBumpOrder{}

var colsPicsBumpOrder = []string{"created_time"}

func (idx PicsBumpOrder) Cols() []string {
	return colsPicsBumpOrder
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

type PicsHidden struct {
	IsHidden *bool
}

var _ db.Idx = PicsHidden{}

var colsPicsHidden = []string{"is_hidden"}

func (idx PicsHidden) Cols() []string {
	return colsPicsHidden
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

var colsPics = []string{"id", "created_time", "is_hidden", "data"}

func (j Job) ScanPics(opts db.Opts, cb func(schema.Pic) error) error {
	return db.Scan(j, "Pics", opts, func(data []byte) error {
		var pb schema.Pic
		if err := proto.Unmarshal(data, &pb); err != nil {
			return err
		}
		return cb(pb)
	}, colsPics)
}

func (j Job) FindPics(opts db.Opts) (rows []schema.Pic, err error) {
	err = j.ScanPics(opts, func(data schema.Pic) error {
		rows = append(rows, data)
		return nil
	})
	return
}

func (j Job) InsertPics(row PicRow) error {
	vals := []interface{}{row.Id, row.CreatedTime, row.IsHidden, row.Data}
	return db.Insert(j, "Pics", colsPics, vals)
}

func (j Job) DeletePics(key PicsPrimary) error {
	return db.Delete(j, "Pics", key)
}

type TagsPrimary struct {
	Id *int64
}

func (_ TagsPrimary) Unique() {}

var _ db.UniqueIdx = TagsPrimary{}

var colsTagsPrimary = []string{"id"}

func (idx TagsPrimary) Cols() []string {
	return colsTagsPrimary
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

type TagsName struct {
	Name *string
}

func (_ TagsName) Unique() {}

var _ db.UniqueIdx = TagsName{}

var colsTagsName = []string{"name"}

func (idx TagsName) Cols() []string {
	return colsTagsName
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

var colsTags = []string{"id", "name", "data"}

func (j Job) ScanTags(opts db.Opts, cb func(schema.Tag) error) error {
	return db.Scan(j, "Tags", opts, func(data []byte) error {
		var pb schema.Tag
		if err := proto.Unmarshal(data, &pb); err != nil {
			return err
		}
		return cb(pb)
	}, colsTags)
}

func (j Job) FindTags(opts db.Opts) (rows []schema.Tag, err error) {
	err = j.ScanTags(opts, func(data schema.Tag) error {
		rows = append(rows, data)
		return nil
	})
	return
}

func (j Job) InsertTags(row TagRow) error {
	vals := []interface{}{row.Id, row.Name, row.Data}
	return db.Insert(j, "Tags", colsTags, vals)
}

func (j Job) DeleteTags(key TagsPrimary) error {
	return db.Delete(j, "Tags", key)
}

type PicTagsPrimary struct {
	PicId *int64

	TagId *int64
}

func (_ PicTagsPrimary) Unique() {}

var _ db.UniqueIdx = PicTagsPrimary{}

var colsPicTagsPrimary = []string{"pic_id", "tag_id"}

func (idx PicTagsPrimary) Cols() []string {
	return colsPicTagsPrimary
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

var colsPicTags = []string{"pic_id", "tag_id", "data"}

func (j Job) ScanPicTags(opts db.Opts, cb func(schema.PicTag) error) error {
	return db.Scan(j, "PicTags", opts, func(data []byte) error {
		var pb schema.PicTag
		if err := proto.Unmarshal(data, &pb); err != nil {
			return err
		}
		return cb(pb)
	}, colsPicTags)
}

func (j Job) FindPicTags(opts db.Opts) (rows []schema.PicTag, err error) {
	err = j.ScanPicTags(opts, func(data schema.PicTag) error {
		rows = append(rows, data)
		return nil
	})
	return
}

func (j Job) InsertPicTags(row PicTagRow) error {
	vals := []interface{}{row.PicId, row.TagId, row.Data}
	return db.Insert(j, "PicTags", colsPicTags, vals)
}

func (j Job) DeletePicTags(key PicTagsPrimary) error {
	return db.Delete(j, "PicTags", key)
}

type PicIdentsPrimary struct {
	PicId *int64

	Type *schema.PicIdent_Type

	Value *[]byte
}

func (_ PicIdentsPrimary) Unique() {}

var _ db.UniqueIdx = PicIdentsPrimary{}

var colsPicIdentsPrimary = []string{"pic_id", "type", "value"}

func (idx PicIdentsPrimary) Cols() []string {
	return colsPicIdentsPrimary
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

type PicIdentsIdent struct {
	Type *schema.PicIdent_Type

	Value *[]byte
}

var _ db.Idx = PicIdentsIdent{}

var colsPicIdentsIdent = []string{"type", "value"}

func (idx PicIdentsIdent) Cols() []string {
	return colsPicIdentsIdent
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

var colsPicIdents = []string{"pic_id", "type", "value", "data"}

func (j Job) ScanPicIdents(opts db.Opts, cb func(schema.PicIdent) error) error {
	return db.Scan(j, "PicIdents", opts, func(data []byte) error {
		var pb schema.PicIdent
		if err := proto.Unmarshal(data, &pb); err != nil {
			return err
		}
		return cb(pb)
	}, colsPicIdents)
}

func (j Job) FindPicIdents(opts db.Opts) (rows []schema.PicIdent, err error) {
	err = j.ScanPicIdents(opts, func(data schema.PicIdent) error {
		rows = append(rows, data)
		return nil
	})
	return
}

func (j Job) InsertPicIdents(row PicIdentRow) error {
	vals := []interface{}{row.PicId, row.Type, row.Value, row.Data}
	return db.Insert(j, "PicIdents", colsPicIdents, vals)
}

func (j Job) DeletePicIdents(key PicIdentsPrimary) error {
	return db.Delete(j, "PicIdents", key)
}

type UsersPrimary struct {
	Id *int64
}

func (_ UsersPrimary) Unique() {}

var _ db.UniqueIdx = UsersPrimary{}

var colsUsersPrimary = []string{"id"}

func (idx UsersPrimary) Cols() []string {
	return colsUsersPrimary
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

type UsersIdent struct {
	Ident *string
}

func (_ UsersIdent) Unique() {}

var _ db.UniqueIdx = UsersIdent{}

var colsUsersIdent = []string{"ident"}

func (idx UsersIdent) Cols() []string {
	return colsUsersIdent
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

var colsUsers = []string{"id", "ident", "data"}

func (j Job) ScanUsers(opts db.Opts, cb func(schema.User) error) error {
	return db.Scan(j, "Users", opts, func(data []byte) error {
		var pb schema.User
		if err := proto.Unmarshal(data, &pb); err != nil {
			return err
		}
		return cb(pb)
	}, colsUsers)
}

func (j Job) FindUsers(opts db.Opts) (rows []schema.User, err error) {
	err = j.ScanUsers(opts, func(data schema.User) error {
		rows = append(rows, data)
		return nil
	})
	return
}

func (j Job) InsertUsers(row UserRow) error {
	vals := []interface{}{row.Id, row.Ident, row.Data}
	return db.Insert(j, "Users", colsUsers, vals)
}

func (j Job) DeleteUsers(key UsersPrimary) error {
	return db.Delete(j, "Users", key)
}
