// Copyright 2021 KVStore Authors
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

// NOTE: The code in this file is based on code from the
// SQLEngine project, licensed under the Apache License v 2.0
//
// https://github.com/acme/sqlengine/tree/cc5e161ac06827589c4966674597c137cc9e809c/store/kvstore/mockstore/deadlock/deadlock.go
//

// Copyright 2019 Acme, Inc.
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

package deadlock

import (
	_ "fmt"
	"sync"
)

// Detector detects deadlock.
type Detector struct {
	waitForMap map[uint64]*txnList
	lock       sync.Mutex
}

type txnList struct {
	txns []txnKeyHashPair
}

type txnKeyHashPair struct {
	txn     uint64
	keyHash uint64
}

// NewDetector creates a new Detector.
func NewDetector() *Detector {
	panic("excised: NewDetector")
}

// ErrDeadlock is returned when deadlock is detected.
type ErrDeadlock struct {
	KeyHash uint64
}

func (e *ErrDeadlock) Error() string {
	panic("excised: ErrDeadlock.Error")
}

// Detect detects deadlock for the sourceTxn on a locked key.
func (d *Detector) Detect(sourceTxn, waitForTxn, keyHash uint64) *ErrDeadlock {
	panic("excised: Detector.Detect")
}

func (d *Detector) doDetect(sourceTxn, waitForTxn uint64) *ErrDeadlock {
	panic("excised: Detector.doDetect")
}

func (d *Detector) register(sourceTxn, waitForTxn, keyHash uint64) {
	panic("excised: Detector.register")
}

// CleanUp removes the wait for entry for the transaction.
func (d *Detector) CleanUp(txn uint64) {
	panic("excised: Detector.CleanUp")
}

// CleanUpWaitFor removes a key in the wait for entry for the transaction.
func (d *Detector) CleanUpWaitFor(txn, waitForTxn, keyHash uint64) {
	panic("excised: Detector.CleanUpWaitFor")
}

// Expire removes entries with TS smaller than minTS.
func (d *Detector) Expire(minTS uint64) {
	panic("excised: Detector.Expire")
}
