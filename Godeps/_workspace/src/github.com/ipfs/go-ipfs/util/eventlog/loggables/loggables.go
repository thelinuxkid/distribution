// Package loggables includes a bunch of transaltor functions for commonplace/stdlib
// objects. This is boilerplate code that shouldn't change much, and not sprinkled
// all over the place (i.e. gather it here).
//
// Note: it may make sense to put all stdlib Loggable functions in the eventlog
// package. Putting it here for now in case we don't want to polute it.
package loggables

import (
	"net"

	ma "github.com/ipfs/go-ipfs/Godeps/_workspace/src/github.com/jbenet/go-multiaddr"

	log "github.com/ipfs/go-ipfs/thirdparty/eventlog"

	peer "github.com/ipfs/go-ipfs/p2p/peer"
)

// NetConn returns an eventlog.Metadata with the conn addresses
func NetConn(c net.Conn) log.Loggable {
	return log.Metadata{
		"localAddr":  c.LocalAddr(),
		"remoteAddr": c.RemoteAddr(),
	}
}

// Error returns an eventlog.Metadata with an error
func Error(e error) log.Loggable {
	return log.Metadata{
		"error": e.Error(),
	}
}

// Dial metadata is metadata for dial events
func Dial(sys string, lid, rid peer.ID, laddr, raddr ma.Multiaddr) DeferredMap {
	m := DeferredMap{}
	m["subsystem"] = sys
	if lid != "" {
		m["localPeer"] = func() interface{} { return lid.Pretty() }
	}
	if laddr != nil {
		m["localAddr"] = func() interface{} { return laddr.String() }
	}
	if rid != "" {
		m["remotePeer"] = func() interface{} { return rid.Pretty() }
	}
	if raddr != nil {
		m["remoteAddr"] = func() interface{} { return raddr.String() }
	}
	return m
}

// DeferredMap is a Loggable which may contained deffered values.
type DeferredMap map[string]interface{}

// Loggable describes objects that can be marshalled into Metadata for logging
func (m DeferredMap) Loggable() map[string]interface{} {
	m2 := map[string]interface{}{}
	for k, v := range m {

		if vf, ok := v.(func() interface{}); ok {
			// if it's a DeferredVal, call it.
			m2[k] = vf()

		} else {
			// else use the value as is.
			m2[k] = v
		}
	}
	return m2
}
