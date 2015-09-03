package commands

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"time"

	humanize "github.com/ipfs/go-ipfs/Godeps/_workspace/src/github.com/dustin/go-humanize"

	cmds "github.com/ipfs/go-ipfs/commands"
	metrics "github.com/ipfs/go-ipfs/metrics"
	peer "github.com/ipfs/go-ipfs/p2p/peer"
	protocol "github.com/ipfs/go-ipfs/p2p/protocol"
	u "github.com/ipfs/go-ipfs/util"
)

var StatsCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline:          "Query IPFS statistics",
		ShortDescription: ``,
	},

	Subcommands: map[string]*cmds.Command{
		"bw": statBwCmd,
	},
}

var statBwCmd = &cmds.Command{
	Helptext: cmds.HelpText{
		Tagline:          "Print ipfs bandwidth information",
		ShortDescription: ``,
	},
	Options: []cmds.Option{
		cmds.StringOption("peer", "p", "specify a peer to print bandwidth for"),
		cmds.StringOption("proto", "t", "specify a protocol to print bandwidth for"),
		cmds.BoolOption("poll", "print bandwidth at an interval"),
		cmds.StringOption("interval", "i", "time interval to wait between updating output"),
	},

	Run: func(req cmds.Request, res cmds.Response) {
		nd, err := req.InvocContext().GetNode()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		// Must be online!
		if !nd.OnlineMode() {
			res.SetError(errNotOnline, cmds.ErrClient)
			return
		}

		pstr, pfound, err := req.Option("peer").String()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		tstr, tfound, err := req.Option("proto").String()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}
		if pfound && tfound {
			res.SetError(errors.New("please only specify peer OR protocol"), cmds.ErrClient)
			return
		}

		var pid peer.ID
		if pfound {
			checkpid, err := peer.IDB58Decode(pstr)
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}
			pid = checkpid
		}

		interval := time.Second
		timeS, found, err := req.Option("interval").String()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}
		if found {
			v, err := time.ParseDuration(timeS)
			if err != nil {
				res.SetError(err, cmds.ErrNormal)
				return
			}
			interval = v
		}

		doPoll, _, err := req.Option("poll").Bool()
		if err != nil {
			res.SetError(err, cmds.ErrNormal)
			return
		}

		out := make(chan interface{})
		res.SetOutput((<-chan interface{})(out))

		go func() {
			defer close(out)
			for {
				if pfound {
					stats := nd.Reporter.GetBandwidthForPeer(pid)
					out <- &stats
				} else if tfound {
					protoId := protocol.ID(tstr)
					stats := nd.Reporter.GetBandwidthForProtocol(protoId)
					out <- &stats
				} else {
					totals := nd.Reporter.GetBandwidthTotals()
					out <- &totals
				}
				if !doPoll {
					return
				}
				select {
				case <-time.After(interval):
				case <-req.Context().Done():
					return
				}
			}
		}()
	},
	Type: metrics.Stats{},
	Marshalers: cmds.MarshalerMap{
		cmds.Text: func(res cmds.Response) (io.Reader, error) {
			outCh, ok := res.Output().(<-chan interface{})
			if !ok {
				return nil, u.ErrCast()
			}

			polling, _, err := res.Request().Option("poll").Bool()
			if err != nil {
				return nil, err
			}

			first := true
			marshal := func(v interface{}) (io.Reader, error) {
				bs, ok := v.(*metrics.Stats)
				if !ok {
					return nil, u.ErrCast()
				}
				out := new(bytes.Buffer)
				if !polling {
					printStats(out, bs)
				} else {
					if first {
						fmt.Fprintln(out, "Total Up\t Total Down\t Rate Up\t Rate Down")
						first = false
					}
					fmt.Fprint(out, "\r")
					fmt.Fprintf(out, "%s \t\t", humanize.Bytes(uint64(bs.TotalOut)))
					fmt.Fprintf(out, " %s \t\t", humanize.Bytes(uint64(bs.TotalIn)))
					fmt.Fprintf(out, " %s/s   \t", humanize.Bytes(uint64(bs.RateOut)))
					fmt.Fprintf(out, " %s/s     ", humanize.Bytes(uint64(bs.RateIn)))
				}
				return out, nil

			}

			return &cmds.ChannelMarshaler{
				Channel:   outCh,
				Marshaler: marshal,
			}, nil
		},
	},
}

func printStats(out io.Writer, bs *metrics.Stats) {
	fmt.Fprintln(out, "Bandwidth")
	fmt.Fprintf(out, "TotalIn: %s\n", humanize.Bytes(uint64(bs.TotalIn)))
	fmt.Fprintf(out, "TotalOut: %s\n", humanize.Bytes(uint64(bs.TotalOut)))
	fmt.Fprintf(out, "RateIn: %s/s\n", humanize.Bytes(uint64(bs.RateIn)))
	fmt.Fprintf(out, "RateOut: %s/s\n", humanize.Bytes(uint64(bs.RateOut)))
}
