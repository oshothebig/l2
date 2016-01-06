// hw.go
package lacp

import (
	"asicdServices"
	"encoding/json"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"io/ioutil"
	"strconv"
	"utils/ipcutils"
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
/*func CreateIPCHandles(address string) (thrift.TTransport, *thrift.TBinaryProtocolFactory) {
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
}*/

func GetClientPort(paramsFile string, c string) int {
	var clientsList []ClientJson

	bytes, err := ioutil.ReadFile(paramsFile)
	if err != nil {
		fmt.Println("Error in reading configuration file:%s err:%s", paramsFile, err)
		return 0
	}

	err = json.Unmarshal(bytes, &clientsList)
	if err != nil {
		fmt.Println("Error in Unmarshalling Json")
		return 0
	}

	for _, client := range clientsList {
		if client.Name == c {
			return client.Port
		}
	}
	return 0
}

func ConnectToClients(paramsFile string) {
	port := GetClientPort(paramsFile, "asicd")
	if port != 0 {
		fmt.Printf("found asicd at port %d\n", port)
		asicdclnt.Address = "localhost:" + strconv.Itoa(port)
		asicdclnt.Transport, asicdclnt.PtrProtocolFactory = ipcutils.CreateIPCHandles(asicdclnt.Address, asicdclnt)
		if asicdclnt.Transport != nil && asicdclnt.PtrProtocolFactory != nil {
			fmt.Println("connecting to asicd\n")
			asicdclnt.ClientHdl = asicdServices.NewAsicdServiceClientFactory(asicdclnt.Transport, asicdclnt.PtrProtocolFactory)
			asicdclnt.IsConnected = true
		}
	}
}
