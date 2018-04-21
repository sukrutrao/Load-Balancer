package constants

import (
	"net"
	"time"
)

// Addressess
var (
	BroadcastReceiveAddress net.IP = []byte{0, 0, 0, 0}
)

// Ports
const (
	MasterBroadcastPort uint16 = 3000
)

type PacketType int8
type Status int8

// Types
const (
	ConnectionRequest PacketType = iota
	ConnectionResponse
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

// Timeouts
const (
	WaitForSlaveTimeout       time.Duration = 5 * time.Second
	WaitForReqTimeout                       = 5 * time.Second
	LoadRequestInterval                     = 5 * time.Second
	GarbageCollectionInterval               = 5 * time.Second
)

// Others
const (
	NumBurstAcks    int = 10
	MaxConnectRetry     = 6

	ConnectRetryBackoffBaseTime time.Duration = 2 * time.Second
)
