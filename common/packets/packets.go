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
type PacketType int8

const (
	ConnectionRequest PacketType = iota
	ConnectionResponse
)

type BroadcastConnectRequest struct {
	Source net.IP
	Port   int16
}

type BroadcastConnectResponse struct {
	Ack bool
	IP  net.IP
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

	network := bytes.NewBuffer(buf)
	dec := gob.NewDecoder(network)

	err := dec.Decode(packet)
	return err
}
