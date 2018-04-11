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

type PacketType int8
type Status int8

// Types
const (
	ConnectionRequest PacketType = iota
	ConnectionResponse
	TaskOfferRequest
	TaskOfferResponse
	TaskRequest
	TaskRequestResponse
	TaskResultResponse
	TaskStatusRequest
	TaskStatusResponse
)

// Status codes
// TODO can we extend this for responses on whether to accept a task?
// would give it finer granularity
// specify an estimate when the slave might be free, so the master can query again?
const (
	Complete Status = iota
	Incomplete
	Invalid
)
