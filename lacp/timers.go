// timers
package lacp

import (
	"time"
)

// WaitWhileTimerStart
// Start the timer
func (rxm *LacpRxMachine) WaitWhileTimerStart() {
	if rxm.waitWhileTimer == nil {
		rxm.waitWhileTimer = time.NewTimer(rxm.waitWhileTimerTimeout)
	} else {
		rxm.waitWhileTimer.Reset(rxm.waitWhileTimerTimeout)
	}
}

// WaitWhileTimerStop
// Stop the timer, which should only happen
// on creation as well as when the lacp mode is "on"
func (rxm *LacpRxMachine) WaitWhileTimerStop() {
	if rxm.waitWhileTimer != nil {
		rxm.waitWhileTimer.Stop()
	}
}

func (rxm *LacpRxMachine) WaitWhileTimerTimeoutSet(timeout time.Duration) {
	rxm.waitWhileTimerTimeout = timeout
}

func (rxm *LacpRxMachine) CurrentWhileTimerStart() {
	if rxm.currentWhileTimer == nil {
		rxm.currentWhileTimer = time.NewTimer(rxm.currentWhileTimerTimeout)
	} else {
		rxm.currentWhileTimer.Reset(rxm.currentWhileTimerTimeout)
	}
}

func (rxm *LacpRxMachine) CurrentWhileTimerStop() {
	if rxm.currentWhileTimer != nil {
		rxm.currentWhileTimer.Stop()
	}
}

func (rxm *LacpRxMachine) CurrentWhileTimerTimeoutSet(timeout time.Duration) {
	rxm.currentWhileTimerTimeout = timeout
}

func (ptxm *LacpPtxMachine) PeriodicTimerStart() {
	if ptxm.periodicTxTimer == nil {
		ptxm.periodicTxTimer = time.NewTimer(ptxm.PeriodicTxTimerInterval)
	} else {
		ptxm.periodicTxTimer.Reset(ptxm.PeriodicTxTimerInterval)
	}
}

func (ptxm *LacpPtxMachine) PeriodicTimerStop() {
	if ptxm.periodicTxTimer != nil {
		ptxm.periodicTxTimer.Stop()
	}
}

func (ptxm *LacpPtxMachine) PeriodicTimerIntervalSet(interval time.Duration) {
	ptxm.PeriodicTxTimerInterval = interval
}

func (p *LaAggPort) ChurnDetectionTimerStart() {
	p.actorChurnTimer.Reset(p.actorChurnTimerInterval)
}

func (p *LaAggPort) ChurnDetectionTimerStop() {
	p.actorChurnTimer.Stop()
}

func (p *LaAggPort) ChurnDetectionTimerIntervalSet(interval time.Duration) {
	p.actorChurnTimerInterval = interval
}

// TxGuardTimerStart used by Tx Machine as described in
// 802.1ax-2014 Section 6.4.17 in order to not transmit
// more than 3 packets in this interval
func (txm *LacpTxMachine) TxGuardTimerStart() {
	if txm.txGuardTimer == nil {
		txm.txGuardTimer = time.AfterFunc(LacpFastPeriodicTime, txm.LacpTxGuardGeneration)
	} else {
		txm.txGuardTimer.Reset(LacpFastPeriodicTime)
	}
}

// TxDelayTimerStop to stop the Delay timer
// in case a port is deleted or initialized
func (txm *LacpTxMachine) TxGuardTimerStop() {
	txm.txGuardTimer.Stop()
}
