package main

import (
	"flag"
	"fmt"
	"github.com/janelia-flyem/overlapservice/overlap"
	"github.com/janelia-flyem/serviceproxy/register"
	"os"
)

const defaultPort = 25123

var (
	proxy    = flag.String("proxy", "", "")
	registry = flag.String("registry", "", "")
	portNum  = flag.Int("port", defaultPort, "")
	showHelp = flag.Bool("help", false, "")
)

const helpMessage = `
Launches service that computes the overlap of a set of cells.

Usage: adderexample
      -proxy    (string)        Server and port number for proxy address of serviceproxy
      -registry (string)        Server and port number for registry address of serviceproxy
      -port     (number)        Port for HTTP server
  -h, -help     (flag)          Show help message
`

func main() {
	flag.BoolVar(showHelp, "h", false, "Show help message")
	flag.Parse()

	if *showHelp {
		fmt.Printf(helpMessage)
		os.Exit(0)
	}

	if *registry != "" {
		// creates adder service and points to first argument
		serfagent := register.NewAgent("calcoverlap", *portNum)
		serfagent.RegisterService(*registry)
	}

	overlap.Serve(*proxy, *portNum)
}
