package main

import "time"

type db interface {
	addTask(description string, dueDate time.Time) (uint64, error)
	getTasks(f func(any) error) error
	updateTask(id uint64, description string, dueDate time.Time, done bool) error
	deleteTask(id uint64) error
}
