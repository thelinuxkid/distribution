package commands

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"

	cmds "github.com/ipfs/go-ipfs/commands"
	swarm "github.com/ipfs/go-ipfs/p2p/net/swarm"
	peer "github.com/ipfs/go-ipfs/p2p/peer"
	iaddr "github.com/ipfs/go-ipfs/util/ipfsaddr"

	ma "github.com/ipfs/go-ipfs/Godeps/_workspace/src/github.com/jbenet/go-multiaddr"
	mafilter "github.com/ipfs/go-ipfs/Godeps/_workspace/src/github.com/whyrusleeping/multiaddr-filter"
)

type stringList struct {
	Strings []string
}

type addrMap struct {
	Addrs map[string][]string
}

var SwarmCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "swarm inspection tool",
		Synopsis: `
ipfs swarm peers                - List peers with open connections
ipfs swarm addrs                - List known addresses. Useful to debug.
ipfs swarm connect <address>    - Open connection to a given address
ipfs swarm disconnect <address> - Close connection to a given address
ipfs swarm filters				- Manipulate filters addresses
`,
		ShortDescription: `
ipfs swarm is a tool to manipulate the network swarm. The swarm is the
component that opens, listens for, and maintains connections to other
ipfs peers in the internet.
`,
	},
	Subcommands: map[string]*cmds.Command{
		"peers":      swarmPeersCmd,
		"addrs":      swarmAddrsCmd,
		"connect":    swarmConnectCmd,
		"disconnect": swarmDisconnectCmd,
		"filters":    swarmFiltersCmd,
	},
}

var swarmPeersCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "List peers with open connections",
		ShortDescription: `
ipfs swarm peers lists the set of peers this node is connected to.
`,
	},
	Run: func(req cmds.Request, res cmds.Response) {

		log.Debug("ipfs swarm peers")
		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		if n.PeerHost == nil {
			res.SetError(errNotOnline, cmds.ErrClient)
			return
		}

		conns := n.PeerHost.Network().Conns()
		addrs := make([]string, len(conns))
		for i, c := range conns {
			pid := c.RemotePeer()
			addr := c.RemoteMultiaddr()
			addrs[i] = fmt.Sprintf("%s/ipfs/%s", addr, pid.Pretty())
		}

		sort.Sort(sort.StringSlice(addrs))
		res.SetOutput(&stringList{addrs})
	},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: stringListMarshaler,
	},
	Type: stringList{},
}

var swarmAddrsCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "List known addresses. Useful to debug.",
		ShortDescription: `
ipfs swarm addrs lists all addresses this node is aware of.
`,
	},
	Subcommands: map[string]*cmds.Command{
		"local": swarmAddrsLocalCmd,
	},
	Run: func(req cmds.Request, res cmds.Response) {

		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		if n.PeerHost == nil {
			res.SetError(errNotOnline, cmds.ErrClient)
			return
		}

		addrs := make(map[string][]string)
		ps := n.PeerHost.Network().Peerstore()
		for _, p := range ps.Peers() {
			s := p.Pretty()
			for _, a := range ps.Addrs(p) {
				addrs[s] = append(addrs[s], a.String())
			}
			sort.Sort(sort.StringSlice(addrs[s]))
		}

		res.SetOutput(&addrMap{Addrs: addrs})
	},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: func(res cmds.Response) (io.Reader, error) {
			m, ok := res.Output().(*addrMap)
			if !ok {
				return nil, errors.New("failed to cast map[string]string")
			}

			// sort the ids first
			ids := make([]string, 0, len(m.Addrs))
			for p := range m.Addrs {
				ids = append(ids, p)
			}
			sort.Sort(sort.StringSlice(ids))

			buf := new(bytes.Buffer)
			for _, p := range ids {
				paddrs := m.Addrs[p]
				buf.WriteString(fmt.Sprintf("%s (%d)\n", p, len(paddrs)))
				for _, addr := range paddrs {
					buf.WriteString("\t" + addr + "\n")
				}
			}
			return buf, nil
		},
	},
	Type: addrMap{},
}

var swarmAddrsLocalCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "List local addresses.",
		ShortDescription: `
ipfs swarm addrs local lists all local addresses the node is listening on.
`,
	},
	Options: []cmds.Option{
		cmds.BoolOption("id", "Show peer ID in addresses"),
	},
	Run: func(req cmds.Request, res cmds.Response) {

		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		if n.PeerHost == nil {
			res.SetError(errNotOnline, cmds.ErrClient)
			return
		}

		showid, _, _ := req.Option("id").Bool()
		id := n.Identity.Pretty()

		var addrs []string
		for _, addr := range n.PeerHost.Addrs() {
			saddr := addr.String()
			if showid {
				saddr = path.Join(saddr, "ipfs", id)
			}
			addrs = append(addrs, saddr)
		}
		sort.Sort(sort.StringSlice(addrs))

		res.SetOutput(&stringList{addrs})
	},
	Type: stringList{},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: stringListMarshaler,
	},
}

var swarmConnectCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Open connection to a given address",
		ShortDescription: `
'ipfs swarm connect' opens a new direct connection to a peer address.

The address format is an ipfs multiaddr:

ipfs swarm connect /ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ
`,
	},
	Arguments: []cmds.Argument{
		cmds.StringArg("address", true, true, "address of peer to connect to").EnableStdin(),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		ctx := req.Context()

		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		addrs := req.Arguments()

		if n.PeerHost == nil {
			res.SetError(errNotOnline, cmds.ErrClient)
			return
		}

		pis, err := peersWithAddresses(addrs)
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		output := make([]string, len(pis))
		for i, pi := range pis {
			output[i] = "connect " + pi.ID.Pretty()

			err := n.PeerHost.Connect(ctx, pi)
			if err != nil {
				output[i] += " failure: " + err.Error()
			} else {
				output[i] += " success"
			}
		}

		res.SetOutput(&stringList{output})
	},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: stringListMarshaler,
	},
	Type: stringList{},
}

var swarmDisconnectCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Close connection to a given address",
		ShortDescription: `
'ipfs swarm disconnect' closes a connection to a peer address. The address format
is an ipfs multiaddr:

ipfs swarm disconnect /ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ
`,
	},
	Arguments: []cmds.Argument{
		cmds.StringArg("address", true, true, "address of peer to connect to").EnableStdin(),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		addrs := req.Arguments()

		if n.PeerHost == nil {
			res.SetError(errNotOnline, cmds.ErrClient)
			return
		}

		iaddrs, err := parseAddresses(addrs)
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		output := make([]string, len(iaddrs))
		for i, addr := range iaddrs {
			taddr := addr.Transport()
			output[i] = "disconnect " + addr.ID().Pretty()

			found := false
			conns := n.PeerHost.Network().ConnsToPeer(addr.ID())
			for _, conn := range conns {
				if !conn.RemoteMultiaddr().Equal(taddr) {
					log.Debug("it's not", conn.RemoteMultiaddr(), taddr)
					continue
				}

				if err := conn.Close(); err != nil {
					output[i] += " failure: " + err.Error()
				} else {
					output[i] += " success"
				}
				found = true
				break
			}

			if !found {
				output[i] += " failure: conn not found"
			}
		}
		res.SetOutput(&stringList{output})
	},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: stringListMarshaler,
	},
	Type: stringList{},
}

func stringListMarshaler(res cmds.Response) (io.Reader, error) {
	list, ok := res.Output().(*stringList)
	if !ok {
		return nil, errors.New("failed to cast []string")
	}

	buf := new(bytes.Buffer)
	for _, s := range list.Strings {
		buf.WriteString(s)
		buf.WriteString("\n")
	}
	return buf, nil
}

// parseAddresses is a function that takes in a slice of string peer addresses
// (multiaddr + peerid) and returns slices of multiaddrs and peerids.
func parseAddresses(addrs []string) (iaddrs []iaddr.IPFSAddr, err error) {
	iaddrs = make([]iaddr.IPFSAddr, len(addrs))
	for i, saddr := range addrs {
		iaddrs[i], err = iaddr.ParseString(saddr)
		if err != nil {
			return nil, cmds.ClientError("invalid peer address: " + err.Error())
		}
	}
	return
}

