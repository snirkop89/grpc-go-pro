package main

import (
	"context"
	"io"
	"log"
	"net"
	"testing"
	"time"

	pb "github.com/snirkop89/grpc-go-pro/proto/todo/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const bufSize = 1024 * 1024

var (
	lis    *bufconn.Listener
	fakeDB *FakeDb = NewFakeDb()
)

func init() {
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	var testServer *server = &server{
		d: fakeDB,
	}
	pb.RegisterTodoServiceServer(s, testServer)
	go func() {
		if err := s.Serve(lis); err != nil && err.Error() != "closed" {
			log.Fatalf("server exited with error: %v\n", err)
		}
	}()
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func TestRunAll(t *testing.T) {
	t.Cleanup(func() {
		t.Log("Closing connection")
		lis.Close()
	})
	t.Run("AddTaskTests", func(t *testing.T) {
		t.Run("TestAddTaskEmptyDescription", testAddTaskEmptyDescription)
		t.Run("TestAddTaskUnavailableDb", testAddTaskUnavailableDb)
	})

	t.Run("ListTasks", func(t *testing.T) {
		t.Run("TestListTasks", testListTasks)
	})

	t.Run("UpdateTasks", testUpdateTasks)
	t.Run("DeleteTasks", testUpdateTasks)

}

const (
	errorInvalidDescription = "invalid AddTaskRequest.Description: value length must be at least 1 runes"
	errorNoDatabaseAccess   = "unexpected error: couldn't access the database"
)

func testAddTaskEmptyDescription(t *testing.T) {
	conn, c := newClient(t)
	defer conn.Close()
	req := &pb.AddTaskRequest{}
	_, err := c.AddTask(context.TODO(), req)
	if !errorIs(err, codes.Unknown, errorInvalidDescription) {
		t.Errorf(
			"expected Unknown with message %q, got %v",
			errorInvalidDescription, err,
		)
	}
}

func testAddTaskUnavailableDb(t *testing.T) {
	conn, c := newClient(t)
	defer conn.Close()
	newDb := NewFakeDb(IsAvailable(false))
	*fakeDB = *newDb
	req := &pb.AddTaskRequest{
		Description: "test",
		DueDate:     timestamppb.New(time.Now().Add(5 * time.Hour)),
	}
	_, err := c.AddTask(context.TODO(), req)
	fakeDB.Reset()
	if !errorIs(err, codes.Internal, errorNoDatabaseAccess) {
		t.Errorf("expected Internal, got %v", err)
	}
}

func testListTasks(t *testing.T) {
	conn, c := newClient(t)
	defer conn.Close()
	fakeDB.d.tasks = []*pb.Task{
		{}, {}, {}, // 3 empty tasks
	}
	expectedRead := len(fakeDB.d.tasks)
	req := &pb.ListTasksRequest{}
	count := 0
	res, err := c.ListTasks(context.TODO(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for {
		_, err := res.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Errorf("error while reading stream: %v", err)
		}
		count++
	}
	if count != expectedRead {
		t.Errorf("expected reading %d tasks, read %d", expectedRead, count)
	}
}

func testUpdateTasks(t *testing.T) {
	conn, c := newClient(t)
	defer conn.Close()
	fakeDB.d.tasks = []*pb.Task{
		{Id: 0, Description: "test1"},
		{Id: 1, Description: "test2"},
		{Id: 2, Description: "test3"},
	}
	requests := []*pb.UpdateTasksRequest{
		{Id: 0}, {Id: 1}, {Id: 2},
	}
	expectedUpdates := len(requests)
	stream, err := c.UpdateTasks(context.TODO())
	count := 0
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	for _, req := range requests {
		if err := stream.Send(req); err != nil {
			t.Fatal(err)
		}
		count++
	}
	_, err = stream.CloseAndRecv()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if count != expectedUpdates {
		t.Errorf("expected updating %d tasks, updated %d", expectedUpdates, count)
	}
}

func testDeleteTasks(t *testing.T) {
	conn, c := newClient(t)
	defer conn.Close()
	fakeDB.d.tasks = []*pb.Task{
		{Id: 1}, {Id: 2}, {Id: 3},
	}
	expectedRead := len(fakeDB.d.tasks)
	waitc := make(chan countAndError)
	requests := []*pb.DeleteTasksRequest{
		{Id: 1}, {Id: 2}, {Id: 3},
	}
	stream, err := c.DeleteTasks(context.TODO())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	go sendRequestOverStream(stream, requests, waitc)
	go readResponsesOverStream(stream, waitc)
	countAndError := <-waitc
	if countAndError.err != nil {
		t.Errorf("expected error: %v", countAndError.err)
	}
	if countAndError.count != expectedRead {
		t.Errorf("expected reading %d responses, read %d", expectedRead, countAndError.count)
	}
}

type countAndError struct {
	count int
	err   error
}

func sendRequestOverStream(stream pb.TodoService_DeleteTasksClient, requests []*pb.DeleteTasksRequest, waitc chan countAndError) {
	for _, req := range requests {
		if err := stream.Send(req); err != nil {
			waitc <- countAndError{err: err}
			close(waitc)
			return
		}
	}
	if err := stream.CloseSend(); err != nil {
		waitc <- countAndError{err: err}
		close(waitc)
	}
}

func readResponsesOverStream(stream pb.TodoService_DeleteTasksClient, waitc chan countAndError) {
	count := 0

	for {
		_, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			waitc <- countAndError{err: err}
			close(waitc)
			return
		}
		count++
	}
	waitc <- countAndError{count: count}
	close(waitc)
}
