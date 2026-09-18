package latch

import (
	"sync"
	"testing"
)

func TestLatchExclusiveOverlap(t *testing.T) {
	sched := NewScheduler(64)
	defer sched.Close()
	var order []int
	var mu sync.Mutex
	started := make(chan struct{})
	releaseA := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		lock := sched.Lock(1, [][]byte{[]byte("k")})
		mu.Lock()
		order = append(order, 1)
		mu.Unlock()
		close(started)
		<-releaseA
		lock.SetCommitTS(10)
		sched.UnLock(lock)
	}()
	<-started
	go func() {
		defer wg.Done()
		lock := sched.Lock(2, [][]byte{[]byte("k")})
		mu.Lock()
		order = append(order, 2)
		mu.Unlock()
		lock.SetCommitTS(20)
		sched.UnLock(lock)
	}()
	close(releaseA)
	wg.Wait()
	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Fatalf("order %v (second locker must wait)", order)
	}
}

func TestLatchConcurrentSameKey(t *testing.T) {
	sched := NewScheduler(32)
	defer sched.Close()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				lock := sched.Lock(uint64(id*1000+j+1), [][]byte{[]byte("same")})
				if !lock.IsStale() {
					lock.SetCommitTS(uint64(id*1000 + j + 2))
				}
				sched.UnLock(lock)
			}
		}(i)
	}
	wg.Wait()
}
