/*
Benchmark github.com/ipfs/go-ipfs/blockservice/worker.

Loop over a range of workers and buffer sizes and measure the time it
per block-transfer operation for each value.  Run with:

  $ go run "${GOPATH}/src/github.com/ipfs/go-ipfs/blockservice/worker/bench/main.go"
*/

package main

import (
	"log"
	"math"
	"testing"
	"time"

	ds "github.com/ipfs/go-ipfs/Godeps/_workspace/src/github.com/jbenet/go-datastore"
	ds_sync "github.com/ipfs/go-ipfs/Godeps/_workspace/src/github.com/jbenet/go-datastore/sync"
	blocks "github.com/ipfs/go-ipfs/blocks"
	blockstore "github.com/ipfs/go-ipfs/blocks/blockstore"
	worker "github.com/ipfs/go-ipfs/blockservice/worker"
	"github.com/ipfs/go-ipfs/exchange/offline"
	"github.com/ipfs/go-ipfs/thirdparty/delay"
	"github.com/ipfs/go-ipfs/util/datastore2"
)

const kEstRoutingDelay = time.Second

const kBlocksPerOp = 100

func main() {
	var bestConfig worker.Config
	var quickestNsPerOp int64 = math.MaxInt64
	for NumWorkers := 1; NumWorkers < 10; NumWorkers++ {
		for ClientBufferSize := 0; ClientBufferSize < 10; ClientBufferSize++ {
			for WorkerBufferSize := 0; WorkerBufferSize < 10; WorkerBufferSize++ {
				c := worker.Config{
					NumWorkers:       NumWorkers,
					ClientBufferSize: ClientBufferSize,
					WorkerBufferSize: WorkerBufferSize,
				}
				result := testing.Benchmark(BenchmarkWithConfig(c))
				if result.NsPerOp() < quickestNsPerOp {
					bestConfig = c
					quickestNsPerOp = result.NsPerOp()
				}
				log.Printf("benched %+v \t result: %+v", c, result)
			}
		}
	}
	log.Println(bestConfig)
}

func BenchmarkWithConfig(c worker.Config) func(b *testing.B) {
	return func(b *testing.B) {

		routingDelay := delay.Fixed(0) // during setup

		dstore := ds_sync.MutexWrap(datastore2.WithDelay(ds.NewMapDatastore(), routingDelay))
		bstore := blockstore.NewBlockstore(dstore)
		var testdata []*blocks.Block
		var i int64
		for i = 0; i < kBlocksPerOp; i++ {
			testdata = append(testdata, blocks.NewBlock([]byte(string(i))))
		}
		b.ResetTimer()
		b.SetBytes(kBlocksPerOp)
		for i := 0; i < b.N; i++ {

			b.StopTimer()
			w := worker.NewWorker(offline.Exchange(bstore), c)
			b.StartTimer()

			prev := routingDelay.Set(kEstRoutingDelay) // during measured section

			for _, block := range testdata {
				if err := w.HasBlock(block); err != nil {
					b.Fatal(err)
				}
			}

			routingDelay.Set(prev) // to hasten the unmeasured close period

			b.StopTimer()
			w.Close()
			b.StartTimer()

		}
	}
}
