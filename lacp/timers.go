// timers
package lacp

import (
	"time"
)

// WaitWhileTimerStart
// Start the timer
func (p *LaAggPort) WaitWhileTimerStart() {
	p.waitWhileTimer.Reset(p.waitWhileTimerTimeout)
}

// WaitWhileTimerStop
// Stop the timer, which should only happen
// on creation as well as when the lacp mode is "on"
func (p *LaAggPort) WaitWhileTimerStop() {
	p.waitWhileTimer.Stop()
}

func (p *LaAggPort) WaitWhileTimerTimeoutSet(timeout time.Duration) {
	p.waitWhileTimerTimeout = timeout
}

func (p *LaAggPort) CurrentWhileTimerStart() {
	p.currentWhileTimer.Reset(p.currentWhileTimerTimeout)
}

func (p *LaAggPort) CurrentWhileTimerStop() {
	p.currentWhileTimer.Stop()
}

func (p *LaAggPort) CurrentWhileTimerTimeoutSet(timeout time.Duration) {
	p.currentWhileTimerTimeout = timeout
}

func (p *LaAggPort) PeriodicTimerStart() {
	p.periodicTxTimer.Reset(p.periodicTxTimerInterval)
}

func (p *LaAggPort) PeriodicTimerStop() {
	p.periodicTxTimer.Stop()
}

func (p *LaAggPort) PeriodicTimerIntervalSet(interval time.Duration) {
	p.periodicTxTimerInterval = interval
}

// TxDelayTimerStart used by Tx Machine as described in
// 802.1ax-2014 Section 6.4.17
func (p *LaAggPort) TxDelayTimerStart() {
	p.txDelayTimer.Reset(LacpFastPeriodicTime)
}

// TxDelayTimerStop to stop the Delay timer
// in case a port is deleted or initialized
func (p *LaAggPort) TxDelayTimerStop() {
	p.txDelayTimer.Stop()
}
