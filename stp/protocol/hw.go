// hw.go
package stp

import (
	hwconst "asicd/asicdConstDefs"
	"asicd/pluginManager/pluginCommon"
	"asicdServices"
	"encoding/json"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"io/ioutil"
	"net"
	"strconv"
	"strings"
	"time"
	"utils/ipcutils"
)

type STPClientBase struct {
	Address            string
	Transport          thrift.TTransport
	PtrProtocolFactory *thrift.TBinaryProtocolFactory
	IsConnected        bool
}

type AsicdClient struct {
	STPClientBase
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
		StpLogger("ERROR", fmt.Sprintf("Error in reading configuration file:%s err:%s\n", paramsFile, err))
		return 0
	}

	err = json.Unmarshal(bytes, &clientsList)
	if err != nil {
		StpLogger("ERROR", "Error in Unmarshalling Json")
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

		for {
			asicdclnt.Address = "localhost:" + strconv.Itoa(port)
			asicdclnt.Transport, asicdclnt.PtrProtocolFactory, _ = ipcutils.CreateIPCHandles(asicdclnt.Address)
			//StpLogger("INFO", fmt.Sprintf("found asicd at port %d Transport %#v PrtProtocolFactory %#v\n", port, asicdclnt.Transport, asicdclnt.PtrProtocolFactory))
			if asicdclnt.Transport != nil && asicdclnt.PtrProtocolFactory != nil {
				StpLogger("INFO", "connecting to asicd\n")
				asicdclnt.ClientHdl = asicdServices.NewASICDServicesClientFactory(asicdclnt.Transport, asicdclnt.PtrProtocolFactory)
				asicdclnt.IsConnected = true
				// lets gather all info needed from asicd such as the port
				ConstructPortConfigMap()
				break
			} else {
				StpLogger("WARNING", "Unable to connect to ASICD, retrying in 500ms")
				time.Sleep(time.Millisecond * 500)
			}
		}
	}
}

func ConstructPortConfigMap() {
	currMarker := asicdServices.Int(hwconst.MIN_SYS_PORTS)
	if asicdclnt.IsConnected {
		StpLogger("INFO", "Calling asicd for port config")
		count := asicdServices.Int(hwconst.MAX_SYS_PORTS)
		for {
			bulkInfo, err := asicdclnt.ClientHdl.GetBulkPortState(currMarker, count)
			if err != nil {
				StpLogger("ERROR", fmt.Sprintf("GetBulkPortState Error: %s", err))
				return
			}
			StpLogger("INFO", fmt.Sprintf("Length of GetBulkPortState: %d", bulkInfo.Count))

			bulkCfgInfo, err := asicdclnt.ClientHdl.GetBulkPortConfig(currMarker, count)
			if err != nil {
				StpLogger("ERROR", fmt.Sprintf("Error: %s", err))
				return
			}

			StpLogger("INFO", fmt.Sprintf("Length of GetBulkPortConfig: %d", bulkCfgInfo.Count))
			objCount := int(bulkInfo.Count)
			more := bool(bulkInfo.More)
			currMarker = asicdServices.Int(bulkInfo.EndIdx)
			for i := 0; i < objCount; i++ {
				ifindex := bulkInfo.PortStateList[i].IfIndex
				ent := PortConfigMap[ifindex]
				ent.PortNum = bulkInfo.PortStateList[i].PortNum
				ent.IfIndex = ifindex
				ent.Name = bulkInfo.PortStateList[i].Name
				ent.HardwareAddr, _ = net.ParseMAC(bulkCfgInfo.PortConfigList[i].MacAddr)
				PortConfigMap[ifindex] = ent
				StpLogger("INIT", fmt.Sprintf("Found Port %d IfIndex %d Name %s\n", ent.PortNum, ent.IfIndex, ent.Name))
			}
			if more == false {
				return
			}
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

func asicdGetPortLinkStatus(intfNum string) bool {

	if asicdclnt.ClientHdl != nil {
		bulkInfo, err := asicdclnt.ClientHdl.GetBulkPortState(asicdServices.Int(hwconst.MIN_SYS_PORTS), asicdServices.Int(hwconst.MIN_SYS_PORTS))
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

func asicdCreateStgBridge(vlanList []uint16) int32 {

	vl := make([]int32, len(vlanList))
	if asicdclnt.ClientHdl != nil {
		for v := range vlanList {
			vl = append(vl, int32(v))
		}
		stgId, err := asicdclnt.ClientHdl.CreateStg(vl)
		if err == nil {
			return stgId
		}
	}
	return -1
}

func asicdDeleteStgBridge(stgid int32) error {

	if asicdclnt.ClientHdl != nil {

		_, err := asicdclnt.ClientHdl.DeleteStg(stgid)
		if err != nil {
			return err
		}
	}
	return nil
}

func asicdSetStgPortState(stgid int32, ifindex int32, state int) error {
	if asicdclnt.ClientHdl != nil {
		for _, pc := range PortConfigMap {
			if pc.IfIndex == ifindex {
				_, err := asicdclnt.ClientHdl.SetPortStpState(stgid, pc.PortNum, int32(state))
				return err
			}
		}
	}
	return nil
}
