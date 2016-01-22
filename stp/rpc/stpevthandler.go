// laevthandler.go
package rpc

import (
	"asicd/asicdConstDefs"
	"encoding/json"
	"fmt"
	"github.com/op/go-nanomsg"
)

const (
	SUB_ASICD = iota
)

var AsicdSub *nanomsg.SubSocket

func processLinkDownEvent(linkType uint8, linkId uint16) {
	fmt.Println("STP: Link Down")
}

func processLinkUpEvent(linkType uint8, linkId uint16) {
	fmt.Println("STP: Link Up")
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
		buf := asicdConstDefs.AsicdNotification{}
		err = json.Unmarshal(rcvdMsg, &buf)
		if err != nil {
			fmt.Println("Error in reading msgtype ", err)
			return
		}
		switch buf.MsgType {
		case asicdConstDefs.NOTIFY_L2INTF_STATE_CHANGE:
			var msg asicdConstDefs.L2IntfStateNotifyMsg
			err := json.Unmarshal(buf.Msg, &msg)
			if err != nil {
				fmt.Println("Error in reading msg ", err)
				return
			}
			fmt.Printf("Msg linkstatus = %d msg port = %d\n", msg.IfState, msg.IfId)
			if msg.IfState == asicdConstDefs.INTF_STATE_DOWN {
				processLinkDownEvent(msg.IfType, msg.IfId) //asicd always sends out link State events for PHY ports
			} else {
				processLinkUpEvent(msg.IfType, msg.IfId)
			}
		}
	}
}

func processEvents(sub *nanomsg.SubSocket, subType int) {
	fmt.Println("in process events for sub ", subType)
	if subType == SUB_ASICD {
		fmt.Println("process asicd events")
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
	go setupEventHandler(AsicdSub, asicdConstDefs.PUB_SOCKET_ADDR, SUB_ASICD)
}
