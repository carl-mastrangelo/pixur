package tables

import (
	"database/sql"

	"github.com/golang/protobuf/proto"

	"pixur.org/pixur/schema"
	"pixur.org/pixur/schema/db"
)

type Job struct {
	Tx *sql.Tx
}

var SqlTables = []string{
	"CREATE TABLE \"Pics\" (" +
		"\"id\" bigint NOT NULL, " +
		"\"created_time\" bigint NOT NULL, " +
		"\"is_hidden\" smallint NOT NULL, " +
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
}

func (idx PicsPrimary) Cols() []string {
	return []string{"id"}
}

func (idx PicsPrimary) Vals() []interface{} {
}

var _ db.Idx = PicsBumpOrder{}

type PicsBumpOrder struct {
}

func (idx PicsBumpOrder) Cols() []string {
	return []string{"created_time"}
}

func (idx PicsBumpOrder) Vals() []interface{} {
}

var _ db.Idx = PicsHidden{}

type PicsHidden struct {
}

func (idx PicsHidden) Cols() []string {
	return []string{"is_hidden"}
}

func (idx PicsHidden) Vals() []interface{} {
}

var _ db.Idx = TagsPrimary{}

type TagsPrimary struct {
}

func (idx TagsPrimary) Cols() []string {
	return []string{"id"}
}

func (idx TagsPrimary) Vals() []interface{} {
}

var _ db.Idx = TagsName{}

type TagsName struct {
}

func (idx TagsName) Cols() []string {
	return []string{"name"}
}

func (idx TagsName) Vals() []interface{} {
}

var _ db.Idx = PicTagsPrimary{}

type PicTagsPrimary struct {
}

func (idx PicTagsPrimary) Cols() []string {
	return []string{"pic_id", "tag_id"}
}

func (idx PicTagsPrimary) Vals() []interface{} {
}

var _ db.Idx = PicIdentsPrimary{}

type PicIdentsPrimary struct {
}

func (idx PicIdentsPrimary) Cols() []string {
	return []string{"pic_id", "type", "value"}
}

func (idx PicIdentsPrimary) Vals() []interface{} {
}

var _ db.Idx = PicIdentsIdent{}

type PicIdentsIdent struct {
}

func (idx PicIdentsIdent) Cols() []string {
	return []string{"type", "value"}
}

func (idx PicIdentsIdent) Vals() []interface{} {
}

var _ db.Idx = UsersPrimary{}

type UsersPrimary struct {
}

func (idx UsersPrimary) Cols() []string {
	return []string{"id"}
}

func (idx UsersPrimary) Vals() []interface{} {
}

var _ db.Idx = UsersIdent{}

type UsersIdent struct {
}

func (idx UsersIdent) Cols() []string {
	return []string{"ident"}
}

func (idx UsersIdent) Vals() []interface{} {
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

func (j Job) FindPics(opts db.Opts) (rows []schema.Pic, err error) {
	err = j.ScanPics(opts, func(data schema.Pic) error {
		rows = append(rows, data)
		return nil
	})
	return
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

func (j Job) FindTags(opts db.Opts) (rows []schema.Tag, err error) {
	err = j.ScanTags(opts, func(data schema.Tag) error {
		rows = append(rows, data)
		return nil
	})
	return
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

func (j Job) FindPicTags(opts db.Opts) (rows []schema.PicTag, err error) {
	err = j.ScanPicTags(opts, func(data schema.PicTag) error {
		rows = append(rows, data)
		return nil
	})
	return
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

func (j Job) FindPicIdents(opts db.Opts) (rows []schema.PicIdent, err error) {
	err = j.ScanPicIdents(opts, func(data schema.PicIdent) error {
		rows = append(rows, data)
		return nil
	})
	return
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

func (j Job) FindUsers(opts db.Opts) (rows []schema.User, err error) {
	err = j.ScanUsers(opts, func(data schema.User) error {
		rows = append(rows, data)
		return nil
	})
	return
}
