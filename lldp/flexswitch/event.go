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

package flexswitch

import (
	"fmt"
	"l2/lldp/config"
	"l2/lldp/utils"
	"models/events"
	"utils/eventUtils"
)

func (p *SystemPlugin) PublishEvent(info config.EventInfo) {
	eventType := info.EventType
	//entry := info.Info
	evtKey := events.LLDPIntfKey{info.IfIndex}
	/*
			LocalPort:           entry.LocalPort,
			NeighborPort:        entry.Port,
			NeighborMac:         entry.PeerMac,
			HoldTime:            entry.HoldTime,
			SystemCapabilities:  entry.SystemCapabilities,
			EnabledCapabilities: entry.EnabledCapabilities,
		}
	*/
	debug.Logger.Info(fmt.Sprintln("Publishing event Type:", eventType, "--->", evtKey))
	var err error
	switch eventType {
	case config.Learned:
		err = eventUtils.PublishEvents(events.NeighborLearned, evtKey, "")
	case config.Updated:
		err = eventUtils.PublishEvents(events.NeighborUpdated, evtKey, "")
	case config.Removed:
		err = eventUtils.PublishEvents(events.NeighborRemoved, evtKey, "")

	}
	if err != nil {
		debug.Logger.Err(fmt.Sprintln("Publishining Event for", info.IfIndex, "event Type", eventType,
			"failed with error:", err))
	}
}
