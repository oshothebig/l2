package main

import (
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"lldpd"
)

const (
	IP   = "localhost"
	PORT = "10011"
)

func main() {
	fmt.Println("Starting LLDP Thrift client for Testing")
	var clientTransport thrift.TTransport

	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	transportFactory := thrift.NewTBufferedTransportFactory(8192)
	clientTransport, err := thrift.NewTSocket(IP + ":" + PORT)
	if err != nil {
		fmt.Println("NewTSocket failed with error:", err)
		return
	}

	clientTransport = transportFactory.GetTransport(clientTransport)
	if err = clientTransport.Open(); err != nil {
		fmt.Println("Failed to open the socket, error:", err)
	}
	client := lldpd.NewLLDPDServicesClientFactory(clientTransport,
		protocolFactory)

	gblConfig := lldpd.NewLLDPIntf()
	gblConfig.Enable = true
	gblConfig.IfIndex = 1
	ret, err := client.CreateLLDPIntf(gblConfig)
	if !ret {
		fmt.Println("Create LLDP Global Config Failed", err)
	} else {
		fmt.Println("Create LLDP Global Config Success")
	}
}
