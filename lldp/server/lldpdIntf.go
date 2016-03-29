package lldpServer

import (
	"asicd/asicdConstDefs"
	"asicdServices"
	"encoding/json"
	"fmt"
	nanomsg "github.com/op/go-nanomsg"
	_ "utils/commonDefs"
)

func (svr *LLDPServer) LLDPGetInfoFromAsicd() error {
	svr.logger.Info("Calling Asicd to initialize port properties")
	err := svr.LLDPRegisterWithAsicdUpdates(asicdConstDefs.PUB_SOCKET_ADDR)
	if err == nil {
		// Asicd subscriber thread
		go svr.LLDPAsicdSubscriber()
	}
	// Get Port List Most Likely Not needed...as we are only interested
	// in Ipv4Intf...
	svr.LLDPGetPortList()
	// Get Vlan List
	svr.LLDPGetVlanList()
	return nil
}

func (svr *LLDPServer) LLDPRegisterWithAsicdUpdates(address string) error {
	var err error
	svr.logger.Info("setting up asicd update listener")
	if svr.asicdSubSocket, err = nanomsg.NewSubSocket(); err != nil {
		svr.logger.Err(fmt.Sprintln("Failed to create ASIC subscribe",
			"socket, error:", err))
		return err
	}

	if err = svr.asicdSubSocket.Subscribe(""); err != nil {
		svr.logger.Err(fmt.Sprintln("Failed to subscribe to \"\" on",
			"ASIC subscribe socket, error:",
			err))
		return err
	}

	if _, err = svr.asicdSubSocket.Connect(address); err != nil {
		svr.logger.Err(fmt.Sprintln("Failed to connect to ASIC",
			"publisher socket, address:", address, "error:", err))
		return err
	}

	if err = svr.asicdSubSocket.SetRecvBuffer(1024 * 1024); err != nil {
		svr.logger.Err(fmt.Sprintln("Failed to set the buffer size for ",
			"ASIC publisher socket, error:", err))
		return err
	}
	svr.logger.Info("asicd update listener is set")
	return nil
}

func (svr *LLDPServer) LLDPAsicdSubscriber() {
	for {
		svr.logger.Info("LLDP: Read on Asic Subscriber socket....")
		rxBuf, err := svr.asicdSubSocket.Recv(0)
		if err != nil {
			svr.logger.Err(fmt.Sprintln("LLDP: Recv on asicd Subscriber",
				"socket failed with error:", err))
			continue
		}
		var msg asicdConstDefs.AsicdNotification
		err = json.Unmarshal(rxBuf, &msg)
		if err != nil {
			svr.logger.Err(fmt.Sprintln("LLDP: Unable to Unmarshal",
				"asicd msg:", msg.Msg))
			continue
		}
		if msg.MsgType == asicdConstDefs.NOTIFY_VLAN_CREATE ||
			msg.MsgType == asicdConstDefs.NOTIFY_VLAN_DELETE {
			//Vlan Create Msg
			var vlanNotifyMsg asicdConstDefs.VlanNotifyMsg
			err = json.Unmarshal(msg.Msg, &vlanNotifyMsg)
			if err != nil {
				svr.logger.Err(fmt.Sprintln("LLDP: Unable to",
					"unmashal vlanNotifyMsg:", msg.Msg))
				return
			}
			//svr.LLDPUpdateVlanGblInfo(vlanNotifyMsg, msg.MsgType)
		} else if msg.MsgType == asicdConstDefs.NOTIFY_IPV4INTF_CREATE ||
			msg.MsgType == asicdConstDefs.NOTIFY_IPV4INTF_DELETE {
			var ipv4IntfNotifyMsg asicdConstDefs.IPv4IntfNotifyMsg
			err = json.Unmarshal(msg.Msg, &ipv4IntfNotifyMsg)
			if err != nil {
				svr.logger.Err(fmt.Sprintln("LLDP: Unable to Unmarshal",
					"ipv4IntfNotifyMsg:", msg.Msg))
				continue
			}
			//svr.LLDPUpdateIPv4GblInfo(ipv4IntfNotifyMsg, msg.MsgType)
		} else if msg.MsgType == asicdConstDefs.NOTIFY_L3INTF_STATE_CHANGE {
			//INTF_STATE_CHANGE
			var l3IntfStateNotifyMsg asicdConstDefs.L3IntfStateNotifyMsg
			err = json.Unmarshal(msg.Msg, &l3IntfStateNotifyMsg)
			if err != nil {
				svr.logger.Err(fmt.Sprintln("LLDP: unable to Unmarshal l3 intf",
					"state change:", msg.Msg))
				continue
			}
			//svr.LLDPUpdateL3IntfStateChange(l3IntfStateNotifyMsg)
		}
	}
}

func (svr *LLDPServer) LLDPGetVlanList() {
	svr.logger.Info("LLDP: Get Vlans")
	objCount := 0
	var currMarker int64
	more := false
	count := 10
	for {
		bulkInfo, err := svr.asicdClient.ClientHdl.GetBulkVlanState(
			asicdServices.Int(currMarker), asicdServices.Int(count))
		if err != nil {
			svr.logger.Err(fmt.Sprintln("DRA: getting bulk vlan config",
				"from asicd failed with reason", err))
			return
		}
		objCount = int(bulkInfo.Count)
		more = bool(bulkInfo.More)
		currMarker = int64(bulkInfo.EndIdx)
		for i := 0; i < objCount; i++ {
			//svr.LLDPCreateVlanEntry(int(bulkInfo.VlanStateList[i].VlanId),
			//	bulkInfo.VlanStateList[i].VlanName)
		}
		if more == false {
			break
		}
	}
}

func (svr *LLDPServer) LLDPGetPortList() {
	svr.logger.Info("Get Port List")
	currMarker := int64(asicdConstDefs.MIN_SYS_PORTS)
	more := false
	objCount := 0
	count := 10
	for {
		bulkInfo, err := svr.asicdClient.ClientHdl.GetBulkPortState(
			asicdServices.Int(currMarker), asicdServices.Int(count))
		if err != nil {
			svr.logger.Err(fmt.Sprintln("LLDP: getting bulk port config"+
				" from asicd failed with reason", err))
			return
		}
		objCount = int(bulkInfo.Count)
		more = bool(bulkInfo.More)
		currMarker = int64(bulkInfo.EndIdx)
		for i := 0; i < objCount; i++ {
			/*
				var ifName string
				var portNum int32
				portNum = bulkInfo.PortStateList[i].IfIndex
				ifName = bulkInfo.PortStateList[i].Name
				//svr.logger.Info("LLDP: interface global init for " + ifName)
				//LLDPInitGblInfo(portNum, ifName, "")
			*/
		}
		if more == false {
			break
		}
	}
}
