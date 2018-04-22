package slave

import (
	/*"net"*/

	"github.com/GoodDeeds/load-balancer/common/constants"
	"github.com/GoodDeeds/load-balancer/common/packets"
)

// TODO - I don't remember what this is supposed to do
/*func (s *Slave) getTask(t *Task, p interface{}) error {

	if s.currentLoad+p.Load > s.maxLoad {
		return "Load exceeded"
	} else {
		t := Task{p.TaskId, p.Task, p.Load, constants.Incomplete}
		go handleTask(t)
		return nil
	}
}*/

func (s *Slave) respondTaskStatusPacket(p interface{}, response interface{}) {
	p = Task.(p)
	status := getStatus(p.TaskId)
	response := TaskStatusResponse{p.TaskId, status}
}

func (s *Slave) sendTaskResult(t *Task, p interface{}) err {
	if t.TaskStatus != constants.Complete {
		return "Task is not yet complete"
	}
	taskResult := *t.Result
	p := TaskResultResponse{t.TaskId, taskResult}
	return nil
}

func (s *Slave) getStatus(taskId int) (status constants.Status) {
	task := s.Tasks[taskId]

}
