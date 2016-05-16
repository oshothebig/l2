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
// markerpdu
package lacp

import ()

type LaMarkerPdu struct {
	subType                   uint8 // 2
	version                   uint8
	tlv_type_marker_info      uint8
	marker_information_length uint8 // 16
	requestor_port            uint16
	requestor_System          [6]uint8
	requestor_transaction_id  [4]uint8
	pad                       uint16
	tlv_type_terminator       uint8 // terminator 0x00
	terminator_length         uint8
	reserved                  [90]uint8
	// FCS 4 bytes
}
