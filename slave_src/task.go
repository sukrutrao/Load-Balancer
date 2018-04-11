package slave

import (
	"net"

	"github.com/GoodDeeds/load-balancer/common/constants"
	"github.com/GoodDeeds/load-balancer/common/packets"
)

func (s *Slave) getTask(t *Task, conn *net.TCPConn) {
	message := []byte{}
	p := TaskRequest{}
	_, err := conn.Read(message)
	err := packets.DecodePacket(message, p)
	if currentLoad+p.Load > maxLoad {
		// reject
	} else {
		task := Task{p.TaskId, p.Task, p.Load, constants.Incomplete}
		go handleTask(task)
		// accept
	}
}

func (s *Slave) respondTaskStatus(t *Task, conn *net.TCPConn) {
	message := []byte{}
	p := TaskStatusRequest{}
	_, err := conn.Read(message)
	err := packets.DecodePacket(message, p)
	status := getStatus(p.TaskId)
	response := TaskStatusResponse{p.TaskId, status}
	enc, err := packets.EncodePacket(response)
	conn.Write(enc)
}

func (s *Slave) sendTaskResult(t *Task, conn *net.TCPConn) {
	if t.TaskStatus != constants.Complete {
		return
	}
	taskResult := *t.Result
	p := TaskResultResponse{t.TaskId, taskResult}
	enc, err := packets.EncodePacket(p)
	conn.Write(enc)
}

func (s *Slave) getStatus(taskId int) (status constants.Status) {
	task := s.Tasks[taskId]

}
