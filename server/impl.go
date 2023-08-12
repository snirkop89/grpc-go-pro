package main

import (
	"context"
	"time"

	pb "github.com/snirkop89/grpc-go-pro/proto/todo/v1"
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
