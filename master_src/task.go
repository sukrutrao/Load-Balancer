package master

import (
	"net"

	"github.com/GoodDeeds/load-balancer/common/constants"
	"github.com/GoodDeeds/load-balancer/common/packets"
)

func (m *Master) assignTask(s *Slave, t *Task, conn *net.TCPConn) {
	p := packets.TaskRequest{t.TaskId, t.Task, t.Load}
	enc, err := packets.EncodePacket{p, constants.TaskRequest}
	_, err := conn.Write(enc)

	// response code pending
}

// not sure if this is needed
func (m *Master) requestTaskStatus(s *Slave, t *Task, conn *net.TCPConn) {
	p := packets.TaskStatusRequest{t.TaskId}
	enc, err := packets.EncodePacket{p, constants.TaskStatusRequest}
	_, err := conn.Write(enc)

	// response code pending
}
