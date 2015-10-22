// aggregator
package lacp

import (
	"time"
)

// Indicates on a port what state
// the aggSelected is in
const (
	LacpAggSelected = iota + 1
	LacpAggStandby
	LacpAggUnSelected
)

type LacpAggrigatorStats struct {
	// does not include lacp or marker pdu
	octetsTx int
	octetsRx int
	framesTx int
	framesRx int
}

// 802.1ax-2014 Section 6.4.6 Variables associated with each Aggregator
// Section 7.3.1.1

type LaAggregator struct {
	// 802.1ax Section 7.3.1.1
	aggId               int
	aggDescription      string   // 255 max chars
	aggName             string   // 255 max chars
	actorSystemId       [6]uint8 // mac address
	actorSystemPriority int
	// aggregation capability
	// TRUE - port attached to this aggregetor is not capable
	//        of aggregation to any other aggregator
	// FALSE - port attached to this aggregator is able of
	//         aggregation to any other aggregator
	aggOrIndividual bool
	actorAdminKey   int
	actorOperKey    int
	aggMacAddr      [6]uint
	// remote system
	partnerSystemId       [6]uint8
	partnerSystemPriority int
	partnerOperKey        int

	// up/down
	adminState int
	operState  int

	// sum of data rate of each link in aggregation (read-only)
	dataRate int

	timeOfLastOperChange time.Time

	receive_state  bool
	transmit_state bool

	ready bool

	// Port number from LaAggPort
	portNumList []int
}

// stores are the actual LAGs
var aggMap = make(map[int]*LaAggregator)

// TODO add more defaults
func NewLaAggregator(aggId int) *LaAggregator {
	agg := &LaAggregator{
		aggId: aggId,
		ready: true,
	}

	// add agg to map
	aggMap[aggId] = agg

	return agg
}

// LacpMuxCheckSelectionLogic will be called after the
// wait while timer has expired.  If this is the last
// port to have its wait while timer expire then
// will transition the mux state from waiting to
// attached
func (agg *LaAggregator) LacpMuxCheckSelectionLogic(p *LaAggPort) {

	readyChan := make(chan bool)
	// lets do a this work in parrallel
	for _, pId := range agg.portNumList {

		go func(id int) {
			var port *LaAggPort
			if LaFindPortById(id, port) {
				readyChan <- port.readyN
			}
		}(pId)
	}
	close(readyChan)

	// lets set the agg.ready flag to true
	// until we know that at least one port
	// is not ready
	agg.ready = true
	for range agg.portNumList {
		select {
		case readyN := <-readyChan:
			if !readyN {
				agg.ready = false
			}
		}
	}

	// if agg is ready then lets attach the
	// ports which are not already attached
	if agg.ready {
		// lets do this work in parrallel
		for _, pId := range agg.portNumList {
			go func(id int) {
				var port *LaAggPort
				if LaFindPortById(id, port) &&
					port.readyN &&
					port.aggAttached != nil {
					// trigger event to mux
					port.muxMachineFsm.MuxmEvents <- LacpMuxmEventSelectedEqualSelectedAndReady
				}
			}(pId)
		}
	}
}

func LaFindAggById(aggId int, agg *LaAggregator) bool {
	agg, ok := aggMap[aggId]
	return ok
}
