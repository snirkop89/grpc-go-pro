package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	pb "github.com/snirkop89/grpc-go-pro/proto/todo/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		log.Fatalln("usage: client [IP_ADDR]")
	}
	addr := args[0]

	creds, err := credentials.NewClientTLSFromFile("./certs/ca_cert.pem", "x.test.example.com")
	if err != nil {
		log.Fatal(err)
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		// grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(unaryAuthInterceptor),
		grpc.WithStreamInterceptor(streamAuthInterceptor),
		// grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)),
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin": {}}]}`),
	}
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer func(conn *grpc.ClientConn) {
		if err := conn.Close(); err != nil {
			log.Fatalf("unexpected error: %v", err)
		}
	}(conn)
	c := pb.NewTodoServiceClient(conn)

	fmt.Println("-----ADD-----")
	dueDate := time.Now().Add(200 * time.Millisecond)
	id1 := addTask(c, "This ia a task", dueDate)
	dueDate = time.Now().Add(15 * time.Second)
	id2 := addTask(c, "This ia task #2", dueDate)
	dueDate = time.Now().Add(15 * time.Second)
	id3 := addTask(c, "This is overdue", dueDate)
	fmt.Println("-------------")

	fm, err := fieldmaskpb.New(&pb.Task{}, "id")
	if err != nil {
		log.Fatalf("unexpected error: %v", err)
	}
	fmt.Println("-----List----")
	printTasks(c, fm)
	fmt.Println("-------------")

	fmt.Println("-----Update----")
	updateTasks(c, []*pb.UpdateTasksRequest{
		{Id: id1, Description: "A better name for task 1"},
		{Id: id2, DueDate: timestamppb.New(dueDate.Add(5 * time.Hour))},
		{Id: id3, Done: true},
	}...)
	printTasks(c, fm)
	fmt.Println("-------------")

	fmt.Println("-----Delete----")
	deleteTasks(c, []*pb.DeleteTasksRequest{
		{Id: id1},
		{Id: id2},
		{Id: id3},
	}...)
	fmt.Println("-------------")

	fmt.Println("-----Error----")
	// addTask(c, "", dueDate)
	// addTask(c, "not empty", time.Now().Add(-5*time.Second))
	fmt.Println("-------------")
}

func addTask(c pb.TodoServiceClient, description string, dueDate time.Time) uint64 {
	req := &pb.AddTaskRequest{
		Description: description,
		DueDate:     timestamppb.New(dueDate),
	}
	res, err := c.AddTask(context.Background(), req)
	if err != nil {
		if s, ok := status.FromError(err); ok {
			switch s.Code() {
			case codes.InvalidArgument, codes.Internal:
				log.Fatalf("%s: %s", s.Code(), s.Message())
			default:
				log.Fatal(s)
			}
		} else {
			panic(err)
		}
	}
	fmt.Printf("added task: %d\n", res.Id)
	return res.Id
}

func printTasks(c pb.TodoServiceClient, fm *fieldmaskpb.FieldMask) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req := &pb.ListTasksRequest{
		Mask: fm,
	}
	stream, err := c.ListTasks(ctx, req)
	if err != nil {
		log.Fatalf("unexpected error: %v\n", err)
	}
	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("unexpected error: %v", err)
		}
		if res.Overdue {
			log.Printf("CANCEL called")
			cancel()
		}
		fmt.Println(res.Task.String(), "overdue: ", res.Overdue)
	}
}

func updateTasks(c pb.TodoServiceClient, reqs ...*pb.UpdateTasksRequest) {
	ctx := context.Background()
	// change "authd" to anything else to see the error
	ctx = metadata.AppendToOutgoingContext(ctx, "auth_token", "authd")
	stream, err := c.UpdateTasks(ctx)
	if err != nil {
		log.Fatalf("unexpected error: %v", err)
	}
	for _, req := range reqs {
		err := stream.Send(req)
		// if err != nil {
		// 	return
		// }
		if err != nil {
			log.Fatalf("unexpected error: %v", err)
		}
		if req.Id != 0 {
			fmt.Printf("updated task with id: %d\n", req.Id)
		}
	}
	if _, err := stream.CloseAndRecv(); err != nil {
		log.Fatalf("unexpected error: %v", err)
	}
}

func deleteTasks(c pb.TodoServiceClient, reqs ...*pb.DeleteTasksRequest) {
	stream, err := c.DeleteTasks(context.Background())
	if err != nil {
		log.Fatalf("unexpected error: %v", err)
	}
	waitc := make(chan struct{})
	go func() {
		for {
			_, err := stream.Recv()
			if err == io.EOF {
				close(waitc)
				break
			}
			if err != nil {
				log.Fatalf("error while receiving: %v", err)
			}
			log.Println("deleted tasks")
		}
	}()
	for _, req := range reqs {
		if err := stream.Send(req); err != nil {
			return
		}
	}
	if err := stream.CloseSend(); err != nil {
		return
	}
	<-waitc
}
