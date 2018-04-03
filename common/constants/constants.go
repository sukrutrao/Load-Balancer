package constants

import (
	"net"
)

// Addressess
var (
	BroadcastReceiveAddress net.IP = []byte{0, 0, 0, 0}
)

// Ports
const (
	MasterBroadcastPort int16 = 3000
	SlaveBroadcastPort  int16 = 3001
	InfoSenderPort      int16 = 3002
	InfoReceiverPort    int16 = 3003
)

// Types
const (
	ConnectionRequest int8 = iota
	ConnectionResponse
)
