package tables

import (
	"database/sql"

	"github.com/golang/protobuf/proto"

	"pixur.org/pixur/schema/db"

	schema "pixur.org/pixur/schema"
	_ "pixur.org/pixur/schema/db/model"
)

var SqlTables = []string{
	"CREATE TABLE \"Pics\" (" +
		"\"id\" bigint NOT NULL, " +
		"\"created_time\" bigint NOT NULL, " +
		"\"is_hidden\" boolean NOT NULL, " +
		"\"data\" bytea NOT NULL, " +
		"PRIMARY KEY(\"id\")" +
		");",
	"CREATE INDEX \"PicsBumpOrder\" ON \"Pics\" (\"created_time\");",
	"CREATE INDEX \"PicsHidden\" ON \"Pics\" (\"is_hidden\");",
	"CREATE TABLE \"Tags\" (" +
		"\"id\" bigint NOT NULL, " +
		"\"name\" bytea NOT NULL, " +
		"\"data\" bytea NOT NULL, " +
		"PRIMARY KEY(\"id\"), " +
		"UNIQUE(\"name\")" +
		");",
	"CREATE TABLE \"PicTags\" (" +
		"\"pic_id\" bigint NOT NULL, " +
		"\"tag_id\" bigint NOT NULL, " +
		"\"data\" bytea NOT NULL, " +
		"PRIMARY KEY(\"pic_id\", \"tag_id\")" +
		");",
	"CREATE TABLE \"PicIdents\" (" +
		"\"pic_id\" bigint NOT NULL, " +
		"\"type\" integer NOT NULL, " +
		"\"value\" bytea NOT NULL, " +
		"\"data\" bytea NOT NULL, " +
		"PRIMARY KEY(\"pic_id\", \"type\", \"value\")" +
		");",
	"CREATE INDEX \"PicIdentsIdent\" ON \"PicIdents\" (\"type\", \"value\");",
	"CREATE TABLE \"Users\" (" +
		"\"id\" bigint NOT NULL, " +
		"\"ident\" bytea NOT NULL, " +
		"\"data\" bytea NOT NULL, " +
		"PRIMARY KEY(\"id\"), " +
		"UNIQUE(\"ident\")" +
		");",
}

var _ db.Idx = PicsPrimary{}

type PicsPrimary struct {
	Id *int64
}

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

var _ db.Idx = TagsPrimary{}

type TagsPrimary struct {
	Id *int64
}

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

var _ db.Idx = TagsName{}

type TagsName struct {
	Name *string
}

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

var _ db.Idx = PicTagsPrimary{}

type PicTagsPrimary struct {
	PicId *int64
	TagId *int64
}

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

var _ db.Idx = PicIdentsPrimary{}

type PicIdentsPrimary struct {
	PicId *int64
	Type  *schema.PicIdent_Type
	Value *[]byte
}

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

var _ db.Idx = UsersPrimary{}

type UsersPrimary struct {
	Id *int64
}

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

var _ db.Idx = UsersIdent{}

type UsersIdent struct {
	Ident *string
}

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

func (j Job) ScanPics(opts db.Opts, cb func(schema.Pic) error) error {
	return db.Scan(j.Tx, "Pics", opts, func(data []byte) error {
		var pb schema.Pic
		if err := proto.Unmarshal(data, &pb); err != nil {
			return err
		}
		return cb(pb)
	})
}

func (j Job) ScanTags(opts db.Opts, cb func(schema.Tag) error) error {
	return db.Scan(j.Tx, "Tags", opts, func(data []byte) error {
		var pb schema.Tag
		if err := proto.Unmarshal(data, &pb); err != nil {
			return err
		}
		return cb(pb)
	})
}

func (j Job) ScanPicTags(opts db.Opts, cb func(schema.PicTag) error) error {
	return db.Scan(j.Tx, "PicTags", opts, func(data []byte) error {
		var pb schema.PicTag
		if err := proto.Unmarshal(data, &pb); err != nil {
			return err
		}
		return cb(pb)
	})
}

func (j Job) ScanPicIdents(opts db.Opts, cb func(schema.PicIdent) error) error {
	return db.Scan(j.Tx, "PicIdents", opts, func(data []byte) error {
		var pb schema.PicIdent
		if err := proto.Unmarshal(data, &pb); err != nil {
			return err
		}
		return cb(pb)
	})
}

func (j Job) ScanUsers(opts db.Opts, cb func(schema.User) error) error {
	return db.Scan(j.Tx, "Users", opts, func(data []byte) error {
		var pb schema.User
		if err := proto.Unmarshal(data, &pb); err != nil {
			return err
		}
		return cb(pb)
	})
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
