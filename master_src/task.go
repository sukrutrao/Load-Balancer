package master

import (
	"strconv"

	"github.com/GoodDeeds/load-balancer/common/logger"
	"github.com/GoodDeeds/load-balancer/common/packets"
)

func (m *Master) assignTaskPacket(t *MasterTask) packets.TaskRequestPacket {
	packet := packets.TaskRequestPacket{t.TaskId, t.Task, t.Load}
	return packet
}

// not sure if this is needed
// requests slave to provide status of a task assigned to it
func (m *Master) requestTaskStatusPacket(t *MasterTask) packets.TaskStatusRequestPacket {
	packet := packets.TaskStatusRequestPacket{t.TaskId}
	return packet
}

// receives task status response, does not do anything right now
func (s *Slave) handleTaskStatusResponse(packet packets.TaskStatusResponsePacket) {
	// TODO - what do you do once you get the status?
}

func (s *Slave) handleTaskRequestResponse(packet packets.TaskRequestResponsePacket) {
	if !packet.Accept {
		s.Logger.Warning(logger.FormatLogMessage("msg", "Slave did not accept task", "Task ID", strconv.Itoa(int(packet.TaskId))))
	} else {
		s.Logger.Info(logger.FormatLogMessage("msg", "Slave accepted task", "Task ID", strconv.Itoa(int(packet.TaskId))))
	}
}

// recieves result of task from slave and displays it
func (s *Slave) handleTaskResult(packet packets.TaskResultResponsePacket) {
	// resultPacket, ok := packet.(packets.TaskResultResponsePacket)
	// if !ok {
	// 	// TODO - handle error
	// }
	s.Logger.Info(logger.FormatLogMessage("Task ID completed", strconv.Itoa(int(packet.TaskId)), "Result", packet.Result))
	// TODO do something more meaningful
}

// takes task string and load and creates a task object
func (m *Master) createTask(task string, load int) *MasterTask {
	taskId := m.lastTaskId + 1
	t := MasterTask{TaskId: taskId,
		Task:       task,
		Load:       load,
		AssignedTo: nil,
		IsAssigned: false,
		TaskStatus: packets.Unassigned}
	m.tasks[taskId] = t
	m.lastTaskId += 1
	return &t // TODO - is this safe?
}

// takes a task, finds which slave to assign to, assigns it in task packet, and returns slave index
func (m *Master) assignTask(t *MasterTask) *Slave {
	slaveAssigned := m.slavePool.slaves[0] // TODO Fix this based on algorithm for load balancing
	t.AssignedTo = slaveAssigned
	t.IsAssigned = true
	return slaveAssigned
}
