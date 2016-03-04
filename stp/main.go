// main
package main

import (
	"flag"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	stp "l2/stp/protocol"
	"l2/stp/rpc"
	"stpd"
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
	asicdConfName := path + "asicd.conf"

	port := stp.GetClientPort(fileName, "stpd")
	if port != 0 {
		addr := fmt.Sprintf("localhost:%d", port)
		transport, err = thrift.NewTServerSocket(addr)
		if err != nil {
			panic(fmt.Sprintf("Failed to create Socket with:", addr))
		}

		handler := rpc.NewSTPDServiceHandler()
		processor := stpd.NewSTPDServicesProcessor(handler)
		transportFactory := thrift.NewTBufferedTransportFactory(8192)
		protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
		server := thrift.NewTSimpleServer4(processor, transport, transportFactory, protocolFactory)

		// connect to any needed services
		stp.SaveSwitchMac(asicdConfName)
		stp.ConnectToClients(fileName)

		// lets replay any config that is in the db
		handler.ReadConfigFromDB(path)

		stp.StpLogger("INFO", "Starting STP Thrift daemon")
		err = server.Serve()
		stp.StpLogger("ERROR", "ERROR server not started")
		panic(err)
	}
}
