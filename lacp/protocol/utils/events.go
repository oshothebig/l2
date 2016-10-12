//
//Copyright [2016] [SnapRoute Inc]
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//	 Unless required by applicable law or agreed to in writing, software
//	 distributed under the License is distributed on an "AS IS" BASIS,
//	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	 See the License for the specific language governing permissions and
//	 limitations under the License.
//
// _______  __       __________   ___      _______.____    __    ____  __  .___________.  ______  __    __
// |   ____||  |     |   ____\  \ /  /     /       |\   \  /  \  /   / |  | |           | /      ||  |  |  |
// |  |__   |  |     |  |__   \  V  /     |   (----` \   \/    \/   /  |  | `---|  |----`|  ,----'|  |__|  |
// |   __|  |  |     |   __|   >   <       \   \      \            /   |  |     |  |     |  |     |   __   |
// |  |     |  `----.|  |____ /  .  \  .----)   |      \    /\    /    |  |     |  |     |  `----.|  |  |  |
// |__|     |_______||_______/__/ \__\ |_______/        \__/  \__/     |__|     |__|      \______||__|  |__|
//
package utils

import (
	"fmt"
	"models/events"
	"utils/eventUtils"
)

// TODO need to add code in order to protect against event flapping

func ProcessLacpGroupOperStateDown(ifindex int32) {
	intfref := GetAggNameFromIfIndex(ifindex)

	if intfref != "" {
		evtKey := events.LacpEntryKey{
			IntfRef: intfref,
		}
		//evtData := EventData{
		//	IpAddr:  msg.IpAddr,
		//	MacAddr: entry.MacAddr,
		//	IfName:  entry.IfName,
		//}
		txEvent := eventUtils.TxEvent{
			EventId: events.LacpdEventPortOperStateDown,
			Key:     evtKey,
			//	AdditionalInfo: "",
			//	AdditionalData: evtData,
		}
		err := eventUtils.PublishEvents(&txEvent)
		if err != nil {
			GlobalLogger.Err("Error in publishing LacpdEventPortOperStateDown Event")
		}
	} else {
		GlobalLogger.Err(fmt.Sprintf("Error in publishing LacpdEventPortOperStateDown Event, ifindex %d not found", ifindex))
	}
}

func ProcessLacpGroupOperStateUp(ifindex int32) {
	intfref := GetAggNameFromIfIndex(ifindex)

	if intfref != "" {

		evtKey := events.LacpEntryKey{
			IntfRef: intfref,
		}
		//evtData := EventData{
		//	IpAddr:  msg.IpAddr,
		//	MacAddr: entry.MacAddr,
		//	IfName:  entry.IfName,
		//}
		txEvent := eventUtils.TxEvent{
			EventId: events.LacpdEventGroupOperStateUp,
			Key:     evtKey,
			//	AdditionalInfo: "",
			//	AdditionalData: evtData,
		}
		err := eventUtils.PublishEvents(&txEvent)
		if err != nil {
			GlobalLogger.Err("Error in publishing LacpdEventGroupOperStateUp Event")
		}
	} else {
		GlobalLogger.Err(fmt.Sprintf("Error in publishing LacpdEventGroupOperStateUp Event, ifindex %d not found", ifindex))
	}
}

func ProcessLacpPortOperStateDown(ifindex int32) {
	intfref := GetNameFromIfIndex(ifindex)

	if intfref != "" {
		evtKey := events.LacpPortEntryKey{
			IntfRef: intfref,
		}
		//evtData := EventData{
		//	IpAddr:  msg.IpAddr,
		//	MacAddr: entry.MacAddr,
		//	IfName:  entry.IfName,
		//}
		txEvent := eventUtils.TxEvent{
			EventId: events.LacpdEventPortOperStateDown,
			Key:     evtKey,
			//	AdditionalInfo: "",
			//	AdditionalData: evtData,
		}
		err := eventUtils.PublishEvents(&txEvent)
		if err != nil {
			GlobalLogger.Err("Error in publishing LacpdEventPortOperStateDown Event")
		}
	} else {
		GlobalLogger.Err(fmt.Sprintf("Error in publishing LacpdEventPortOperStateDown Event, ifindex %d not found", ifindex))
	}
}

func ProcessLacpPortOperStateUp(ifindex int32) {
	intfref := GetNameFromIfIndex(ifindex)

	if intfref != "" {
		evtKey := events.LacpPortEntryKey{
			IntfRef: intfref,
		}
		//evtData := EventData{
		//	IpAddr:  msg.IpAddr,
		//	MacAddr: entry.MacAddr,
		//	IfName:  entry.IfName,
		//}
		txEvent := eventUtils.TxEvent{
			EventId: events.LacpdEventPortOperStateUp,
			Key:     evtKey,
			//	AdditionalInfo: "",
			//	AdditionalData: evtData,
		}
		err := eventUtils.PublishEvents(&txEvent)
		if err != nil {
			GlobalLogger.Err("Error in publishing LacpdEventPortOperStateUp Event")
		}
	} else {
		GlobalLogger.Err(fmt.Sprintf("Error in publishing LacpdEventPortOperStateUp Event, ifindex %d not found", ifindex))
	}
}

func ProcessLacpPortPartnerInfoMismatch(ifindex int32) {
	intfref := GetNameFromIfIndex(ifindex)

	if intfref != "" {
		evtKey := events.LacpPortEntryKey{
			IntfRef: intfref,
		}
		//evtData := EventData{
		//	IpAddr:  msg.IpAddr,
		//	MacAddr: entry.MacAddr,
		//	IfName:  entry.IfName,
		//}
		txEvent := eventUtils.TxEvent{
			EventId: events.LacpdEventPortPartnerInfoMismatch,
			Key:     evtKey,
			//	AdditionalInfo: "",
			//	AdditionalData: evtData,
		}
		err := eventUtils.PublishEvents(&txEvent)
		if err != nil {
			GlobalLogger.Err("Error in publishing LacpdEventPortPartnerInfoMismatch Event")
		}
	} else {
		GlobalLogger.Err(fmt.Sprintf("Error in publishing LacpdEventPortPartnerInfoMismatch Event, ifindex %d not found", ifindex))
	}
}

func ProcessLacpPortPartnerInfoSync(ifindex int32) {
	intfref := GetNameFromIfIndex(ifindex)

	if intfref != "" {
		evtKey := events.LacpPortEntryKey{
			IntfRef: intfref,
		}
		//evtData := EventData{
		//	IpAddr:  msg.IpAddr,
		//	MacAddr: entry.MacAddr,
		//	IfName:  entry.IfName,
		//}
		txEvent := eventUtils.TxEvent{
			EventId: events.LacpdEventPortPartnerInfoSync,
			Key:     evtKey,
			//	AdditionalInfo: "",
			//	AdditionalData: evtData,
		}
		err := eventUtils.PublishEvents(&txEvent)
		if err != nil {
			GlobalLogger.Err("Error in publishing LacpdEventPortPartnerInfoSync Event")
		}
	} else {
		GlobalLogger.Err(fmt.Sprintf("Error in publishing LacpdEventPortPartnerInfoSync Event, ifindex %d not found", ifindex))
	}
}
