package agent

import (
	"log"
	"net"
	"os/exec"

	"github.com/chrislusf/glow/driver/cmd"
	"github.com/chrislusf/glow/resource"
	"github.com/golang/protobuf/proto"
)

func (as *AgentServer) handleCommandConnection(conn net.Conn,
	command *cmd.ControlMessage) *cmd.ControlMessage {
	reply := &cmd.ControlMessage{}
	if command.GetType() == cmd.ControlMessage_StartRequest {
		reply.Type = cmd.ControlMessage_StartResponse.Enum()
		reply.StartResponse = as.handleStart(conn, command.StartRequest)
	}
	if command.GetType() == cmd.ControlMessage_DeleteDatasetShardRequest {
		reply.Type = cmd.ControlMessage_DeleteDatasetShardResponse.Enum()
		reply.DeleteDatasetShardResponse = as.handleDeleteDatasetShard(conn, command.DeleteDatasetShardRequest)
	}
	// TODO: skip return reply for now
	return nil
}

func (as *AgentServer) handleStart(conn net.Conn,
	startRequest *cmd.StartRequest) *cmd.StartResponse {
	reply := &cmd.StartResponse{}

	// println("received command:", *startRequest.Path)

	allocated := resource.ComputeResource{
		CPUCount: int(startRequest.Resource.GetCpuCount()),
		MemoryMB: int64(startRequest.Resource.GetMemory()),
	}

	as.plusAllocated(allocated)
	defer as.minusAllocated(allocated)

	cmd := exec.Command(
		*startRequest.Path,
		startRequest.Args...,
	)
	cmd.Env = startRequest.Envs
	cmd.Dir = *startRequest.Dir
	cmd.Stdout = conn
	cmd.Stderr = conn
	err := cmd.Start()
	if err != nil {
		log.Printf("Failed to start command %s under %s: %v",
			cmd.Path, cmd.Dir, err)
		*reply.Error = err.Error()
	} else {
		reply.Pid = proto.Int32(int32(cmd.Process.Pid))
	}

	cmd.Wait()

	return reply
}

func (as *AgentServer) plusAllocated(allocated resource.ComputeResource) {
	as.allocatedResourceLock.Lock()
	defer as.allocatedResourceLock.Unlock()
	*as.allocatedResource = as.allocatedResource.Plus(allocated)
}

func (as *AgentServer) minusAllocated(allocated resource.ComputeResource) {
	as.allocatedResourceLock.Lock()
	defer as.allocatedResourceLock.Unlock()
	*as.allocatedResource = as.allocatedResource.Minus(allocated)
}

func (as *AgentServer) handleDeleteDatasetShard(conn net.Conn,
	deleteRequest *cmd.DeleteDatasetShardRequest) *cmd.DeleteDatasetShardResponse {

	as.handleDelete(*deleteRequest.Name)

	return nil
}
