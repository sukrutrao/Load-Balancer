package slave

import (
	"strconv"
	"sync/atomic"

	// "github.com/GoodDeeds/load-balancer/common/constants"
	"github.com/GoodDeeds/load-balancer/common/logger"
	"github.com/GoodDeeds/load-balancer/common/packets"
)

func (s *Slave) getTask(p packets.TaskRequestPacket) {
	response := packets.TaskRequestResponsePacket{TaskId: p.TaskId}
	atomic.AddUint32(&s.metric.TasksRequested, 1)
	if s.currentLoad+p.Load > s.maxLoad {
		response.Accept = false
		s.Logger.Warning(logger.FormatLogMessage("msg", "Slave refused task due to load limit", "Task ID", strconv.Itoa(int(p.TaskId))))
	} else {
		t := SlaveTask{TaskId: p.TaskId, Task: p.Task, Load: p.Load, TaskStatus: packets.Incomplete}
		go s.handleTask(&t)
		response.Accept = true
		atomic.AddUint64(&s.currentLoad, packets.LoadFunctions[p.Task.TaskTypeID](p.Task.N))
		atomic.AddUint32(&s.metric.TasksAccepted, 1)
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
		response.Result = t.Task
		response.TaskStatus = packets.Complete
		s.Logger.Info(logger.FormatLogMessage("msg", "Task is complete", "Task ID", strconv.Itoa(int(t.TaskId))))
		s.displayResult(&t.Task, t.TaskId)
	}
	atomic.AddUint64(&s.currentLoad, ^(packets.LoadFunctions[t.Task.TaskTypeID](t.Task.N) - 1))
	atomic.AddUint32(&s.metric.TasksCompleted, 1)
	pt := packets.CreatePacketTransmit(response, packets.TaskResultResponse)
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
	s.Logger.Info(logger.FormatLogMessage("msg", "Handling Task", "Task ID", strconv.Itoa(int(t.TaskId))))
	s.runTask(&t.Task)
	//	time.Sleep(2 * time.Second)
	t.TaskStatus = packets.Complete
	s.Logger.Info(logger.FormatLogMessage("msg", "Done Task", "Task ID", strconv.Itoa(int(t.TaskId))))
	s.sendTaskResult(t)
}

func (s *Slave) displayResult(t *packets.TaskPacket, taskId int) {
	switch t.TaskTypeID {
	case packets.FibonacciTaskType:
		s.Logger.Info(logger.FormatLogMessage("Task ID", strconv.Itoa(taskId), "Result", strconv.Itoa(int(t.Result)), "Description", t.Description()))
	case packets.CountPrimesTaskType:
		s.Logger.Info(logger.FormatLogMessage("Task ID", strconv.Itoa(taskId), "Result", strconv.Itoa(int(t.IntResult)), "Description", t.Description()))
	default:
		s.Logger.Warning("msg", "Unknown Task Type")
	}
}
