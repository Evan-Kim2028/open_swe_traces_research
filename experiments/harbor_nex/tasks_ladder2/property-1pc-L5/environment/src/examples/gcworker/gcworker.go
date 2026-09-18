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

package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"example.internal/kvstore/v2/oracle"
	"example.internal/kvstore/v2/kvclient"
	"example.internal/kvstore/v2/txnkv"
)

var (
	pdAddr    = flag.String("meta", "127.0.0.1:2379", "meta address")
	safepoint = flag.Uint64("safepoint", oracle.GoTimeToTS(time.Now().Add(-24*7*time.Hour)), "safepoint")
)

func main() {
	flag.Parse()
	client, err := txnkv.NewClient([]string{*pdAddr})
	if err != nil {
		panic(err)
	}

	sysSafepoint, err := client.GC(context.Background(), *safepoint, tikv.WithConcurrency(10))
	if err != nil {
		panic(err)
	}
	fmt.Printf("Finished GC, expect safepoint:%d(%+v),real safepoint:%d(+%v)\n", *safepoint, oracle.GetTimeFromTS(*safepoint), sysSafepoint, oracle.GetTimeFromTS(sysSafepoint))

	err = client.Close()
	if err != nil {
		panic(err)
	}
}
