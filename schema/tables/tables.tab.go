package tables

import (
	"log"
	"runtime"

	"github.com/golang/protobuf/proto"

	"pixur.org/pixur/schema/db"

	schema "pixur.org/pixur/schema"

	model "pixur.org/pixur/schema/db/model"
)

var (
	_ = schema.Pic{}

	_ = model.TableOptions{}
)

var SqlTables = map[string][]string{

	"mysql": {

		"CREATE TABLE `Pics` (" +

			"`id` bigint(20) NOT NULL, " +

			"`index_order` bigint(20) NOT NULL, " +

			"`data` blob NOT NULL, " +

			"PRIMARY KEY(`id`)" +

			");",

		"CREATE INDEX `PicsIndexOrder` ON `Pics` (`index_order`);",

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

		"CREATE TABLE `_SequenceTable` (`the_sequence` bigint(20) NOT NULL);",
	},

	"postgresql": {

		"CREATE TABLE \"Pics\" (" +

			"\"id\" bigint NOT NULL, " +

			"\"index_order\" bigint NOT NULL, " +

			"\"data\" bytea NOT NULL, " +

			"PRIMARY KEY(\"id\")" +

			");",

		"CREATE INDEX \"PicsIndexOrder\" ON \"Pics\" (\"index_order\");",

		"CREATE TABLE \"Tags\" (" +

			"\"id\" bigint NOT NULL, " +

			"\"name\" bytea NOT NULL, " +

			"\"data\" bytea NOT NULL, " +

			"UNIQUE(\"name\"), " +

			"PRIMARY KEY(\"id\")" +

			");",

		"CREATE TABLE \"PicTags\" (" +

			"\"pic_id\" bigint NOT NULL, " +

			"\"tag_id\" bigint NOT NULL, " +

			"\"data\" bytea NOT NULL, " +

			"PRIMARY KEY(\"pic_id\",\"tag_id\")" +

			");",

		"CREATE TABLE \"PicIdents\" (" +

			"\"pic_id\" bigint NOT NULL, " +

			"\"type\" integer NOT NULL, " +

			"\"value\" bytea NOT NULL, " +

			"\"data\" bytea NOT NULL, " +

			"PRIMARY KEY(\"pic_id\",\"type\",\"value\")" +

			");",

		"CREATE INDEX \"PicIdentsIdent\" ON \"PicIdents\" (\"type\",\"value\");",

		"CREATE TABLE \"Users\" (" +

			"\"id\" bigint NOT NULL, " +

			"\"ident\" bytea NOT NULL, " +

			"\"data\" bytea NOT NULL, " +

			"UNIQUE(\"ident\"), " +

			"PRIMARY KEY(\"id\")" +

			");",

		"CREATE TABLE \"_SequenceTable\" (\"the_sequence\" bigint NOT NULL);",
	},

	"sqlite3": {

		"CREATE TABLE \"Pics\" (" +

			"\"id\" integer NOT NULL, " +

			"\"index_order\" integer NOT NULL, " +

			"\"data\" blob NOT NULL, " +

			"PRIMARY KEY(\"id\")" +

			");",

		"CREATE INDEX \"PicsIndexOrder\" ON \"Pics\" (\"index_order\");",

		"CREATE TABLE \"Tags\" (" +

			"\"id\" integer NOT NULL, " +

			"\"name\" blob NOT NULL, " +

			"\"data\" blob NOT NULL, " +

			"UNIQUE(\"name\"), " +

			"PRIMARY KEY(\"id\")" +

			");",

		"CREATE TABLE \"PicTags\" (" +

			"\"pic_id\" integer NOT NULL, " +

			"\"tag_id\" integer NOT NULL, " +

			"\"data\" blob NOT NULL, " +

			"PRIMARY KEY(\"pic_id\",\"tag_id\")" +

			");",

		"CREATE TABLE \"PicIdents\" (" +

			"\"pic_id\" integer NOT NULL, " +

			"\"type\" integer NOT NULL, " +

			"\"value\" blob NOT NULL, " +

			"\"data\" blob NOT NULL, " +

			"PRIMARY KEY(\"pic_id\",\"type\",\"value\")" +

			");",

		"CREATE INDEX \"PicIdentsIdent\" ON \"PicIdents\" (\"type\",\"value\");",

		"CREATE TABLE \"Users\" (" +

			"\"id\" integer NOT NULL, " +

			"\"ident\" blob NOT NULL, " +

			"\"data\" blob NOT NULL, " +

			"UNIQUE(\"ident\"), " +

			"PRIMARY KEY(\"id\")" +

			");",

		"CREATE TABLE \"_SequenceTable\" (\"the_sequence\" integer NOT NULL);",
	},
}

