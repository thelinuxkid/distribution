package decision

import (
	"math"
	"testing"

	key "github.com/ipfs/go-ipfs/blocks/key"
	"github.com/ipfs/go-ipfs/exchange/bitswap/wantlist"
	"github.com/ipfs/go-ipfs/p2p/peer"
	"github.com/ipfs/go-ipfs/util/testutil"
)

// FWIW: At the time of this commit, including a timestamp in task increases
// time cost of Push by 3%.
func BenchmarkTaskQueuePush(b *testing.B) {
	q := newPRQ()
	peers := []peer.ID{
		testutil.RandPeerIDFatal(b),
		testutil.RandPeerIDFatal(b),
		testutil.RandPeerIDFatal(b),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Push(wantlist.Entry{Key: key.Key(i), Priority: math.MaxInt32}, peers[i%len(peers)])
	}
}
