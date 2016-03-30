// hw.go
package lacp

import (
	hwconst "asicd/asicdConstDefs"
	"asicd/pluginManager/pluginCommon"
	"asicdServices"
	"encoding/json"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"io/ioutil"
	"strconv"
	"strings"
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
	ClientHdl *asicdServices.ASICDServicesClient
}

type ClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
}

var asicdclnt AsicdClient

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
		asicdclnt.Transport, asicdclnt.PtrProtocolFactory, _ = ipcutils.CreateIPCHandles(asicdclnt.Address)
		if asicdclnt.Transport != nil && asicdclnt.PtrProtocolFactory != nil {
			fmt.Println("connecting to asicd\n")
			asicdclnt.ClientHdl = asicdServices.NewASICDServicesClientFactory(asicdclnt.Transport, asicdclnt.PtrProtocolFactory)
			asicdclnt.IsConnected = true
		}
	}
}

// convert the lacp port names name to asic format string list
func asicDPortBmpFormatGet(distPortList []string) string {
	s := ""
	dLength := len(distPortList)

	for i := 0; i < dLength; i++ {
		var num string
		if strings.Contains(distPortList[i], "-") {
			num = strings.Split(distPortList[i], "-")[1]
		} else if strings.HasPrefix(distPortList[i], "eth") {
			num = strings.TrimLeft(distPortList[i], "eth")
		} else if strings.HasPrefix(distPortList[i], "fpPort") {
			num = strings.TrimLeft(distPortList[i], "fpPort")
		}
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
	if asicdclnt.ClientHdl != nil {
		hwAggId, _ = asicdclnt.ClientHdl.CreateLag(asicDHashModeGet(a.LagHash),
			asicDPortBmpFormatGet(a.DistributedPortNumList))
		a.LacpDebug.logger.Info(fmt.Sprintf("asicDCreateLag : id %d hash %d hwhash %d portList %s\n", hwAggId, a.LagHash, asicDHashModeGet(a.LagHash), asicDPortBmpFormatGet(a.DistributedPortNumList)))
	}
	return hwAggId
}

// delete the lag
func asicDDeleteLag(a *LaAggregator) {
	if asicdclnt.ClientHdl != nil {
		asicdclnt.ClientHdl.DeleteLag(a.HwAggId)
	}
}

// update the lag ports or hashing algorithm
func asicDUpdateLag(a *LaAggregator) {

	if asicdclnt.ClientHdl != nil {
		asicdclnt.ClientHdl.UpdateLag(a.HwAggId,
			asicDHashModeGet(a.LagHash),
			asicDPortBmpFormatGet(a.DistributedPortNumList))
		a.LacpDebug.logger.Info(fmt.Sprintf("asicDUpdateLag : id %d hash %d hwhash %d portList %s\n", a.HwAggId, a.LagHash, asicDHashModeGet(a.LagHash), asicDPortBmpFormatGet(a.DistributedPortNumList)))
	}
}

func asicdGetPortLinkStatus(intfNum string) bool {

	if asicdclnt.ClientHdl != nil {
		bulkInfo, err := asicdclnt.ClientHdl.GetBulkPortState(hwconst.MIN_SYS_PORTS, hwconst.MAX_SYS_PORTS)
		if err == nil && bulkInfo.Count != 0 {
			objCount := int64(bulkInfo.Count)
			for i := int64(0); i < objCount; i++ {
				if bulkInfo.PortStateList[i].Name == intfNum {
					return bulkInfo.PortStateList[i].OperState == pluginCommon.UpDownState[1]
				}
			}
		}
		fmt.Printf("asicDGetPortLinkSatus: could not get status for port %s, failure in get method\n", intfNum)
	}
	return true

}

func asicdGetIfName(ifindex int32) string {
	if asicdclnt.ClientHdl != nil {
		bulkInfo, err := asicdclnt.ClientHdl.GetBulkPortState(hwconst.MIN_SYS_PORTS, hwconst.MAX_SYS_PORTS)
		if err == nil && bulkInfo.Count != 0 {
			objCount := int64(bulkInfo.Count)
			for i := int64(0); i < objCount; i++ {
				if bulkInfo.PortStateList[i].IfIndex == ifindex {
					return bulkInfo.PortStateList[i].Name
				}
			}
		}
		fmt.Printf("asicDGetPortLinkSatus: could not get status for port %s, failure in get method\n", ifindex)
	}
	return ""
}
