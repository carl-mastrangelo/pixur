package tasks

import (
	"context"
	"sync"
	"testing"

	"pixur.org/pixur/be/schema/db"
)

// Rather than have schema/db depend on a live database, do the id allocator tests here.

func TestAllocDBSerial(t *testing.T) {
	c := Container(t)
	defer c.Close()
	db.AllocatorGrab = 1

	alloc := new(db.IDAlloc)
	d := c.DB()
	ids := make(map[int64]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		num, err := db.AllocID(context.Background(), d, alloc, d.Adapter())
		if err != nil {
			t.Error(err)
		}
		ids[num] = struct{}{}
	}
	if len(ids) != 1000 {
		t.Error("wrong number of ids", len(ids))
	}
}

func TestAllocDBParallel(t *testing.T) {
	c := Container(t)
	defer c.Close()
	db.AllocatorGrab = 1

	alloc := new(db.IDAlloc)
	d := c.DB()
	idschan := make(chan int64, 1000)
	var wg sync.WaitGroup
	wg.Add(1000)
	for i := 0; i < 1000; i++ {
		go func() {
			defer wg.Done()
			num, err := db.AllocID(context.Background(), d, alloc, d.Adapter())
			if err != nil {
				t.Error(err)
			}
			idschan <- num
		}()
	}
	wg.Wait()
	close(idschan)
	ids := make(map[int64]struct{}, 1000)
	for num := range idschan {
		ids[num] = struct{}{}
	}

	if len(ids) != 1000 {
		t.Error("wrong number of ids", len(ids))
	}
}

func TestAllocDBSerialMulti(t *testing.T) {
	c := Container(t)
	defer c.Close()
	db.AllocatorGrab = 100

	alloc := new(db.IDAlloc)
	d := c.DB()
	ids := make(map[int64]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		num, err := db.AllocID(context.Background(), d, alloc, d.Adapter())
		if err != nil {
			t.Error(err)
		}
		ids[num] = struct{}{}
	}
	if len(ids) != 1000 {
		t.Error("wrong number of ids", len(ids))
	}
}

func TestAllocDBParallelMulti(t *testing.T) {
	c := Container(t)
	defer c.Close()
	db.AllocatorGrab = 100

	alloc := new(db.IDAlloc)
	d := c.DB()
	idschan := make(chan int64, 1000)
	var wg sync.WaitGroup
	wg.Add(1000)
	for i := 0; i < 1000; i++ {
		go func() {
			defer wg.Done()
			num, err := db.AllocID(context.Background(), d, alloc, d.Adapter())
			if err != nil {
				t.Error(err)
			}
			idschan <- num
		}()
	}
	wg.Wait()
	close(idschan)
	ids := make(map[int64]struct{}, 1000)
	for num := range idschan {
		ids[num] = struct{}{}
	}

	if len(ids) != 1000 {
		t.Error("wrong number of ids", len(ids))
	}
}

func TestAllocJobSerial(t *testing.T) {
	c := Container(t)
	defer c.Close()

	alloc := new(db.IDAlloc)
	d := c.DB()
	j, err := d.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer j.Rollback()
	ids := make(map[int64]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		num, err := db.AllocIDJob(j, alloc, d.Adapter())
		if err != nil {
			t.Error(err)
		}
		ids[num] = struct{}{}
	}
	if len(ids) != 1000 {
		t.Error("wrong number of ids", len(ids))
	}
}

func TestAllocMixed(t *testing.T) {
	c := Container(t)
	defer c.Close()
	db.AllocatorGrab = 100

	alloc := new(db.IDAlloc)
	d := c.DB()
	num0, err := db.AllocID(context.Background(), d, alloc, d.Adapter())
	if err != nil {
		t.Error(err)
	}

	j1, err := d.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer j1.Rollback()
	num1, err := db.AllocIDJob(j1, alloc, d.Adapter())

	if num1 != num0+100 {
		t.Error(num1, num0)
	}
	if err := j1.Rollback(); err != nil {
		t.Error(err)
	}

	j2, err := d.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer j2.Rollback()
	num2, err := db.AllocIDJob(j2, alloc, d.Adapter())
	if num2 != num0+100 {
		t.Error(num2, num0)
	}
	if err := j2.Commit(); err != nil {
		t.Error(err)
	}

	num3, err := db.AllocID(context.Background(), d, alloc, d.Adapter())
	if err != nil {
		t.Error(err)
	}
	if num3 != num0+1 {
		t.Error(num3, num0)
	}
}

func BenchmarkAllocDBSerial(b *testing.B) {
	c := Container(b)
	defer c.Close()
	db.AllocatorGrab = 1

	d := c.DB()
	ids := make(map[int64]struct{}, b.N)

	b.ResetTimer()
	alloc := new(db.IDAlloc)

	for i := 0; i < b.N; i++ {
		num, err := db.AllocID(context.Background(), d, alloc, d.Adapter())
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
	db.AllocatorGrab = 100

	d := c.DB()
	ids := make(map[int64]struct{}, b.N)

	b.ResetTimer()
	alloc := new(db.IDAlloc)

	for i := 0; i < b.N; i++ {
		num, err := db.AllocID(context.Background(), d, alloc, d.Adapter())
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
