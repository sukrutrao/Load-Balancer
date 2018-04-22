package master

import (
	/*	"net"

		"github.com/GoodDeeds/load-balancer/common/constants"*/
	"github.com/GoodDeeds/load-balancer/common/packets"
)

type TaskRequestPacket struct {
	TaskId int
	Task   string // TODO - change this
	Load   int
}

type TaskRequestResponsePacket struct {
	TaskId int
	Accept bool
}

type TaskResultResponsePacket struct {
	TaskId int
	Result TaskResult
}

type TaskStatusRequestPacket struct {
	TaskId int
}

type TaskStatusResponsePacket struct {
	TaskId     int
	TaskStatus Status // from status constants in constants.go
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

// TODO - this should be in slave.go
type TaskResult struct {
	Result string
}

func (m *Master) assignTaskPacket(t *MasterTask) interface{} {
	packet := packets.TaskRequestPacket{t.TaskId, t.Task, t.Load}
	return packet
}

// not sure if this is needed
// requests slave to provide status of a task assigned to it
func (m *Master) requestTaskStatusPacket(t *MasterTask) interface{} {
	packet := packets.TaskStatusRequestPacket{t.TaskId}
	return packet
}

// receives task status response, does not do anything right now
func (m *Master) handleTaskStatusResponse(packet interface{}) {
	// TODO - what do you do once you get the status?
}

// recieves result of task from slave and displays it
func (m *Master) handleTaskResult(packet interface{}) {
	resultPacket := TaskResultResponsePacket{packet}
	m.Logger.Info(logger.FormatLogMessage("Task ID completed", *resultPacket.TaskId))
	m.Logger.Info(logger.FormatLogMessage("Result", *resultPacket.Result.Result))
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
		TaskStatus: Unassigned}
	m.tasks[taskId] = t
	return t
}

// takes a task, finds which slave to assign to, assigns it in task packet, and returns slave index
func (m *Master) assignTask(t *Task) *Slave {
	slaveAssigned := m.slavePool.slaves[0] // TODO Fix this based on algorithm for load balancing
	*t.AssignedTo = slaveAssigned
	*t.IsAssigned = true
	return slaveAssigned
}
