package tasks

import (
	"context"
	"sync"
	"testing"

	"pixur.org/pixur/be/schema/db"
	"pixur.org/pixur/be/status"
)

var _ Task = (fakeIdAllocTask)(nil)

type fakeIdAllocTask func() error

func (t fakeIdAllocTask) Run(context.Context) status.S {
	if err := t(); err != nil {
		return status.From(err)
	}
	return nil
}

func runIdAllocTest(c *TestContainer, fn func() error) {
	c.T.Helper()
	idAllocTaskRunner := &TaskRunner{
		logger: silentLogger,
	}
	if sts := idAllocTaskRunner.Run(c.Ctx, fakeIdAllocTask(fn)); sts != nil {
		c.T.Fatal(sts)
	}
}

// Rather than have schema/db depend on a live database, do the id allocator tests here.
func TestAllocDBSerial(t *testing.T) {
	c := Container(t)
	defer c.Close()

	alloc := new(db.IdAlloc)
	alloc.SetWatermark(1, 1)
	d := c.DB()
	ids := make(map[int64]int, 100)
	for i := 0; i < 100; i++ {
		runIdAllocTest(c, func() error {
			num, err := db.AllocId(c.Ctx, d, alloc, d.Adapter())
			if err != nil {
				return err
			}
			ids[num]++
			return nil
		})
	}
	if len(ids) != 100 {
		t.Error("wrong number of ids", len(ids))
	}
	for i, val := range ids {
		if val != 1 {
			t.Error("bad id count", i, val)
		}
	}
}

func TestAllocDBParallel(t *testing.T) {
	c := Container(t)
	defer c.Close()

	alloc := new(db.IdAlloc)
	alloc.SetWatermark(1, 1)
	d := c.DB()
	idschan := make(chan int64, 100)
	var wg sync.WaitGroup
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func() {
			defer wg.Done()
			runIdAllocTest(c, func() error {
				num, err := db.AllocId(c.Ctx, d, alloc, d.Adapter())
				if err != nil {
					return err
				}
				idschan <- num
				return nil
			})
		}()
	}
	wg.Wait()
	close(idschan)
	ids := make(map[int64]int, 100)
	for num := range idschan {
		ids[num]++
	}

	if len(ids) != 100 {
		t.Error("wrong number of ids", len(ids))
	}
	for i, val := range ids {
		if val != 1 {
			t.Error("bad id count", i, val)
		}
	}
}

func TestAllocDBSerialMulti(t *testing.T) {
	c := Container(t)
	defer c.Close()

	alloc := new(db.IdAlloc)
	alloc.SetWatermark(1, 10)
	d := c.DB()
	ids := make(map[int64]int, 100)
	for i := 0; i < 100; i++ {
		runIdAllocTest(c, func() error {
			num, err := db.AllocId(c.Ctx, d, alloc, d.Adapter())
			if err != nil {
				return err
			}
			ids[num]++
			return nil
		})
	}
	if len(ids) != 100 {
		t.Error("wrong number of ids", len(ids))
	}
	for i, val := range ids {
		if val != 1 {
			t.Error("bad id count", i, val)
		}
	}
}

func TestAllocDBParallelMulti(t *testing.T) {
	c := Container(t)
	defer c.Close()

	alloc := new(db.IdAlloc)
	alloc.SetWatermark(1, 10)
	d := c.DB()
	idschan := make(chan int64, 100)
	var wg sync.WaitGroup
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func() {
			defer wg.Done()
			runIdAllocTest(c, func() error {
				num, err := db.AllocId(c.Ctx, d, alloc, d.Adapter())
				if err != nil {
					return err
				}
				idschan <- num
				return nil
			})
		}()
	}
	wg.Wait()
	close(idschan)
	ids := make(map[int64]int, 100)
	for num := range idschan {
		ids[num]++
	}
	if len(ids) != 100 {
		t.Error("wrong number of ids", len(ids))
	}
	for i, val := range ids {
		if val != 1 {
			t.Error("bad id count", i, val)
		}
	}
}

func TestAllocJobSerial(t *testing.T) {
	c := Container(t)
	defer c.Close()

	alloc := new(db.IdAlloc)
	d := c.DB()
	j, err := d.Begin(c.Ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	defer j.Rollback()
	ids := make(map[int64]int, 100)
	for i := 0; i < 100; i++ {
		num, err := db.AllocIdJob(c.Ctx, j, alloc, d.Adapter())
		if err != nil {
			t.Fatal(err)
		}
		ids[num]++
	}
	if len(ids) != 100 {
		t.Error("wrong number of ids", len(ids))
	}
	for i, val := range ids {
		if val != 1 {
			t.Error("bad id count", i, val)
		}
	}
}

func TestAllocMixed(t *testing.T) {
	c := Container(t)
	defer c.Close()

	alloc := new(db.IdAlloc)
	alloc.SetWatermark(1, 10)
	d := c.DB()
	num0, err := db.AllocId(c.Ctx, d, alloc, d.Adapter())
	if err != nil {
		t.Error(err)
	}

	j1, err := d.Begin(c.Ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	defer j1.Rollback()
	num1, err := db.AllocIdJob(c.Ctx, j1, alloc, d.Adapter())

	if num1 != num0+1 {
		t.Error(num1, num0)
	}
	if err := j1.Rollback(); err != nil {
		t.Error(err)
	}

	j2, err := d.Begin(c.Ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	defer j2.Rollback()
	num2, err := db.AllocIdJob(c.Ctx, j2, alloc, d.Adapter())
	if num2 != num0+2 {
		t.Error(num2, num0)
	}
	if err := j2.Commit(); err != nil {
		t.Error(err)
	}

	num3, err := db.AllocId(c.Ctx, d, alloc, d.Adapter())
	if err != nil {
		t.Error(err)
	}
	if num3 != num0+3 {
		t.Error(num3, num0)
	}
}

func BenchmarkAllocDBSerial(b *testing.B) {
	c := Container(b)
	defer c.Close()

	d := c.DB()
	ids := make(map[int64]struct{}, b.N)

	b.ResetTimer()
	alloc := new(db.IdAlloc)
	alloc.SetWatermark(1, 1)

	for i := 0; i < b.N; i++ {
		num, err := db.AllocId(c.Ctx, d, alloc, d.Adapter())
		if err != nil {
			b.Error(err)
		}
		ids[num] = struct{}{}
	}
	if len(ids) != b.N {
		b.Error("wrong number of ids", len(ids))
	}
	b.StopTimer()
}

func BenchmarkAllocDBSerialMulti(b *testing.B) {
	c := Container(b)
	defer c.Close()

	d := c.DB()
	ids := make(map[int64]struct{}, b.N)

	b.ResetTimer()
	alloc := new(db.IdAlloc)
	alloc.SetWatermark(1, 100)

	for i := 0; i < b.N; i++ {
		num, err := db.AllocId(c.Ctx, d, alloc, d.Adapter())
		if err != nil {
			b.Error(err)
		}
		ids[num] = struct{}{}
	}
	if len(ids) != b.N {
		b.Error("wrong number of ids", len(ids))
	}
	b.StopTimer()
}
