// system
package lacp

import ()

// 6.4.5 Variables associated with the System
type LacpSystem struct {
	// System Priority
	actor_system_priority uint16
	// MAC address component of the System Id
	actor_system [6]uint8
}

func (s *LacpSystem) LacpSystemActorSystemIdSet(actor_system [6]uint8) {
	s.actor_system[0] = actor_system[0]
	s.actor_system[1] = actor_system[1]
	s.actor_system[2] = actor_system[2]
	s.actor_system[3] = actor_system[3]
	s.actor_system[4] = actor_system[4]
	s.actor_system[5] = actor_system[5]
}

func (s *LacpSystem) LacpSystemActorSystemPrioritySet(actor_system_priority uint16) {
	s.actor_system_priority = actor_system_priority
}
