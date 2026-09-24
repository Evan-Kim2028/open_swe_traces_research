// Copyright 2023 KVStore Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	_ "container/heap"
	"sort"
)

// Item is the interface that all entries in a priority queue must implement.
type Item interface {
	priority() uint64
	// isCanceled returns true if the item is canceled by the caller.
	isCanceled() bool
}

// entry is an entry in a priority queue.
//type entry struct {
//	entry Item
//}

// prioritySlice implements heap.Interface and holds Entries.
type prioritySlice []Item

// Len returns the length of the priority queue.
func (ps prioritySlice) Len() int {
	panic("excised: prioritySlice.Len")
}

// Less compares two entries in the priority queue.
// The higher priority entry is the one with the lower value.
func (ps prioritySlice) Less(i, j int) bool {
	panic("excised: prioritySlice.Less")
}

// Swap swaps two entries in the priority queue.
func (ps prioritySlice) Swap(i, j int) {
	panic("excised: prioritySlice.Swap")
}

// Push adds an entry to the priority queue.
func (ps *prioritySlice) Push(x interface{}) {
	panic("excised: prioritySlice.Push")
}

// Pop removes the highest priority entry from the priority queue.
func (ps *prioritySlice) Pop() interface{} {
	panic("excised: prioritySlice.Pop")
}

// PriorityQueue is a priority queue.
type PriorityQueue struct {
	ps prioritySlice
}

// NewPriorityQueue creates a new priority queue.
func NewPriorityQueue() *PriorityQueue {
	return &PriorityQueue{}
}

// Len returns the length of the priority queue.
func (pq *PriorityQueue) Len() int {
	return len(pq.ps)
}

// Push adds an entry to the priority queue.
func (pq *PriorityQueue) Push(item Item) {
	pq.ps = append(pq.ps, item)
	sort.SliceStable(pq.ps, func(i, j int) bool { return pq.ps[i].priority() > pq.ps[j].priority() })
}

// pop removes the highest priority entry from the priority queue.
func (pq *PriorityQueue) pop() Item {
	panic("excised: PriorityQueue.pop")
}

// Take returns the highest priority entries from the priority queue.
func (pq *PriorityQueue) Take(n int) []Item {
	if n <= 0 {
		return nil
	}
	if n > len(pq.ps) {
		n = len(pq.ps)
	}
	ret := pq.ps[:n]
	pq.ps = pq.ps[n:]
	return ret
}

func (pq *PriorityQueue) highestPriority() uint64 {
	if len(pq.ps) == 0 {
		return 0
	}
	return pq.ps[0].priority()
}

// all returns all entries in the priority queue not ensure the priority.
func (pq *PriorityQueue) all() []Item {
	panic("excised: PriorityQueue.all")
}

// clean removes all canceled entries from the priority queue.
func (pq *PriorityQueue) clean() {
	kept := pq.ps[:0]
	for _, it := range pq.ps {
		if !it.isCanceled() {
			kept = append(kept, it)
		}
	}
	pq.ps = kept
}

// reset clear all entry in the queue.
func (pq *PriorityQueue) reset() {
	panic("excised: PriorityQueue.reset")
}
