// debugEventLog this code is meant to serialize the logging states
package lacp

import (
	"fmt"
	"strings"
	"time"
)

type LacpDebug struct {
	LacpLogChan chan string
}

// NewLacpRxMachine will create a new instance of the LacpRxMachine
func NewLacpDebug() *LacpDebug {
	lacpdebug := &LacpDebug{
		LacpLogChan: make(chan string)}

	return lacpdebug
}

func (l *LacpDebug) Stop() {
	close(l.LacpLogChan)
}

func (p *LaAggPort) LacpDebugEventLogMain() {

	p.LacpDebug = NewLacpDebug()

	go func(port *LaAggPort) {

		for {
			select {

			case msg, logEvent := <-port.LacpDebug.LacpLogChan:
				if logEvent {
					fmt.Println(strings.Join([]string{time.Now().Format("Mon Jan 2 15:04:05.99999999 -0700 MST 2006"), p.intfNum, msg}, "-"))
				} else {
					return
				}
			}
		}
	}(p)
}

func (txm *LacpTxMachine) LacpTxmLog(msg string) {
	if txm.Machine.Curr.IsLoggerEna() {
		txm.log <- strings.Join([]string{"TXM", msg}, ":")
	}
}

func (cdm *LacpCdMachine) LacpCdmLog(msg string) {
	if cdm.Machine.Curr.IsLoggerEna() {
		cdm.log <- strings.Join([]string{"CDM", msg}, ":")
	}
}

func (ptxm *LacpPtxMachine) LacpPtxmLog(msg string) {
	if ptxm.Machine.Curr.IsLoggerEna() {
		ptxm.log <- strings.Join([]string{"PTXM", msg}, ":")
	}
}

func (rxm *LacpRxMachine) LacpRxmLog(msg string) {
	if rxm.Machine.Curr.IsLoggerEna() {
		rxm.log <- strings.Join([]string{"RXM", msg}, ":")
	}
}

func (muxm *LacpMuxMachine) LacpMuxmLog(msg string) {
	if muxm.Machine.Curr.IsLoggerEna() {
		muxm.log <- strings.Join([]string{"MUXM", msg}, ":")
	}
}
