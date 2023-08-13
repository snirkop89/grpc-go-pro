package main

import (
	"context"
	"io"
	"log"
	"time"

	pb "github.com/snirkop89/grpc-go-pro/proto/todo/v1"
	"google.golang.org/protobuf/proto"
)

func (s *server) AddTask(ctx context.Context, in *pb.AddTaskRequest) (*pb.AddTaskResponse, error) {
	id, err := s.d.addTask(in.Description, in.DueDate.AsTime())
	if err != nil {
		return nil, err
	}
	return &pb.AddTaskResponse{Id: id}, nil
}

func (s *server) ListTasks(req *pb.ListTasksRequest, stream pb.TodoService_ListTasksServer) error {
	return s.d.getTasks(func(a any) error {
		task := a.(*pb.Task)
		overdue := task.DueDate != nil && !task.Done &&
			task.DueDate.AsTime().Before(time.Now().UTC())
		err := stream.Send(&pb.ListTasksResponse{
			Task:    task,
			Overdue: overdue,
		})
		return err
	})
}

func (s *server) UpdateTasks(stream pb.TodoService_UpdateTasksServer) error {
	totalLength := 0
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			log.Println("TOTAL:", totalLength)
			return stream.SendAndClose(&pb.UpdateTaskResponse{})
		}
		if err != nil {
			return err
		}
		out, _ := proto.Marshal(req)
		totalLength += len(out)
		s.d.updateTask(
			req.Task.Id,
			req.Task.Description,
			req.Task.DueDate.AsTime(),
			req.Task.Done,
		)
	}
}

func (s *server) DeleteTasks(stream pb.TodoService_DeleteTasksServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		s.d.deleteTask(req.Id)
		stream.Send(&pb.DeleteTasksResponse{})
	}
}
