package main

import "time"

type db interface {
	addTask(description string, dueDate time.Time) (uint64, error)
}
