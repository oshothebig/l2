// hw.go
package lacp

import (
	hwconst "asicd/asicdConstDefs"
	"asicdServices"
	"encoding/json"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"io/ioutil"
	"strconv"
	"strings"
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
// TODO move this method to different file name
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

// look up the various other daemons based on c string
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

// connect the the asic d
func ConnectToClients(paramsFile string) {
	port := GetClientPort(paramsFile, "asicd")
	if port != 0 {
		fmt.Printf("found asicd at port %d\n", port)
		asicdclnt.Address = "localhost:" + strconv.Itoa(port)
		asicdclnt.Transport, asicdclnt.PtrProtocolFactory = CreateIPCHandles(asicdclnt.Address)
		if asicdclnt.Transport != nil && asicdclnt.PtrProtocolFactory != nil {
			fmt.Println("connecting to asicd\n")
			asicdclnt.ClientHdl = asicdServices.NewAsicdServiceClientFactory(asicdclnt.Transport, asicdclnt.PtrProtocolFactory)
			asicdclnt.IsConnected = true
		}
	}
}

// convert the lacp port names name to asic format string list
func asicDPortBmpFormatGet(distPortList []string) string {
	s := ""
	dLength := len(distPortList)

	for i := 0; i < dLength; i++ {
		num := strings.Split(distPortList[i], "-")[1]
		if i == dLength-1 {
			s += num
		} else {
			s += num + ","
		}
	}
	return s

}

// convert the model value to asic value
func asicDHashModeGet(hashmode uint32) (laghash int32) {
	switch hashmode {
	case 0: //L2
		laghash = hwconst.HASH_SEL_SRCDSTMAC
		break
	case 1: //L2 + L3
		laghash = hwconst.HASH_SEL_SRCDSTIP
		break
	//case 2: //L3 + L4
	//break
	default:
		laghash = hwconst.HASH_SEL_SRCDSTMAC
	}
	return laghash
}

// create the lag with hashing algorithm and ports
func asicDCreateLag(a *LaAggregator) (hwAggId int32) {
	hwAggId, _ = asicdclnt.ClientHdl.CreateLag(asicDHashModeGet(a.LagHash),
		asicDPortBmpFormatGet(a.DistributedPortNumList))
	return hwAggId
}

// delete the lag
func asicDDeleteLag(a *LaAggregator) {
	asicdclnt.ClientHdl.DeleteLag(a.HwAggId)
}

// update the lag ports or hashing algorithm
func asicDUpdateLag(a *LaAggregator) {

	rv1, rv2 := asicdclnt.ClientHdl.UpdateLag(a.HwAggId,
		asicDHashModeGet(a.LagHash),
		asicDPortBmpFormatGet(a.DistributedPortNumList))
	fmt.Println("asicDUpdateLag:", a.HwAggId, asicDHashModeGet(a.LagHash), asicDPortBmpFormatGet(a.DistributedPortNumList), rv1, rv2)
}
