Copyright [2016] [SnapRoute Inc]

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

	 Unless required by applicable law or agreed to in writing, software
	 distributed under the License is distributed on an "AS IS" BASIS,
	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	 See the License for the specific language governing permissions and
	 limitations under the License.
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

		// Start keepalive routine
		go keepalive.InitKeepAlive("lacpd", path)

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
