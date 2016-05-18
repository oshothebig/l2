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
	"encoding/json"
	_ "errors"
	"fmt"
	nanomsg "github.com/op/go-nanomsg"
	"infra/sysd/sysdCommonDefs"
	"l2/lldp/api"
	"l2/lldp/utils"
	_ "strconv"
	_ "sysd"
	_ "time"
	_ "utils/ipcutils"
)

type SystemPlugin struct {
	sysdSubSocket *nanomsg.SubSocket
}

func NewSystemPlugin(fileName string) (*SystemPlugin, error) {
	mgr := &SystemPlugin{}
	return mgr, nil
}

func (p *SystemPlugin) connectSubSocket() error {
	var err error
	address := sysdCommonDefs.PUB_SOCKET_ADDR
	debug.Logger.Info(" setting up sysd update listener")
	if p.sysdSubSocket, err = nanomsg.NewSubSocket(); err != nil {
		debug.Logger.Err(fmt.Sprintln("Failed to create SYS subscribe socket, error:",
			err))
		return err
	}

	if err = p.sysdSubSocket.Subscribe(""); err != nil {
		debug.Logger.Err(fmt.Sprintln("Failed to subscribe to SYS subscribe socket",
			"error:", err))
		return err
	}

	if _, err = p.sysdSubSocket.Connect(address); err != nil {
		debug.Logger.Err(fmt.Sprintln("Failed to connect to SYS publisher socket",
			"address:", address, "error:", err))
		return err
	}

	debug.Logger.Info(fmt.Sprintln(" Connected to SYS publisher at address:", address))
	if err = p.sysdSubSocket.SetRecvBuffer(1024 * 1024); err != nil {
		debug.Logger.Err(fmt.Sprintln(" Failed to set the buffer size for SYS publisher",
			"socket, error:", err))
		return err
	}
	debug.Logger.Info("sysd update listener is set")
	return nil
}

func (p *SystemPlugin) listenSystemdUpdates() {
	for {
		debug.Logger.Debug(" Read on System Subscriber socket....")
		rxBuf, err := p.sysdSubSocket.Recv(0)
		if err != nil {
			debug.Logger.Err(fmt.Sprintln(
				"Recv on sysd Subscriber socket failed with error:", err))
			continue
		}
		var msg sysdCommonDefs.Notification
		err = json.Unmarshal(rxBuf, &msg)
		if err != nil {
			debug.Logger.Err(fmt.Sprintln("Unable to Unmarshal sysd err:", err))
			continue
		}
		switch msg.Type {
		case sysdCommonDefs.SYSTEM_Info:
			api.UpdateCache()
		}
	}
}

func (p *SystemPlugin) Start() {
	err := p.connectSubSocket()
	if err != nil {
		return
	}
	go p.listenSystemdUpdates()
}
