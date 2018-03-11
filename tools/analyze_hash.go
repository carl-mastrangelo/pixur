package main

import (
	"container/heap"
	"context"
	"encoding/binary"
	"encoding/csv"
	"flag"
	"log"
	"os"
	"sort"
	"strconv"

	"pixur.org/pixur/be/imaging"
	"pixur.org/pixur/be/schema"
	sdb "pixur.org/pixur/be/schema/db"
	tab "pixur.org/pixur/be/schema/tables"
	"pixur.org/pixur/be/server/config"
)

var allMedians = []float32{
	-1.726552, 2.964345, -0.13845398, -3.3160644, 0.51064676, -0.11798997, 4.7331654e-30, 1.4199496e-29,
	-3.9443045e-30, 4.9267707, -0.24924812, -6.854629, 3.9443045e-30, 7.099748e-30, 3.1554436e-29, -0.5742262,
	0.9638992, -0.4710116, -0.065479964, 4.7331654e-30, -7.099748e-30, 0.30259687, -0.33178842, -0.42226526,
	-7.888609e-31, -1.4325413, -0.014203334, -1.0259316, -6.661338e-16, 1.5777218e-30, 0.034086976, -0.18520246,
	0.5318168, -0.527036, -2.2088105e-29, -0.4211659, 0.5060927, 0.09501513, -0.35342145, 0.046108875,
	-0.08382997, -0.77674377, -0.034913637, 1.0007774, -0.10231467, 7.888609e-31, -0.072848566, 0.60153186,
	-0.03843284, -1.2586465, -2.2088105e-29, 0.41432515, -0.24322185, 0.011488877, 0.008951891, 7.888609e-30,
	0.3603853, -0.14286107, -0.44190237, 0.09469462, -7.099748e-30, 8.67747e-30, 0.44547608, 0.24279852}

type imgDiff struct {
	leftBits, rightBits uint64
	diff                int
	ids                 pair
}

type pair struct {
	leftId, rightId int64
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
	db, err := sdb.Open(config.Conf.DbName, config.Conf.DbConfig)
	if err != nil {
		return err
	}
	defer db.Close()

	pis, err := getIdents(db)
	if err != nil {
		return err
	}

	log.Println("meds", allMedians)

	hist, bitCounts := hashBitHistogram(pis)
	log.Println("Found", hist, bitCounts)

	comp1 := findSimilar(pis)
	firstPairs := make(map[pair]*imgDiff, comp1.Len())

	for comp1.Len() > 0 {
		d := heap.Pop(comp1).(imgDiff)
		firstPairs[d.ids] = &d
	}

	allMedians = findMedians(pis)
	updateHashes(pis)

	comp2 := findSimilar(pis)
	secondPairs := make(map[pair]*imgDiff, comp2.Len())

	for comp2.Len() > 0 {
		d := heap.Pop(comp2).(imgDiff)
		secondPairs[d.ids] = &d
	}

	allDiffs, firstOnly, secondOnly := compareHashes(firstPairs, secondPairs)
	log.Println(len(allDiffs), len(firstOnly), len(secondOnly))

	var comp = make(imgDiffs, 0)
	heap.Init(&comp)
	for _, d := range secondOnly {
		heap.Push(&comp, d)
	}

	if err := writeDiffs(&comp); err != nil {
		return err
	}

	return nil
}

type float32Slice []float32

func (f float32Slice) Len() int           { return len(f) }
func (f float32Slice) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }
func (f float32Slice) Less(i, j int) bool { return f[i] < f[j] }

func findMedians(pis []*schema.PicIdent) []float32 {
	vals := make([][]float32, 64)
	meds := make([]float32, 0, 64)

	for _, pi := range pis {
		for i := 0; i < 64; i++ {
			vals[i] = append(vals[i], pi.Dct0Values[i])
		}
	}
	for _, val := range vals {
		sort.Sort(float32Slice(val))
		// handles even and odd lists
		left := val[(len(val)-1)/2]
		right := val[(len(val))/2]
		meds = append(meds, (left+right)/2)
	}
	return meds
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func compareHashes(first, second map[pair]*imgDiff) (allDiffs, firstOnly, secondOnly imgDiffs) {
	allKeys := make(map[pair]struct{}, len(first))
	for k, _ := range first {
		allKeys[k] = struct{}{}
	}
	for k, _ := range second {
		allKeys[k] = struct{}{}
	}

	for k, _ := range allKeys {
		left, leftPresent := first[k]
		right, rightPresent := second[k]
		if leftPresent && rightPresent {
			// we don't care if one of them was not a match anyways
			if (left.diff > 10) == (right.diff > 10) {
				continue
			}
			allDiffs = append(allDiffs, imgDiff{
				diff: left.diff - right.diff,
				ids:  k,
			})
		} else if leftPresent && !rightPresent {
			if left.diff > 10 {
				continue
			}
			firstOnly = append(firstOnly, *left)
		} else if !leftPresent && rightPresent {
			if right.diff > 10 {
				continue
			}
			secondOnly = append(secondOnly, *right)
		} else {
			panic(k)
		}
	}
	return
}

func updateHashes(pis []*schema.PicIdent) {
	for _, pi := range pis {
		var hash uint64
		for i := uint(0); i < 64; i++ {
			if pi.Dct0Values[i] > allMedians[i] {
				hash |= 1 << i
			}
		}
		binary.BigEndian.PutUint64(pi.Value, hash)
	}
}

func writeDiffs(h heap.Interface) error {
	f, err := os.Create("diffs.csv")
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)

	i := 0
	for h.Len() > 0 {
		d := heap.Pop(h).(imgDiff)
		if i++; i < 2000 {
			log.Println(d.diff, schema.Varint(d.ids.leftId), schema.Varint(d.ids.rightId))
		}
		dd, dl, dr := strconv.Itoa(d.diff), strconv.FormatInt(d.ids.leftId, 10), strconv.FormatInt(d.ids.rightId, 10)
		if err := w.Write([]string{dd, dl, dr}); err != nil {
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}
	return nil
}

func findSimilar(pis []*schema.PicIdent) heap.Interface {
	var comp = make(imgDiffs, 0)
	heap.Init(&comp)
	for i := 0; i < len(pis); i++ {
		leftBits := binary.BigEndian.Uint64(pis[i].Value)
		if imaging.CountBits(leftBits) < 20 {
			continue // skip, probably not worth our time
		}
		for k := i + 1; k < len(pis)-1; k++ {
			rightBits := binary.BigEndian.Uint64(pis[k].Value)
			if imaging.CountBits(rightBits) < 20 {
				continue // skip, probably not worth our time
			}
			if count := imaging.CountBits(leftBits ^ rightBits); count <= 20 {
				heap.Push(&comp, imgDiff{
					//leftBits:  leftBits,
					//rightBits: rightBits,
					diff: count,
					ids: pair{
						pis[i].PicId,
						pis[k].PicId},
				})
			}
		}
	}
	return &comp
}

func hashBitHistogram(pis []*schema.PicIdent) ([]int, map[int]int) {
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

func getIdents(db sdb.DB) ([]*schema.PicIdent, error) {
	j, err := tab.NewJob(context.Background(), db)
	if err != nil {
		return nil, err
	}
	defer j.Rollback()

	pis, err := j.FindPicIdents(sdb.Opts{})
	if err != nil {
		return nil, err
	}
	return pis, nil
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		log.Println(err)
	}
}
