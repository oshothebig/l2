Copyright [2016] [SnapRoute Inc]

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

	 Unless required by applicable law or agreed to in writing, software
	 distributed under the License is distributed on an "AS IS" BASIS,
	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	 See the License for the specific language governing permissions and
	 limitations under the License.
package server

import (
	"fmt"
	"l2/lldp/utils"
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
	debug.Logger.Info("DB connection is established")
	return err
}

func (svr *LLDPServer) CloseDB() {
	debug.Logger.Info("Closed lldp db")
	svr.lldpDbHdl.Disconnect()
}

func (svr *LLDPServer) ReadDB() error {
	debug.Logger.Info("Reading from Database")
	debug.Logger.Info("Done reading from DB")
	return nil
}
