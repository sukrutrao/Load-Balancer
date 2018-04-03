package packets

import (
	"net"
)

/*
	The first byte in the packet should be packet types from load-balancer/common/constants.
	The following structs follow from the 2nd byte in the packet.
*/

type BroadcastConnectRequest struct {
	Source net.IP
	Port   int16
}

type BroadcastConnectResponse struct {
	Ack bool
	IP  net.IP
}
