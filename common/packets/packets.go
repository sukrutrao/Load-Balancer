package packets

import (
	"bytes"
	"encoding/gob"
	"errors"
	"net"
	"time"
)

/*
	The first byte in the packet should be packet types from load-balancer/common/constants.
	The following structs follow from the 2nd byte in the packet.
*/

// Types
type PacketType uint8
type TaskType uint8
type Status int8

const (
	PacketTypeBeg PacketType = iota
	ConnectionRequest
	ConnectionResponse
	ConnectionAck
	LoadRequest
	LoadResponse
	MonitorConnectionRequest
	MonitorConnectionResponse
	MonitorConnectionAck
	MonitorRequest
	MonitorResponse
	TaskRequest
	TaskRequestResponse
	TaskResultResponse
	TaskStatusRequest
	TaskStatusResponse
	PacketTypeEnd
)

// Task Type IDs
const (
	FibonacciTaskType TaskType = iota
	CountPrimesTaskType
)

var LoadFunctions map[TaskType]func(int) uint64 = map[TaskType]func(int) uint64{
	FibonacciTaskType: func(n int) uint64 {
		return uint64(n)
	},
	CountPrimesTaskType: func(n int) uint64 {
		return uint64(n) * uint64(n)
	},
}

// Status codes
// TODO can we extend this for responses on whether to accept a task?
// would give it finer granularity
// specify an estimate when the slave might be free, so the master can query again?
const (
	Complete Status = iota
	Incomplete
	Invalid
	Unassigned
)

type PacketTransmit struct {
	Packet     interface{}
	PacketType PacketType
}

func (pt PacketType) String() string {
	switch pt {
	case ConnectionRequest:
		return "ConnectionRequest"
	case ConnectionResponse:
		return "ConnectionResponse"
	case ConnectionAck:
		return "ConnectionAck"
	case LoadRequest:
		return "InfoRequest"
	case LoadResponse:
		return "InfoResponse"
	case MonitorConnectionRequest:
		return "MonitorConnectionRequest"
	case MonitorConnectionResponse:
		return "MonitorConnectionResponse"
	case MonitorConnectionAck:
		return "MonitorConnectionAck"
	case MonitorRequest:
		return "MonitorRequest"
	case MonitorResponse:
		return "MonitorResponse"
	case TaskRequest:
		return "AssignTaskToSlave"
	case TaskRequestResponse:
		return "SlaveReplyToTaskAssignment"
	case TaskResultResponse:
		return "TaskResultFromSlave"
	case TaskStatusRequest:
		return "AskSlaveForTaskStatus"
	case TaskStatusResponse:
		return "SlaveReplyTaskStatus"
	default:
		return ""
	}
}

type BroadcastConnectRequest struct {
	Source net.IP
	Port   uint16
}

type BroadcastConnectResponse struct {
	Ack bool
	IP  net.IP

	// Used only by Slave.
	Port        uint16
	LoadReqPort uint16
	ReqSendPort uint16
	ReqRecvPort uint16
}

type LoadRequestPacket struct {
	Port uint16
}

type LoadResponsePacket struct {
	Timestamp time.Time
	Load      uint64
	MaxLoad   uint64
}

type MonitorRequestPacket struct {
}

type MonitorResponsePacket struct {
	SlaveIPs []string
}

func GetPacketType(buf []byte) (PacketType, error) {
	packetType := PacketType(buf[0])
	if packetType <= PacketTypeBeg || packetType >= PacketTypeEnd {
		return packetType, errors.New("Invalid PacketType")
	}
	return packetType, nil
}

func EncodePacket(packet interface{}, packetType PacketType) ([]byte, error) {

	if packetType <= PacketTypeBeg || packetType >= PacketTypeEnd {
		return nil, errors.New("Invalid packet type")
	}
	switch t := packet.(type) {
	case BroadcastConnectRequest:
	case BroadcastConnectResponse:
	case LoadRequestPacket:
	case LoadResponsePacket:
	case MonitorRequestPacket:
	case MonitorResponsePacket:
	case TaskRequestPacket:
	case TaskRequestResponsePacket:
	case TaskResultResponsePacket:
	case TaskStatusRequestPacket:
	case TaskStatusResponsePacket:
	default:
		_ = t
		return nil, errors.New("Invalid packet")
	}

	var network bytes.Buffer
	network.WriteByte(byte(packetType))
	enc := gob.NewEncoder(&network)
	err := enc.Encode(packet)
	if err != nil {
		return nil, err
	}
	return network.Bytes(), nil
}

func DecodePacket(buf []byte, packet interface{}) error {

	network := bytes.NewBuffer(buf[1:])
	dec := gob.NewDecoder(network)

	err := dec.Decode(packet)
	return err
}

func CreatePacketTransmit(packet interface{}, packetType PacketType) PacketTransmit {
	pt := PacketTransmit{packet, packetType}
	return pt
}

type TaskRequestPacket struct {
	TaskId int
	Task   TaskPacket // TODO - change this
	Load   uint64
}

type TaskRequestResponsePacket struct {
	TaskId int
	Accept bool
}

type TaskResultResponsePacket struct {
	TaskId     int
	Result     TaskPacket // TODO - change this
	TaskStatus Status
}

type TaskStatusRequestPacket struct {
	TaskId int
}

type TaskStatusResponsePacket struct {
	TaskId     int
	TaskStatus Status // from status constants in constants.go
}

// TODO - this should be in slave.go
type TaskResult struct {
	Result string
}

type TaskPacket struct {
	TaskTypeID TaskType
	N          int
	Result     uint64
	IntResult  int
	Close      chan struct{}
}

func (t *TaskPacket) Description() string {
	switch t.TaskTypeID {
	case FibonacciTaskType:
		return "Task to find Nth fibonacci number"
	case CountPrimesTaskType:
		return "Task to find the number of primes <= N"
	default:
		return "Unknown task type"
	}
}
