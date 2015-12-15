// main
package main

import (
	"flag"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	lacp "l2/lacp/protocol"
	"l2/lacp/rpc"
	"lacpdServices"
	"net"
)

func main() {

	var transport thrift.TServerTransport
	var err error

	// lookup port
	paramsDir := flag.String("params", "./params", "Params directory")
	flag.Parse()
	fileName := *paramsDir
	if fileName[len(fileName)-1] != '/' {
		fileName = fileName + "/"
	}
	fileName = fileName + "clients.json"

	port := lacp.GetClientPort(fileName, "lacpd")
	if port != 0 {
		addr := fmt.Sprintf("localhost:%d", port)
		transport, err = thrift.NewTServerSocket(addr)
		if err != nil {
			panic(fmt.Sprintf("Failed to create Socket with:", addr))
		}

		handler := rpc.NewLACPDServiceHandler()
		processor := lacpdServices.NewLACPDServicesProcessor(handler)
		transportFactory := thrift.NewTBufferedTransportFactory(8192)
		protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
		server := thrift.NewTSimpleServer4(processor, transport, transportFactory, protocolFactory)

		// connect to any needed services
		lacp.ConnectToClients(fileName)

		fmt.Println("Available Interfaces for use:")
		intfs, err := net.Interfaces()
		if err == nil {
			for _, intf := range intfs {
				fmt.Println(intf)
			}
			fmt.Println("Starting LACP Thrift daemon")
			server.Serve()
		} else {
			panic(err)
		}

	}
}
