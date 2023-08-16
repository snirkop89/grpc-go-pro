package main

import (
	"context"
	"testing"

	pb "github.com/snirkop89/grpc-go-pro/proto/todo/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func newClient(t *testing.T) (*grpc.ClientConn, pb.TodoServiceClient) {
	ctx := context.Background()
	creds := grpc.WithTransportCredentials(insecure.NewCredentials())
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), creds)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	return conn, pb.NewTodoServiceClient(conn)
}

func errorIs(err error, code codes.Code, msg string) bool {
	if err != nil {
		if s, ok := status.FromError(err); ok {
			if code == s.Code() && s.Message() == msg {
				return true
			}
		}
	}
	return false
}
