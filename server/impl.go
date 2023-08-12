package main

import (
	"context"

	pb "github.com/snirkop89/grpc-go-pro/proto/todo/v1"
)

func (s *server) AddTask(ctx context.Context, in *pb.AddTaskRequest) (*pb.AddTaskResponse, error) {
	id, err := s.d.addTask(in.Description, in.DueDate.AsTime())
	if err != nil {
		return nil, err
	}
	return &pb.AddTaskResponse{Id: id}, nil
}
