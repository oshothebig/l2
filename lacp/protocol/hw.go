// hw.go
package lacp

import (
	"asicdServices"
	"encoding/json"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"io/ioutil"
	"strconv"
)

type LACPClientBase struct {
	Address            string
	Transport          thrift.TTransport
	PtrProtocolFactory *thrift.TBinaryProtocolFactory
	IsConnected        bool
}

type AsicdClient struct {
	LACPClientBase
	ClientHdl *asicdServices.AsicdServiceClient
}

type ClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
}

var asicdclnt AsicdClient

//
// This method gets Thrift related IPC handles.
//
func CreateIPCHandles(address string) (thrift.TTransport, *thrift.TBinaryProtocolFactory) {
	var transportFactory thrift.TTransportFactory
	var transport thrift.TTransport
	var protocolFactory *thrift.TBinaryProtocolFactory
	var err error

	protocolFactory = thrift.NewTBinaryProtocolFactoryDefault()
	transportFactory = thrift.NewTTransportFactory()
	transport, err = thrift.NewTSocket(address)
	transport = transportFactory.GetTransport(transport)
	if err = transport.Open(); err != nil {
		fmt.Println("Failed to Open Transport", transport, protocolFactory)
		return nil, nil
	}
	return transport, protocolFactory
}

func GetClientPort(paramsFile string, client string) int {
	var clientsList []ClientJson

	bytes, err := ioutil.ReadFile(paramsFile)
	if err != nil {
		fmt.Println("Error in reading configuration file %s", paramsFile)
		return 0
	}

	err = json.Unmarshal(bytes, &clientsList)
	if err != nil {
		fmt.Println("Error in Unmarshalling Json")
		return 0
	}

	for _, client := range clientsList {
		if client.Name == "asicd" {
			return client.Port
		}
	}
	return 0
}

func ConnectToClients(paramsFile string) {
	port := GetClientPort(paramsFile, "asicd")
	if port != 0 {
		fmt.Printf("found asicd at port %d", port)
		asicdclnt.Address = "localhost:" + strconv.Itoa(port)
		asicdclnt.Transport, asicdclnt.PtrProtocolFactory = CreateIPCHandles(asicdclnt.Address)
		if asicdclnt.Transport != nil && asicdclnt.PtrProtocolFactory != nil {
			fmt.Println("connecting to asicd")
			asicdclnt.ClientHdl = asicdServices.NewAsicdServiceClientFactory(asicdclnt.Transport, asicdclnt.PtrProtocolFactory)
			asicdclnt.IsConnected = true
		}
	}
}
