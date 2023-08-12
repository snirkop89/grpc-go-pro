package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	pb "github.com/snirkop89/grpc-go-pro/proto/todo/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		log.Fatalln("usage: client [IP_ADDR]")
	}
	addr := args[0]

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	c := pb.NewTodoServiceClient(conn)
	fmt.Println("-----ADD-----")
	dueDate := time.Now().Add(5 * time.Second)
	addTask(c, "This ia a task", dueDate)
	fmt.Println("-------------")
	defer func(conn *grpc.ClientConn) {
		if err := conn.Close(); err != nil {
			log.Fatalf("unexpected error: %v", err)
		}
	}(conn)
}

func addTask(c pb.TodoServiceClient, description string, dueDate time.Time) uint64 {
	req := &pb.AddTaskRequest{
		Description: description,
		DueDate:     timestamppb.New(dueDate),
	}
	res, err := c.AddTask(context.Background(), req)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("added task: %d\n", res.Id)
	return res.Id
}
