// laevthandler.go
package rpc

import (
	"encoding/json"
	"fmt"
	"github.com/op/go-nanomsg"
	"infra/portd/portdCommonDefs"
)

var PortdSub *nanomsg.SubSocket

func processLinkDownEvent(linkType uint8, linkId uint8) {

}

func processLinkUpEvent(linkType uint8, linkId uint8) {

}

func processPortdEvents(sub *nanomsg.SubSocket) {

	fmt.Println("in process Port events")
	for {
		fmt.Println("In for loop")
		rcvdMsg, err := sub.Recv(0)
		if err != nil {
			fmt.Println("Error in receiving")
			return
		}
		fmt.Println("After recv rcvdMsg buf", rcvdMsg)
		MsgType := portdCommonDefs.PortdNotifyMsg{}
		err = json.Unmarshal(rcvdMsg, &MsgType)
		if err != nil {
			fmt.Println("Error in Unmarshalling rcvdMsg Json")
			return
		}
		fmt.Printf(" MsgTYpe=%v\n", MsgType)
		switch MsgType.MsgType {
		case portdCommonDefs.NOTIFY_LINK_STATE_CHANGE:
			fmt.Println("received link state change event")
			msg := portdCommonDefs.LinkStateInfo{}
			err = json.Unmarshal(MsgType.MsgBuf, &msg)
			if err != nil {
				fmt.Println("Error in Unmarshalling msgType Json")
				return
			}
			fmt.Printf("Msg linkstatus = %d msg linktype = %d linkId = %d\n", msg.LinkStatus, msg.LinkType, msg.LinkId)
			if msg.LinkStatus == portdCommonDefs.LINK_STATE_DOWN {
				processLinkDownEvent(msg.LinkType, msg.LinkId)
			} else {
				processLinkUpEvent(msg.LinkType, msg.LinkId)
			}
		}
	}
}
func processEvents(sub *nanomsg.SubSocket, subType int) {
	fmt.Println("in process events for sub ", subType)
	if subType == SUB_PORTD {
		fmt.Println("process portd events")
		processPortdEvents(sub)
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
	go setupEventHandler(PortdSub, portdCommonDefs.PUB_SOCKET_ADDR, SUB_PORTD)
}
