package main

import (
	"flag"
	"fmt"
	"github.com/janelia-flyem/serviceproxy/register"
	"os"
)

const defaultPort = 25123

var (
        proxy  = flag.String("proxy", "", "")
	portNum  = flag.Int("port", defaultPort, "")
	showHelp = flag.Bool("help", false, "")
)

const helpMessage = `
Launches service that computes the overlap of a set of cells.

Usage: adderexample <registry address>
      -proxy    (string)        Server and port number for proxy
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

	// register service
	if flag.NArg() != 1 {
		fmt.Printf("Must provide registry address")
		fmt.Printf(helpMessage)
		os.Exit(0)
	}

	// creates adder service and points to first argument
	serfagent := register.NewAgent("calcoverlap", *portNum)
	serfagent.RegisterService(flag.Arg(0))

	Serve(*proxy, *portNum)
}
