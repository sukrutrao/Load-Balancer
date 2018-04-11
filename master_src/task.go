package master

import (
	"net"

	"github.com/GoodDeeds/load-balancer/common/constants"
	"github.com/GoodDeeds/load-balancer/common/packets"
)

func (m *Master) offerTask(s *Slave, t *Task, conn *net.TCPConn) {
	p := packets.TaskOfferRequest{t.TaskId, t.Load}
	enc := packets.EncodePacket(p, constants.TaskOfferRequest)
	_, err := conn.Write(enc)

	// response code pending
}

func (m *Master) assignTask(s *Slave, t *Task, conn *net.TCPConn) {
	p := packets.TaskRequest{t.TaskId, t.Task, t.Load}
	enc := packets.EncodePacket{p, constants.TaskRequest}
	_, err := conn.Write(enc)

	// response code pending
}
