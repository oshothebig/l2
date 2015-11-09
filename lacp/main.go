// main
package main

import (
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"l2/lacp/protocol"
	"lacpd"
)

func main() {

	var transport thrift.TServerTransport
	var err error
	var addr = "localhost:9090"

	transport, err = thrift.NewTServerSocket(addr)
	if err != nil {
		panic(fmt.Sprintf("Failed to create Socket with:", addr))
	}
	handler := NewLaServiceHandler()
	processor := lacpd.NewLaServiceProcessor(handler)
	transportFactory := thrift.NewTBufferedTransportFactory(8192)
	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	server := thrift.NewTSimpleServer4(processor, transport, transportFactory, protocolFactory)
	fmt.Println("Starting LACP Thrift daemon")
	
	// register the tx func
	LacpSysGlobalInfoGet(LaSystemIdDefault).LaSysGlobalRegisterTxCallback('eth0', TxViaLinuxIf)
	
	server.Serve()

}
