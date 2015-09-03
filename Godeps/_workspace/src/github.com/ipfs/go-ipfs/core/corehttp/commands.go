package corehttp

import (
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	cors "github.com/ipfs/go-ipfs/Godeps/_workspace/src/github.com/rs/cors"

	commands "github.com/ipfs/go-ipfs/commands"
	cmdsHttp "github.com/ipfs/go-ipfs/commands/http"
	core "github.com/ipfs/go-ipfs/core"
	corecommands "github.com/ipfs/go-ipfs/core/commands"
	config "github.com/ipfs/go-ipfs/repo/config"
)

const originEnvKey = "API_ORIGIN"
const originEnvKeyDeprecate = `You are using the ` + originEnvKey + `ENV Variable.
This functionality is deprecated, and will be removed in future versions.
Instead, try either adding headers to the config, or passing them via
cli arguments:

	ipfs config API.HTTPHeaders 'Access-Control-Allow-Origin' '*'
	ipfs daemon

or

	ipfs daemon --api-http-header 'Access-Control-Allow-Origin: *'
`

var defaultLocalhostOrigins = []string{
	"http://127.0.0.1:<port>",
	"https://127.0.0.1:<port>",
	"http://localhost:<port>",
	"https://localhost:<port>",
}

func addCORSFromEnv(c *cmdsHttp.ServerConfig) {
	origin := os.Getenv(originEnvKey)
	if origin != "" {
		log.Warning(originEnvKeyDeprecate)
		if c.CORSOpts == nil {
			c.CORSOpts.AllowedOrigins = []string{origin}
		}
		c.CORSOpts.AllowedOrigins = append(c.CORSOpts.AllowedOrigins, origin)
	}
}

func addHeadersFromConfig(c *cmdsHttp.ServerConfig, nc *config.Config) {
	log.Info("Using API.HTTPHeaders:", nc.API.HTTPHeaders)

	if acao := nc.API.HTTPHeaders[cmdsHttp.ACAOrigin]; acao != nil {
		c.CORSOpts.AllowedOrigins = acao
	}
	if acam := nc.API.HTTPHeaders[cmdsHttp.ACAMethods]; acam != nil {
		c.CORSOpts.AllowedMethods = acam
	}
	if acac := nc.API.HTTPHeaders[cmdsHttp.ACACredentials]; acac != nil {
		for _, v := range acac {
			c.CORSOpts.AllowCredentials = (strings.ToLower(v) == "true")
		}
	}

	c.Headers = nc.API.HTTPHeaders
}

func addCORSDefaults(c *cmdsHttp.ServerConfig) {
	// by default use localhost origins
	if len(c.CORSOpts.AllowedOrigins) == 0 {
		c.CORSOpts.AllowedOrigins = defaultLocalhostOrigins
	}

	// by default, use GET, PUT, POST
	if len(c.CORSOpts.AllowedMethods) == 0 {
		c.CORSOpts.AllowedMethods = []string{"GET", "POST", "PUT"}
	}
}

func patchCORSVars(c *cmdsHttp.ServerConfig, addr net.Addr) {

	// we have to grab the port from an addr, which may be an ip6 addr.
	// TODO: this should take multiaddrs and derive port from there.
	port := ""
	if tcpaddr, ok := addr.(*net.TCPAddr); ok {
		port = strconv.Itoa(tcpaddr.Port)
	} else if udpaddr, ok := addr.(*net.UDPAddr); ok {
		port = strconv.Itoa(udpaddr.Port)
	}

	// we're listening on tcp/udp with ports. ("udp!?" you say? yeah... it happens...)
	for i, o := range c.CORSOpts.AllowedOrigins {
		// TODO: allow replacing <host>. tricky, ip4 and ip6 and hostnames...
		if port != "" {
			o = strings.Replace(o, "<port>", port, -1)
		}
		c.CORSOpts.AllowedOrigins[i] = o
	}
}

func commandsOption(cctx commands.Context, command *commands.Command) ServeOption {
	return func(n *core.IpfsNode, l net.Listener, mux *http.ServeMux) (*http.ServeMux, error) {

		cfg := &cmdsHttp.ServerConfig{
			CORSOpts: &cors.Options{
				AllowedMethods: []string{"GET", "POST", "PUT"},
			},
		}

		addHeadersFromConfig(cfg, n.Repo.Config())
		addCORSFromEnv(cfg)
		addCORSDefaults(cfg)
		patchCORSVars(cfg, l.Addr())

		cmdHandler := cmdsHttp.NewHandler(cctx, command, cfg)
		mux.Handle(cmdsHttp.ApiPath+"/", cmdHandler)
		return mux, nil
	}
}

func CommandsOption(cctx commands.Context) ServeOption {
	return commandsOption(cctx, corecommands.Root)
}

func CommandsROOption(cctx commands.Context) ServeOption {
	return commandsOption(cctx, corecommands.RootRO)
}
