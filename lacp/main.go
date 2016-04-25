// main
package main

import (
	"flag"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	lacp "l2/lacp/protocol"
	"l2/lacp/rpc"
	"lacpd"
	"net"
	"utils/keepalive"
)

func main() {

	var transport thrift.TServerTransport
	var err error

	// lookup port
	paramsDir := flag.String("params", "./params", "Params directory")
	flag.Parse()
	path := *paramsDir
	if path[len(path)-1] != '/' {
		path = path + "/"
	}
	fileName := path + "clients.json"

	// Start keepalive routine
	go keepalive.InitKeepAlive("lacpd", path)

	port := lacp.GetClientPort(fileName, "lacpd")
	if port != 0 {
		addr := fmt.Sprintf("localhost:%d", port)
		transport, err = thrift.NewTServerSocket(addr)
		if err != nil {
			panic(fmt.Sprintf("Failed to create Socket with:", addr))
		}

		handler := rpc.NewLACPDServiceHandler()
		processor := lacpd.NewLACPDServicesProcessor(handler)
		transportFactory := thrift.NewTBufferedTransportFactory(8192)
		protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
		server := thrift.NewTSimpleServer4(processor, transport, transportFactory, protocolFactory)

		// connect to any needed services
		lacp.ConnectToClients(fileName)

		// lets replay any config that is in the db
		handler.ReadConfigFromDB()

		fmt.Println("Available Interfaces for use:")
		intfs, err := net.Interfaces()
		if err == nil {
			for _, intf := range intfs {
				fmt.Println(intf)
			}
			fmt.Println("Starting LACP Thrift daemon")
			err = server.Serve()
			fmt.Println("ERROR server not started")
			panic(err)
		} else {
			panic(err)
		}

	}
}
