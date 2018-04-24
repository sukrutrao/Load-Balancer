package slave

import (
	/*	"net"*/
	"time"

	// "github.com/GoodDeeds/load-balancer/common/constants"
	"github.com/GoodDeeds/load-balancer/common/logger"
	"github.com/GoodDeeds/load-balancer/common/packets"
)

func (s *Slave) getTask(t *SlaveTask, p packets.TaskRequestPacket) packets.TaskRequestResponsePacket {
	response := packets.TaskRequestResponsePacket{TaskId: p.TaskId}
	if s.currentLoad+p.Load > s.maxLoad {
		response.Accept = false
		s.Logger.Warning(logger.FormatLogMessage("msg", "Slave refused task due to load limit", "Task ID", string(p.TaskId)))
		return response
	} else {
		t := SlaveTask{TaskId: p.TaskId, Task: p.Task, Load: p.Load, TaskStatus: packets.Incomplete}
		go s.handleTask(&t)
		response.Accept = true
		s.Logger.Info(logger.FormatLogMessage("msg", "Slave accepted task", "Task ID", string(p.TaskId)))
		return response
	}
}

func (s *Slave) respondTaskStatusPacket(p packets.TaskStatusRequestPacket) packets.TaskStatusResponsePacket {
	status := s.getStatus(p.TaskId)
	response := packets.TaskStatusResponsePacket{p.TaskId, status}
	return response
}

func (s *Slave) sendTaskResult(t *SlaveTask) packets.TaskResultResponsePacket {
	response := packets.TaskResultResponsePacket{TaskId: t.TaskId}
	if t.TaskStatus != packets.Complete {
		s.Logger.Warning("msg", "Task is not yet complete", "Task ID", string(t.TaskId))
		response.TaskStatus = packets.Incomplete
		return response
	}
	response.Result = t.Result
	response.TaskStatus = packets.Complete
	s.Logger.Warning("msg", "Task is complete", "Task ID", string(t.TaskId))
	return response
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
	s.Logger.Info("msg", "Handling Task", "Task ID", string(t.TaskId))
	time.Sleep(10000 * time.Millisecond)
	t.Result = "Complete"
	s.Logger.Info("msg", "Done Task", "Task ID", string(t.TaskId))
	go s.sendTaskResult(t)
}
