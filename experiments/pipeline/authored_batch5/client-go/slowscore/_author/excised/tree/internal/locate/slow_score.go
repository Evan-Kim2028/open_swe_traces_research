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

package locate

import (
	_ "fmt"
	_ "math"
	_ "sync/atomic"
	"time"
)

const (
	slowScoreInitVal         = 1
	slowScoreThreshold       = 80
	slowScoreMax             = 100
	slowScoreInitTimeoutInUs = 500000   // unit: us
	slowScoreMaxTimeoutInUs  = 30000000 // max timeout of one txn, unit: us
	slidingWindowSize        = 10       // default size of sliding window
)

// CountSlidingWindow represents the statistics on a bunch of sliding windows.
type CountSlidingWindow struct {
	avg     uint64
	sum     uint64
	history []uint64
}

// Avg returns the average value of this sliding window
func (cnt *CountSlidingWindow) Avg() uint64 {
	panic("excised: CountSlidingWindow.Avg")
}

// Sum returns the sum value of this sliding window
func (cnt *CountSlidingWindow) Sum() uint64 {
	panic("excised: CountSlidingWindow.Sum")
}

// Append adds one value into this sliding windown and returns the gradient.
func (cnt *CountSlidingWindow) Append(value uint64) (gradient float64) {
	panic("excised: CountSlidingWindow.Append")
}

// SlowScoreStat represents the statistics on business of Store.
type SlowScoreStat struct {
	avgScore            uint64
	avgTimecost         uint64
	intervalTimecost    uint64             // sum of the timecost in one counting interval. Unit: us
	intervalUpdCount    uint64             // count of update in one counting interval.
	tsCntSlidingWindow  CountSlidingWindow // sliding window on timecost
	updCntSlidingWindow CountSlidingWindow // sliding window on update count
}

func (ss *SlowScoreStat) getSlowScore() uint64 {
	panic("excised: SlowScoreStat.getSlowScore")
}

// updateSlowScore updates the statistics on SlowScore periodically.
//
// updateSlowScore will update the SlowScore of each Store according to the two factors:
//   - Requests in one timing tick. This factor can be regarded as QPS on each store.
//   - Average timecost on each request in one timing tick. This factor is used to detect
//     whether the relative store is busy on processing requests.
//
// If one Store is slow, its Requests will keep decreasing gradually, but Average timecost will
// keep ascending. And the updating algorithm just follows this mechanism and compute the
// trend of slow, by calculating gradients of slow in each tick.
func (ss *SlowScoreStat) updateSlowScore() {
	panic("excised: SlowScoreStat.updateSlowScore")
}

// recordSlowScoreStat records the timecost of each request.
func (ss *SlowScoreStat) recordSlowScoreStat(timecost time.Duration) {
	panic("excised: SlowScoreStat.recordSlowScoreStat")
}

func (ss *SlowScoreStat) markAlreadySlow() {
	panic("excised: SlowScoreStat.markAlreadySlow")
}

// resetSlowScore resets the slow score to 0. It's used for test.
func (ss *SlowScoreStat) resetSlowScore() {
	panic("excised: SlowScoreStat.resetSlowScore")
}

func (ss *SlowScoreStat) isSlow() bool {
	panic("excised: SlowScoreStat.isSlow")
}

// replicaFlowsType indicates the type of the destination replica of flows.
type replicaFlowsType int

const (
	// toLeader indicates that flows are sent to leader replica.
	toLeader replicaFlowsType = iota
	// toFollower indicates that flows are sent to followers' replica
	toFollower
	// numflowsDestType reserved to keep max replicaFlowsType value.
	numReplicaFlowsType
)

func (a replicaFlowsType) String() string {
	panic("excised: replicaFlowsType.String")
}