// peersWithAddresses is a function that takes in a slice of string peer addresses
// (multiaddr + peerid) and returns a slice of properly constructed peers
func peersWithAddresses(addrs []string) (pis []peer.PeerInfo, err error) {
	iaddrs, err := parseAddresses(addrs)
	if err != nil {
		return nil, err
	}

	for _, iaddr := range iaddrs {
		pis = append(pis, peer.PeerInfo{
			ID:    iaddr.ID(),
			Addrs: []ma.Multiaddr{iaddr.Transport()},
		})
	}
	return pis, nil
}

var swarmFiltersCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "Manipulate address filters",
		ShortDescription: `
'ipfs swarm filters' will list out currently applied filters. Its subcommands can be used
to add or remove said filters. Filters are specified using the multiaddr-filter format:

example:

    /ip4/192.168.0.0/ipcidr/16

Where the above is equivalent to the standard CIDR:

    192.168.0.0/16

Filters default to those specified under the "Swarm.AddrFilters" config key.
`,
	},
	Subcommands: map[string]*cmds.Command{
		"add": swarmFiltersAddCmd,
		"rm":  swarmFiltersRmCmd,
	},
	Run: func(req cmds.Request, res cmds.Response) {
		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		if n.PeerHost == nil {
			res.SetError(errNotOnline, cmds.ErrNormal)
			return
		}

		snet, ok := n.PeerHost.Network().(*swarm.Network)
		if !ok {
			res.SetError(errors.New("failed to cast network to swarm network"), cmds.ErrNormal)
			return
		}

		var output []string
		for _, f := range snet.Filters.Filters() {
			s, err := mafilter.ConvertIPNet(f)
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}
			output = append(output, s)
		}
		res.SetOutput(&stringList{output})
	},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: stringListMarshaler,
	},
	Type: stringList{},
}

var swarmFiltersAddCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "add an address filter",
		ShortDescription: `
'ipfs swarm filters add' will add an address filter to the daemons swarm.
Filters applied this way will not persist daemon reboots, to acheive that,
add your filters to the ipfs config file.
`,
	},
	Arguments: []cmds.Argument{
		cmds.StringArg("address", true, true, "multiaddr to filter").EnableStdin(),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		if n.PeerHost == nil {
			res.SetError(errNotOnline, cmds.ErrNormal)
			return
		}

		snet, ok := n.PeerHost.Network().(*swarm.Network)
		if !ok {
			res.SetError(errors.New("failed to cast network to swarm network"), cmds.ErrNormal)
			return
		}

		for _, arg := range req.Arguments() {
			mask, err := mafilter.NewMask(arg)
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}

			snet.Filters.AddDialFilter(mask)
		}
	},
}

var swarmFiltersRmCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline: "remove an address filter",
		ShortDescription: `
'ipfs swarm filters rm' will remove an address filter from the daemons swarm.
Filters removed this way will not persist daemon reboots, to acheive that,
remove your filters from the ipfs config file.
`,
	},
	Arguments: []cmds.Argument{
		cmds.StringArg("address", true, true, "multiaddr filter to remove").EnableStdin(),
	},
	Run: func(req cmds.Request, res cmds.Response) {
		n, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		if n.PeerHost == nil {
			res.SetError(errNotOnline, cmds.ErrNormal)
			return
		}

		snet, ok := n.PeerHost.Network().(*swarm.Network)
		if !ok {
			res.SetError(errors.New("failed to cast network to swarm network"), cmds.ErrNormal)
			return
		}

		if req.Arguments()[0] == "all" || req.Arguments()[0] == "*" {
			fs := snet.Filters.Filters()
			for _, f := range fs {
				snet.Filters.Remove(f)
			}
			return
		}

		for _, arg := range req.Arguments() {
			mask, err := mafilter.NewMask(arg)
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}

			snet.Filters.Remove(mask)
		}
	},
}
