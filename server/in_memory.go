package main

import (
	"fmt"
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

func (d *inMemoryDB) getTasks(f func(any) error) error {
	for _, task := range d.tasks {
		if err := f(task); err != nil {
			return err
		}
	}
	return nil
}

func (d *inMemoryDB) updateTask(id uint64, description string, dueDate time.Time, done bool) error {
	for i, task := range d.tasks {
		if task.Id == id {
			t := d.tasks[i]
			t.Description = description
			t.DueDate = timestamppb.New(dueDate)
			t.Done = done
			return nil
		}
	}
	return fmt.Errorf("task with id %d not found", id)
}

func (d *inMemoryDB) deleteTask(id uint64) error {
	for i, task := range d.tasks {
		if task.Id == id {
			d.tasks = append(d.tasks[:i], d.tasks[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("task with id %d not found", id)
}
