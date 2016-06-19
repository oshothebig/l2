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

package server

import (
	"fmt"
	"l2/lldp/config"
	"l2/lldp/utils"
	"models/objects"
	"utils/dbutils"
)

func (svr *LLDPServer) InitDB() error {
	var err error
	debug.Logger.Info("Initializing DB")
	svr.lldpDbHdl = dbutils.NewDBUtil(debug.Logger)
	err = svr.lldpDbHdl.Connect()
	if err != nil {
		debug.Logger.Err(fmt.Sprintln("Failed to Create DB Handle", err))
		return err
	}
	debug.Logger.Info(fmt.Sprintln("DB connection is established, error:", err))
	return nil
}

func (svr *LLDPServer) CloseDB() {
	debug.Logger.Info("Closed lldp db")
	svr.lldpDbHdl.Disconnect()
}

func (svr *LLDPServer) readLLDPIntfConfig() {
	debug.Logger.Info("Reading LLDPIntf from db")
	var dbObj objects.LLDPIntf
	objList, err := svr.lldpDbHdl.GetAllObjFromDb(dbObj)
	if err != nil {
		debug.Logger.Err(fmt.Sprintln("DB querry faile for LLDPIntf Config", err))
		//return nil
	}
	// READ DB is always called before calling asicd get ports..
	debug.Logger.Info(fmt.Sprintln("Objects from db are", objList))
	for _, obj := range objList {
		dbEntry := obj.(objects.LLDPIntf)
		gblInfo, _ := svr.lldpGblInfo[dbEntry.IfIndex]
		debug.Logger.Info(fmt.Sprintln("IfIndex", dbEntry.IfIndex, "is set to", dbEntry.Enable))
		switch dbEntry.Enable {
		case true:
			gblInfo.Enable()
		case false:
			gblInfo.Disable()
		}
		svr.lldpGblInfo[dbEntry.IfIndex] = gblInfo
	}
	debug.Logger.Info("Done with LLDPIntf")
}

func (svr *LLDPServer) readLLDPGlobalConfig() {
	debug.Logger.Info("Reading LLDPGlobal from db")
	var dbObj objects.LLDPGlobal
	objList, err := svr.lldpDbHdl.GetAllObjFromDb(dbObj)
	if err != nil {
		debug.Logger.Err(fmt.Sprintln("DB querry faile for LLDPGlobal Config", err))
		//return nil
	}
	// READ DB is always called before calling asicd get ports..
	debug.Logger.Info(fmt.Sprintln("Objects from db are", objList))
	for _, obj := range objList {
		dbEntry := obj.(objects.LLDPGlobal)
		if svr.Global == nil {
			svr.Global = &config.Global{}
		}
		svr.Global.Vrf = dbEntry.Vrf
		svr.Global.Enable = dbEntry.Enable
	}
	debug.Logger.Info("Done with LLDPGlobal")
}

func (svr *LLDPServer) ReadDB() error {
	if svr.lldpDbHdl == nil {
		debug.Logger.Info("Invalid db HDL")
		return nil
	}
	svr.readLLDPGlobalConfig()
	svr.readLLDPIntfConfig()
	return nil
}
