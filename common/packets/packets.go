package packets

import (
	"bytes"
	"encoding/gob"
	"errors"
	"net"
)

/*
	The first byte in the packet should be packet types from load-balancer/common/constants.
	The following structs follow from the 2nd byte in the packet.
*/

// Types
type PacketType uint8

const (
	PacketTypeBeg PacketType = iota
	ConnectionRequest
	ConnectionResponse
	ConnectionAck
	InfoRequest
	InfoResponse
	PacketTypeEnd
)

func (pt PacketType) String() string {
	switch pt {
	case ConnectionRequest:
		return "ConnectionRequest"
	case ConnectionResponse:
		return "ConnectionResponse"
	case ConnectionAck:
		return "ConnectionAck"
	case InfoRequest:
		return "InfoRequest"
	case InfoResponse:
		return "InfoResponse"
	default:
		return ""
	}
}

type BroadcastConnectRequest struct {
	Source net.IP
	Port   uint16
}

type BroadcastConnectResponse struct {
	Ack         bool
	IP          net.IP
	Port        uint16
	InfoReqPort uint16
	ReqSendPort uint16
}

type InfoRequestPacket struct {
	Port uint16
}

type InfoResponsePacket struct {
	Load float64
}

func GetPacketType(buf []byte) (PacketType, error) {
	packetType := PacketType(buf[0])
	if packetType <= PacketTypeBeg || packetType >= PacketTypeEnd {
		return packetType, errors.New("Invalid PacketType")
	}
	return packetType, nil
}

func EncodePacket(packet interface{}, packetType PacketType) ([]byte, error) {

	switch t := packet.(type) {
	case BroadcastConnectRequest:
	case BroadcastConnectResponse:
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
