package kbucket

import (
	"container/list"
	peer "github.com/ipfs/go-ipfs/p2p/peer"
	"sort"
)

// A helper struct to sort peers by their distance to the local node
type peerDistance struct {
	p        peer.ID
	distance ID
}

// peerSorterArr implements sort.Interface to sort peers by xor distance
type peerSorterArr []*peerDistance

func (p peerSorterArr) Len() int      { return len(p) }
func (p peerSorterArr) Swap(a, b int) { p[a], p[b] = p[b], p[a] }
func (p peerSorterArr) Less(a, b int) bool {
	return p[a].distance.less(p[b].distance)
}

//

func copyPeersFromList(target ID, peerArr peerSorterArr, peerList *list.List) peerSorterArr {
	for e := peerList.Front(); e != nil; e = e.Next() {
		p := e.Value.(peer.ID)
		pID := ConvertPeerID(p)
		pd := peerDistance{
			p:        p,
			distance: xor(target, pID),
		}
		peerArr = append(peerArr, &pd)
		if e == nil {
			log.Debug("list element was nil")
			return peerArr
		}
	}
	return peerArr
}

func SortClosestPeers(peers []peer.ID, target ID) []peer.ID {
	var psarr peerSorterArr
	for _, p := range peers {
		pID := ConvertPeerID(p)
		pd := &peerDistance{
			p:        p,
			distance: xor(target, pID),
		}
		psarr = append(psarr, pd)
	}
	sort.Sort(psarr)
	var out []peer.ID
	for _, p := range psarr {
		out = append(out, p.p)
	}
	return out
}