var SqlInitTables = map[string][]string{

	"mysql": {
		"INSERT INTO `_SequenceTable` (`the_sequence`) VALUES (1);",
	},

	"postgresql": {
		"INSERT INTO \"_SequenceTable\" (\"the_sequence\") VALUES (1);",
	},

	"sqlite3": {
		"INSERT INTO \"_SequenceTable\" (\"the_sequence\") VALUES (1);",
	},
}

func NewJob(DB db.DB) (*Job, error) {
	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}
	j := &Job{
		beg:  DB,
		tx:   tx,
		adap: DB.Adapter(),
	}
	runtime.SetFinalizer(j, jobCloser)
	return j, nil
}

type Job struct {
	beg  db.Beginner
	tx   db.QuerierExecutorCommitter
	adap db.DBAdapter
}

func (j *Job) Commit() error {
	defer runtime.SetFinalizer(j, nil)
	return j.tx.Commit()
}

func (j *Job) Rollback() error {
	defer runtime.SetFinalizer(j, nil)
	return j.tx.Rollback()
}

var jobCloser = func(j *Job) {
	log.Println("warning: found orphaned job")
	if err := j.Rollback(); err != nil {
		log.Println("error rolling back orphaned job", err)
	}
}

var alloc db.IDAlloc

func (j *Job) AllocID() (int64, error) {
	return db.AllocID(j.beg, &alloc, j.adap)
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

type PicsIndexOrder struct {
	IndexOrder *int64
}

var _ db.Idx = PicsIndexOrder{}

var colsPicsIndexOrder = []string{"index_order"}

func (idx PicsIndexOrder) Cols() []string {
	return colsPicsIndexOrder
}

func (idx PicsIndexOrder) Vals() (vals []interface{}) {
	var done bool

	if idx.IndexOrder != nil {
		if done {
			panic("Extra value IndexOrder")
		}
		vals = append(vals, *idx.IndexOrder)
	} else {
		done = true
	}

	return
}

var colsPics = []string{"id", "index_order", "data"}

func (j *Job) ScanPics(opts db.Opts, cb func(*schema.Pic) error) error {
	return db.Scan(j.tx, "Pics", opts, func(data []byte) error {
		var pb schema.Pic
		if err := proto.Unmarshal(data, &pb); err != nil {
			return err
		}
		return cb(&pb)
	}, j.adap)
}

func (j *Job) FindPics(opts db.Opts) (rows []*schema.Pic, err error) {
	err = j.ScanPics(opts, func(data *schema.Pic) error {
		rows = append(rows, data)
		return nil
	})
	return
}

var _ interface {
	IdCol() int64
} = (*schema.Pic)(nil)

var _ interface {
	IndexOrderCol() int64
} = (*schema.Pic)(nil)

func (j *Job) InsertPic(pb *schema.Pic) error {
	return j.InsertPicRow(&PicRow{
		Data: pb,

		Id: pb.IdCol(),

		IndexOrder: pb.IndexOrderCol(),
	})
}

func (j *Job) InsertPicRow(row *PicRow) error {
	var vals []interface{}

	vals = append(vals, row.Id)

	vals = append(vals, row.IndexOrder)

	if val, err := proto.Marshal(row.Data); err != nil {
		return err
	} else {
		vals = append(vals, val)
	}

	return db.Insert(j.tx, "Pics", colsPics, vals, j.adap)
}

var _ interface {
	IdCol() int64
} = (*schema.Pic)(nil)

var _ interface {
	IndexOrderCol() int64
} = (*schema.Pic)(nil)

func (j *Job) UpdatePic(pb *schema.Pic) error {
	return j.UpdatePicRow(&PicRow{
		Data: pb,

		Id: pb.IdCol(),

		IndexOrder: pb.IndexOrderCol(),
	})
}

func (j *Job) UpdatePicRow(row *PicRow) error {

	key := PicsPrimary{

		Id: &row.Id,
	}

	var vals []interface{}

	vals = append(vals, row.Id)

	vals = append(vals, row.IndexOrder)

	if val, err := proto.Marshal(row.Data); err != nil {
		return err
	} else {
		vals = append(vals, val)
	}

	return db.Update(j.tx, "Pics", colsPics, vals, key, j.adap)
}

func (j *Job) DeletePic(key PicsPrimary) error {
	return db.Delete(j.tx, "Pics", key, j.adap)
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

func (j *Job) ScanTags(opts db.Opts, cb func(*schema.Tag) error) error {
	return db.Scan(j.tx, "Tags", opts, func(data []byte) error {
		var pb schema.Tag
		if err := proto.Unmarshal(data, &pb); err != nil {
			return err
		}
		return cb(&pb)
	}, j.adap)
}

func (j *Job) FindTags(opts db.Opts) (rows []*schema.Tag, err error) {
	err = j.ScanTags(opts, func(data *schema.Tag) error {
		rows = append(rows, data)
		return nil
	})
	return
}

var _ interface {
	IdCol() int64
} = (*schema.Tag)(nil)

var _ interface {
	NameCol() string
} = (*schema.Tag)(nil)

func (j *Job) InsertTag(pb *schema.Tag) error {
	return j.InsertTagRow(&TagRow{
		Data: pb,

		Id: pb.IdCol(),

		Name: pb.NameCol(),
	})
}

func (j *Job) InsertTagRow(row *TagRow) error {
	var vals []interface{}

	vals = append(vals, row.Id)

	vals = append(vals, row.Name)

	if val, err := proto.Marshal(row.Data); err != nil {
		return err
	} else {
		vals = append(vals, val)
	}

	return db.Insert(j.tx, "Tags", colsTags, vals, j.adap)
}

var _ interface {
	IdCol() int64
} = (*schema.Tag)(nil)

var _ interface {
	NameCol() string
} = (*schema.Tag)(nil)

func (j *Job) UpdateTag(pb *schema.Tag) error {
	return j.UpdateTagRow(&TagRow{
		Data: pb,

		Id: pb.IdCol(),

		Name: pb.NameCol(),
	})
}

func (j *Job) UpdateTagRow(row *TagRow) error {

	key := TagsPrimary{

		Id: &row.Id,
	}

	var vals []interface{}

	vals = append(vals, row.Id)

	vals = append(vals, row.Name)

	if val, err := proto.Marshal(row.Data); err != nil {
		return err
	} else {
		vals = append(vals, val)
	}

	return db.Update(j.tx, "Tags", colsTags, vals, key, j.adap)
}

func (j *Job) DeleteTag(key TagsPrimary) error {
	return db.Delete(j.tx, "Tags", key, j.adap)
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

func (j *Job) ScanPicTags(opts db.Opts, cb func(*schema.PicTag) error) error {
	return db.Scan(j.tx, "PicTags", opts, func(data []byte) error {
		var pb schema.PicTag
		if err := proto.Unmarshal(data, &pb); err != nil {
			return err
		}
		return cb(&pb)
	}, j.adap)
}

func (j *Job) FindPicTags(opts db.Opts) (rows []*schema.PicTag, err error) {
	err = j.ScanPicTags(opts, func(data *schema.PicTag) error {
		rows = append(rows, data)
		return nil
	})
	return
}

var _ interface {
	PicIdCol() int64
} = (*schema.PicTag)(nil)

var _ interface {
	TagIdCol() int64
} = (*schema.PicTag)(nil)

func (j *Job) InsertPicTag(pb *schema.PicTag) error {
	return j.InsertPicTagRow(&PicTagRow{
		Data: pb,

		PicId: pb.PicIdCol(),

		TagId: pb.TagIdCol(),
	})
}

func (j *Job) InsertPicTagRow(row *PicTagRow) error {
	var vals []interface{}

	vals = append(vals, row.PicId)

	vals = append(vals, row.TagId)

	if val, err := proto.Marshal(row.Data); err != nil {
		return err
	} else {
		vals = append(vals, val)
	}

	return db.Insert(j.tx, "PicTags", colsPicTags, vals, j.adap)
}

var _ interface {
	PicIdCol() int64
} = (*schema.PicTag)(nil)

var _ interface {
	TagIdCol() int64
} = (*schema.PicTag)(nil)

func (j *Job) UpdatePicTag(pb *schema.PicTag) error {
	return j.UpdatePicTagRow(&PicTagRow{
		Data: pb,

		PicId: pb.PicIdCol(),

		TagId: pb.TagIdCol(),
	})
}

func (j *Job) UpdatePicTagRow(row *PicTagRow) error {

	key := PicTagsPrimary{

		PicId: &row.PicId,

		TagId: &row.TagId,
	}

	var vals []interface{}

	vals = append(vals, row.PicId)

	vals = append(vals, row.TagId)

	if val, err := proto.Marshal(row.Data); err != nil {
		return err
	} else {
		vals = append(vals, val)
	}

	return db.Update(j.tx, "PicTags", colsPicTags, vals, key, j.adap)
}

func (j *Job) DeletePicTag(key PicTagsPrimary) error {
	return db.Delete(j.tx, "PicTags", key, j.adap)
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

func (j *Job) ScanPicIdents(opts db.Opts, cb func(*schema.PicIdent) error) error {
	return db.Scan(j.tx, "PicIdents", opts, func(data []byte) error {
		var pb schema.PicIdent
		if err := proto.Unmarshal(data, &pb); err != nil {
			return err
		}
		return cb(&pb)
	}, j.adap)
}

func (j *Job) FindPicIdents(opts db.Opts) (rows []*schema.PicIdent, err error) {
	err = j.ScanPicIdents(opts, func(data *schema.PicIdent) error {
		rows = append(rows, data)
		return nil
	})
	return
}

var _ interface {
	PicIdCol() int64
} = (*schema.PicIdent)(nil)

var _ interface {
	TypeCol() schema.PicIdent_Type
} = (*schema.PicIdent)(nil)

var _ interface {
	ValueCol() []byte
} = (*schema.PicIdent)(nil)

func (j *Job) InsertPicIdent(pb *schema.PicIdent) error {
	return j.InsertPicIdentRow(&PicIdentRow{
		Data: pb,

		PicId: pb.PicIdCol(),

		Type: pb.TypeCol(),

		Value: pb.ValueCol(),
	})
}

func (j *Job) InsertPicIdentRow(row *PicIdentRow) error {
	var vals []interface{}

	vals = append(vals, row.PicId)

	vals = append(vals, row.Type)

	vals = append(vals, row.Value)

	if val, err := proto.Marshal(row.Data); err != nil {
		return err
	} else {
		vals = append(vals, val)
	}

	return db.Insert(j.tx, "PicIdents", colsPicIdents, vals, j.adap)
}

var _ interface {
	PicIdCol() int64
} = (*schema.PicIdent)(nil)

var _ interface {
	TypeCol() schema.PicIdent_Type
} = (*schema.PicIdent)(nil)

var _ interface {
	ValueCol() []byte
} = (*schema.PicIdent)(nil)

func (j *Job) UpdatePicIdent(pb *schema.PicIdent) error {
	return j.UpdatePicIdentRow(&PicIdentRow{
		Data: pb,

		PicId: pb.PicIdCol(),

		Type: pb.TypeCol(),

		Value: pb.ValueCol(),
	})
}

func (j *Job) UpdatePicIdentRow(row *PicIdentRow) error {

	key := PicIdentsPrimary{

		PicId: &row.PicId,

		Type: &row.Type,

		Value: &row.Value,
	}

	var vals []interface{}

	vals = append(vals, row.PicId)

	vals = append(vals, row.Type)

	vals = append(vals, row.Value)

	if val, err := proto.Marshal(row.Data); err != nil {
		return err
	} else {
		vals = append(vals, val)
	}

	return db.Update(j.tx, "PicIdents", colsPicIdents, vals, key, j.adap)
}

func (j *Job) DeletePicIdent(key PicIdentsPrimary) error {
	return db.Delete(j.tx, "PicIdents", key, j.adap)
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

func (j *Job) ScanUsers(opts db.Opts, cb func(*schema.User) error) error {
	return db.Scan(j.tx, "Users", opts, func(data []byte) error {
		var pb schema.User
		if err := proto.Unmarshal(data, &pb); err != nil {
			return err
		}
		return cb(&pb)
	}, j.adap)
}

func (j *Job) FindUsers(opts db.Opts) (rows []*schema.User, err error) {
	err = j.ScanUsers(opts, func(data *schema.User) error {
		rows = append(rows, data)
		return nil
	})
	return
}

var _ interface {
	IdCol() int64
} = (*schema.User)(nil)

var _ interface {
	IdentCol() string
} = (*schema.User)(nil)

func (j *Job) InsertUser(pb *schema.User) error {
	return j.InsertUserRow(&UserRow{
		Data: pb,

		Id: pb.IdCol(),

		Ident: pb.IdentCol(),
	})
}

func (j *Job) InsertUserRow(row *UserRow) error {
	var vals []interface{}

	vals = append(vals, row.Id)

	vals = append(vals, row.Ident)

	if val, err := proto.Marshal(row.Data); err != nil {
		return err
	} else {
		vals = append(vals, val)
	}

	return db.Insert(j.tx, "Users", colsUsers, vals, j.adap)
}

var _ interface {
	IdCol() int64
} = (*schema.User)(nil)

var _ interface {
	IdentCol() string
} = (*schema.User)(nil)

func (j *Job) UpdateUser(pb *schema.User) error {
	return j.UpdateUserRow(&UserRow{
		Data: pb,

		Id: pb.IdCol(),

		Ident: pb.IdentCol(),
	})
}

func (j *Job) UpdateUserRow(row *UserRow) error {

	key := UsersPrimary{

		Id: &row.Id,
	}

	var vals []interface{}

	vals = append(vals, row.Id)

	vals = append(vals, row.Ident)

	if val, err := proto.Marshal(row.Data); err != nil {
		return err
	} else {
		vals = append(vals, val)
	}

	return db.Update(j.tx, "Users", colsUsers, vals, key, j.adap)
}

func (j *Job) DeleteUser(key UsersPrimary) error {
	return db.Delete(j.tx, "Users", key, j.adap)
}
