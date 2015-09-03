package dht

import (
	"time"

	"github.com/ipfs/go-ipfs/Godeps/_workspace/src/github.com/jbenet/goprocess"
	goprocessctx "github.com/ipfs/go-ipfs/Godeps/_workspace/src/github.com/jbenet/goprocess/context"
	key "github.com/ipfs/go-ipfs/blocks/key"
	peer "github.com/ipfs/go-ipfs/p2p/peer"

	context "github.com/ipfs/go-ipfs/Godeps/_workspace/src/golang.org/x/net/context"
)

type ProviderManager struct {
	// all non channel fields are meant to be accessed only within
	// the run method
	providers map[key.Key]*providerSet
	local     map[key.Key]struct{}
	lpeer     peer.ID

	getlocal chan chan []key.Key
	newprovs chan *addProv
	getprovs chan *getProv
	period   time.Duration
	proc     goprocess.Process
}

type providerSet struct {
	providers []peer.ID
	set       map[peer.ID]time.Time
}

type addProv struct {
	k   key.Key
	val peer.ID
}

type getProv struct {
	k    key.Key
	resp chan []peer.ID
}

func NewProviderManager(ctx context.Context, local peer.ID) *ProviderManager {
	pm := new(ProviderManager)
	pm.getprovs = make(chan *getProv)
	pm.newprovs = make(chan *addProv)
	pm.providers = make(map[key.Key]*providerSet)
	pm.getlocal = make(chan chan []key.Key)
	pm.local = make(map[key.Key]struct{})
	pm.proc = goprocessctx.WithContext(ctx)
	pm.proc.Go(func(p goprocess.Process) { pm.run() })

	return pm
}

func (pm *ProviderManager) run() {
	tick := time.NewTicker(time.Hour)
	for {
		select {
		case np := <-pm.newprovs:
			if np.val == pm.lpeer {
				pm.local[np.k] = struct{}{}
			}
			provs, ok := pm.providers[np.k]
			if !ok {
				provs = newProviderSet()
				pm.providers[np.k] = provs
			}
			provs.Add(np.val)

		case gp := <-pm.getprovs:
			var parr []peer.ID
			provs, ok := pm.providers[gp.k]
			if ok {
				parr = provs.providers
			}

			gp.resp <- parr

		case lc := <-pm.getlocal:
			var keys []key.Key
			for k := range pm.local {
				keys = append(keys, k)
			}
			lc <- keys

		case <-tick.C:
			for _, provs := range pm.providers {
				var filtered []peer.ID
				for p, t := range provs.set {
					if time.Now().Sub(t) > time.Hour*24 {
						delete(provs.set, p)
					} else {
						filtered = append(filtered, p)
					}
				}
				provs.providers = filtered
			}

		case <-pm.proc.Closing():
			return
		}
	}
}

func (pm *ProviderManager) AddProvider(ctx context.Context, k key.Key, val peer.ID) {
	prov := &addProv{
		k:   k,
		val: val,
	}
	select {
	case pm.newprovs <- prov:
	case <-ctx.Done():
	}
}

func (pm *ProviderManager) GetProviders(ctx context.Context, k key.Key) []peer.ID {
	gp := &getProv{
		k:    k,
		resp: make(chan []peer.ID, 1), // buffered to prevent sender from blocking
	}
	select {
	case <-ctx.Done():
		return nil
	case pm.getprovs <- gp:
	}
	select {
	case <-ctx.Done():
		return nil
	case peers := <-gp.resp:
		return peers
	}
}

func (pm *ProviderManager) GetLocal() []key.Key {
	resp := make(chan []key.Key)
	pm.getlocal <- resp
	return <-resp
}

func newProviderSet() *providerSet {
	return &providerSet{
		set: make(map[peer.ID]time.Time),
	}
}

func (ps *providerSet) Add(p peer.ID) {
	_, found := ps.set[p]
	if !found {
		ps.providers = append(ps.providers, p)
	}

	ps.set[p] = time.Now()
}
