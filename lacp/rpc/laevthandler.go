// laevthandler.go
package rpc

import (
	"asicd/asicdConstDefs"
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/op/go-nanomsg"
	"infra/portd/portdCommonDefs"
	lacp "l2/lacp/protocol"
)

const (
	SUB_PORTD = 0
	SUB_ASICD = 1
)

var PortdSub *nanomsg.SubSocket

func processLinkDownEvent(linkType uint8, linkId uint8) {
	var p *lacp.LaAggPort
	if lacp.LaFindPortById(uint16(linkId), &p) {
		//if p.IsPortEnabled() {
		p.LaAggPortDisable()
		p.LinkOperStatus = false
		//}
	}
}

func processLinkUpEvent(linkType uint8, linkId uint8) {
	var p *lacp.LaAggPort
	if lacp.LaFindPortById(uint16(linkId), &p) {
		//if p.IsPortAdminEnabled() && !p.IsPortOperStatusUp() {
		p.LaAggPortEnabled()
		p.LinkOperStatus = true
		//}
	}
}

func processAsicdEvents(sub *nanomsg.SubSocket) {

	fmt.Println("in process Asicd events")
	for {
		fmt.Println("In for loop Asicd events")
		rcvdMsg, err := sub.Recv(0)
		if err != nil {
			fmt.Println("Error in receiving ", err)
			return
		}
		fmt.Println("After recv rcvdMsg buf", rcvdMsg)
		buf := bytes.NewReader(rcvdMsg)
		var MsgType asicdConstDefs.AsicdNotifyMsg
		err = binary.Read(buf, binary.LittleEndian, &MsgType)
		if err != nil {
			fmt.Println("Error in reading msgtype ", err)
			return
		}
		switch MsgType {
		case asicdConstDefs.NOTIFY_LINK_STATE_CHANGE:
			var msg asicdConstDefs.LinkStateInfo
			err = binary.Read(buf, binary.LittleEndian, &msg)
			if err != nil {
				fmt.Println("Error in reading msg ", err)
				return
			}
			fmt.Printf("Msg linkstatus = %d msg port = %d\n", msg.LinkStatus, msg.Port)
			if msg.LinkStatus == asicdConstDefs.LINK_STATE_DOWN {
				processLinkDownEvent(portdCommonDefs.PHY, msg.Port) //asicd always sends out link state events for PHY ports
			} else {
				processLinkUpEvent(portdCommonDefs.PHY, msg.Port)
			}
		}
	}
}

func processEvents(sub *nanomsg.SubSocket, subType int) {
	fmt.Println("in process events for sub ", subType)
	if subType == SUB_ASICD {
		fmt.Println("process portd events")
		processAsicdEvents(sub)
	}
}
func setupEventHandler(sub *nanomsg.SubSocket, address string, subtype int) {
	fmt.Println("Setting up event handlers for sub type ", subtype)
	sub, err := nanomsg.NewSubSocket()
	if err != nil {
		fmt.Println("Failed to open sub socket")
		return
	}
	fmt.Println("opened socket")
	ep, err := sub.Connect(address)
	if err != nil {
		fmt.Println("Failed to connect to pub socket - ", ep)
		return
	}
	fmt.Println("Connected to ", ep.Address)
	err = sub.Subscribe("")
	if err != nil {
		fmt.Println("Failed to subscribe to all topics")
		return
	}
	fmt.Println("Subscribed")
	err = sub.SetRecvBuffer(1024 * 1204)
	if err != nil {
		fmt.Println("Failed to set recv buffer size")
		return
	}
	processEvents(sub, subtype)
}

func startEvtHandler() {
	go setupEventHandler(PortdSub, asicdConstDefs.PUB_SOCKET_ADDR, SUB_ASICD)
}
