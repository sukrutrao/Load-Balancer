package slave

import (
	/*	"net"*/
	"strconv"
	"time"

	// "github.com/GoodDeeds/load-balancer/common/constants"
	"github.com/GoodDeeds/load-balancer/common/logger"
	"github.com/GoodDeeds/load-balancer/common/packets"
)

func (s *Slave) getTask(p packets.TaskRequestPacket) {
	response := packets.TaskRequestResponsePacket{TaskId: p.TaskId}
	if s.currentLoad+p.Load > s.maxLoad {
		response.Accept = false
		s.Logger.Warning(logger.FormatLogMessage("msg", "Slave refused task due to load limit", "Task ID", strconv.Itoa(int(p.TaskId))))
	} else {
		t := SlaveTask{TaskId: p.TaskId, Task: p.Task, Load: p.Load, TaskStatus: packets.Incomplete}
		go s.handleTask(&t)
		response.Accept = true
		s.Logger.Info(logger.FormatLogMessage("msg", "Slave accepted task", "Task ID", strconv.Itoa(int(p.TaskId))))
	}
	pt := packets.CreatePacketTransmit(response, packets.TaskRequestResponse)
	s.sendChan <- pt
}

func (s *Slave) respondTaskStatusPacket(p packets.TaskStatusRequestPacket) {
	status := s.getStatus(p.TaskId)
	response := packets.TaskStatusResponsePacket{p.TaskId, status}
	pt := packets.CreatePacketTransmit(response, packets.TaskStatusResponse)
	s.sendChan <- pt
}

func (s *Slave) sendTaskResult(t *SlaveTask) {
	response := packets.TaskResultResponsePacket{TaskId: t.TaskId}
	if t.TaskStatus != packets.Complete {
		s.Logger.Warning(logger.FormatLogMessage("msg", "Task is not yet complete", "Task ID", strconv.Itoa(int(t.TaskId))))
		response.TaskStatus = packets.Incomplete
	} else {
		response.Result = t.Result
		response.TaskStatus = packets.Complete
		s.Logger.Info(logger.FormatLogMessage("msg", "Task is complete", "Task ID", strconv.Itoa(int(t.TaskId))))
	}
	pt := packets.CreatePacketTransmit(response, packets.TaskResultResponse)
	// s.Logger.Info(logger.FormatLogMessage("msg", "Sending result to channel"))
	s.sendChan <- pt
}

func (s *Slave) getStatus(taskId int) (status packets.Status) {
	task, ok := s.tasks[taskId]
	if !ok {
		return packets.Invalid
	}
	return task.TaskStatus
}

func (s *Slave) handleTask(t *SlaveTask) {
	s.currentLoad += t.Load
	s.Logger.Info(logger.FormatLogMessage("msg", "Handling Task", "Task ID", strconv.Itoa(int(t.TaskId))))
	time.Sleep(10000 * time.Millisecond)
	t.Result = "Complete"
	t.TaskStatus = packets.Complete
	s.Logger.Info(logger.FormatLogMessage("msg", "Done Task", "Task ID", strconv.Itoa(int(t.TaskId))))
	go s.sendTaskResult(t)
}
