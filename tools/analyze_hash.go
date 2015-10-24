package main

import (
	"container/heap"
	"database/sql"
	"encoding/binary"
	"log"

	"pixur.org/pixur/image"
	"pixur.org/pixur/schema"
	"pixur.org/pixur/tools/batch"
)

type imgDiff struct {
	leftBits, rightBits uint64
	diff                int
	leftId, rightId     int64
}

type imgDiffs []imgDiff

func (id imgDiffs) Len() int {
	return len(id)
}

func (id imgDiffs) Less(i, j int) bool {
	return id[i].diff < id[j].diff
}

func (id imgDiffs) Swap(i, j int) {
	id[i], id[j] = id[j], id[i]
}

func (id *imgDiffs) Push(x interface{}) {
	*id = append(*id, x.(imgDiff))
}

func (id *imgDiffs) Pop() interface{} {
	old := *id
	n := len(old)
	x := old[n-1]
	*id = old[0 : n-1]
	return x
}

func run() error {
	db, err := getDB()
	if err != nil {
		return err
	}

	pis, err := getIdents(db)
	if err != nil {
		return err
	}

	hist, bitCounts := hashBitHistogram(pis)
	log.Println("Found", hist, bitCounts)

	comp := findSimilar(pis)
	log.Println(comp.Len())
	for i := 0; i < 100 && comp.Len() > 0; i++ {
		diff := heap.Pop(comp).(imgDiff)
		log.Println(diff.diff, schema.B32Varint(diff.leftId), schema.B32Varint(diff.rightId))
	}
	return nil
}

func findSimilar(pis []*schema.PicIdentifier) heap.Interface {
	var comp = make(imgDiffs, 0)
	heap.Init(&comp)
	for i := 0; i < len(pis); i++ {
		leftBits := binary.BigEndian.Uint64(pis[i].Value)
		for k := i + 1; k < len(pis)-1; k++ {
			rightBits := binary.BigEndian.Uint64(pis[k].Value)
			if count := image.CountBits(leftBits ^ rightBits); count <= 10 {
				heap.Push(&comp, imgDiff{
					leftBits:  leftBits,
					rightBits: rightBits,
					diff:      count,
					leftId:    pis[i].PicId,
					rightId:   pis[k].PicId,
				})
			}
		}
	}
	return &comp
}

func hashBitHistogram(pis []*schema.PicIdentifier) ([]int, map[int]int) {
	hist := make([]int, 64)
	histCount := make(map[int]int)
	for _, pi := range pis {
		bits := binary.BigEndian.Uint64(pi.Value)
		bitCount := 0
		for i := uint(0); i < 64; i++ {
			if (bits & (1 << i)) > 0 {
				hist[i]++
				bitCount++
			}
		}
		histCount[bitCount]++
	}
	return hist, histCount
}

func getDB() (*sql.DB, error) {
	var db *sql.DB
	err := batch.ForEachPic(func(p *schema.Pic, sc *batch.ServerConfig, err error) error {
		if err != nil {
			return err
		}
		if db != nil {
			return nil
		}
		db = sc.DB
		return nil
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func getIdents(db *sql.DB) ([]*schema.PicIdentifier, error) {
	stmt, err := schema.PicIdentifierPrepare("SELECT * FROM_ WHERE %s = ? ORDER BY %s;",
		db, schema.PicIdentColType, schema.PicIdentColPicId)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	pis, err := schema.FindPicIdentifiers(stmt, schema.PicIdentifier_DCT_0)
	if err != nil {
		return nil, err
	}
	return pis, nil
}

func main() {
	if err := run(); err != nil {
		log.Println(err)
	}
}
