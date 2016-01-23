// def.go
package stp

import (
	"strconv"
	"strings"
	"utils/fsm"
)

const (
	MigrateTimeDefault        = 3
	BridgeHelloTimeDefault    = 2
	BridgeMaxAgeDefault       = 20
	BridgeForwardDelayDefault = 15
	TransmitHoldCountDefault  = 6
)

type MachineEvent struct {
	e            fsm.Event
	src          string
	responseChan chan string
}

type StpStateEvent struct {
	// current State
	s fsm.State
	// previous State
	ps fsm.State
	// current event
	e fsm.Event
	// previous event
	pe fsm.Event

	// event src
	esrc        string
	owner       string
	strStateMap map[fsm.State]string
	logEna      bool
	logger      func(string)
}

func SendResponse(msg string, responseChan chan string) {
	responseChan <- msg
}

func (se *StpStateEvent) LoggerSet(log func(string))                 { se.logger = log }
func (se *StpStateEvent) EnableLogging(ena bool)                     { se.logEna = ena }
func (se *StpStateEvent) IsLoggerEna() bool                          { return se.logEna }
func (se *StpStateEvent) StateStrMapSet(strMap map[fsm.State]string) { se.strStateMap = strMap }
func (se *StpStateEvent) PreviousState() fsm.State                   { return se.ps }
func (se *StpStateEvent) CurrentState() fsm.State                    { return se.s }
func (se *StpStateEvent) PreviousEvent() fsm.Event                   { return se.pe }
func (se *StpStateEvent) CurrentEvent() fsm.Event                    { return se.e }
func (se *StpStateEvent) SetEvent(es string, e fsm.Event) {
	se.esrc = es
	se.pe = se.e
	se.e = e
}
func (se *StpStateEvent) SetState(s fsm.State) {
	se.ps = se.s
	se.s = s
	if se.IsLoggerEna() && se.ps != se.s {
		se.logger((strings.Join([]string{"Src", se.esrc, "OldState", se.strStateMap[se.ps], "Evt", strconv.Itoa(int(se.e)), "NewState", se.strStateMap[s]}, ":")))
	}
}
