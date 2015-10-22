package tasks

import (
	"database/sql"
	"encoding/binary"

	"pixur.org/pixur/schema"
)

// TODO: add tests

type FindSimilarPicsTask struct {
	// Deps
	DB *sql.DB

	// Inputs
	PicID int64

	// Results
	SimilarPicIDs []int64
}

func (t *FindSimilarPicsTask) Run() error {
	t.SimilarPicIDs = make([]int64, 0) // set default, to make json encoding easier
	picStmt, err := schema.PicPrepare("SELECT * FROM_ WHERE %s = ? LOCK IN SHARE MODE;", t.DB, schema.PicColId)
	if err != nil {
		return err
	}
	defer picStmt.Close()

	p, err := schema.LookupPic(picStmt, t.PicID)
	if err != nil {
		return err
	}

	identStmt, err := schema.PicIdentifierPrepare("SELECT * FROM_ WHERE %s = ? AND %s = ?;", t.DB,
		schema.PicIdentColPicId, schema.PicIdentColType)
	if err != nil {
		return err
	}
	defer identStmt.Close()

	picIdent, err := schema.LookupPicIdentifier(identStmt, p.PicId, schema.PicIdentifier_DCT_0)
	if err != nil {
		return err
	}
	match := binary.BigEndian.Uint64(picIdent.Value)

	allIdentStmt, err := schema.PicIdentifierPrepare("SELECT * FROM_ WHERE %s = ?;", t.DB, schema.PicIdentColType)
	if err != nil {
		return err
	}
	defer allIdentStmt.Close()

	idents, err := schema.FindPicIdentifiers(allIdentStmt, schema.PicIdentifier_DCT_0)
	if err != nil {
		return err
	}

	// Linear time, sigh
	for _, ident := range idents {
		if ident.PicId == p.PicId {
			continue
		}
		guess := binary.BigEndian.Uint64(ident.Value)
		bits := guess ^ match
		bitCount := 0
		// replace this with something that isn't hideously slow.  Hamming distance would be
		// better served by a look up table or some 64 bit specific bit magic.  Cosine similarity
		// on the attached floats would also work.
		for i := uint(0); i < 64; i++ {
			if ((1 << i) & bits) > 0 {
				bitCount++
			}
		}
		if bitCount <= 10 {
			t.SimilarPicIDs = append(t.SimilarPicIDs, ident.PicId)
		}
	}

	return nil
}
