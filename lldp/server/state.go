package server

import (
	"fmt"
	"l2/lldp/config"
	"l2/lldp/utils"
	"strconv"
)

/*  helper function to convert Mandatory TLV's (chassisID, portID, TTL) from byte
 *  format to string
 */
func (svr *LLDPServer) PopulateMandatoryTLV(ifIndex int32, entry *config.IntfState) bool {
	gblInfo, exists := svr.lldpGblInfo[ifIndex]
	if !exists {
		debug.Logger.Err(fmt.Sprintln("Entry not found for", ifIndex))
		return exists
	}
	entry.LocalPort = gblInfo.Port.Name
	if gblInfo.RxInfo.RxFrame != nil {
		entry.PeerMac = gblInfo.GetChassisIdInfo()
		entry.Port = gblInfo.GetPortIdInfo()
		entry.HoldTime = strconv.Itoa(int(gblInfo.RxInfo.RxFrame.TTL))
	}
	entry.IfIndex = gblInfo.Port.IfIndex
	entry.Enable = true
	return exists
}

/*  Server get bulk for lldp up intf state's
 */
func (svr *LLDPServer) GetIntfStates(idx, cnt int) (int, int, []config.IntfState) {
	var nextIdx int
	var count int

	if svr.lldpIntfStateSlice == nil {
		debug.Logger.Info("No neighbor learned")
		return 0, 0, nil
	}

	length := len(svr.lldpUpIntfStateSlice)
	result := make([]config.IntfState, cnt)

	var i, j int

	for i, j = 0, idx; i < cnt && j < length; j++ {
		key := svr.lldpUpIntfStateSlice[j]
		succes := svr.PopulateMandatoryTLV(key, &result[i])
		if !succes {
			result = nil
			return 0, 0, nil
		}
		i++
	}

	if j == length {
		nextIdx = 0
	}
	count = i
	return nextIdx, count, result
}
