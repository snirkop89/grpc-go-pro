package main

import (
	"time"

	pb "github.com/snirkop89/grpc-go-pro/proto/todo/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type inMemoryDB struct {
	tasks []*pb.Task
}

func New() *inMemoryDB {
	return &inMemoryDB{}
}

func (d *inMemoryDB) addTask(description string, dueDate time.Time) (uint64, error) {
	nextID := uint64(len(d.tasks) + 1)
	task := &pb.Task{
		Id:          nextID,
		Description: description,
		DueDate:     timestamppb.New(dueDate),
	}
	d.tasks = append(d.tasks, task)
	return nextID, nil
}
