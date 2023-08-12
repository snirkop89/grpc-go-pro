package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"

	pb "github.com/snirkop89/grpc-go-pro/proto/todo/v1"
)

type server struct {
	d db
	pb.UnimplementedTodoServiceServer
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		log.Fatalln("usage: server [IP_ADDR]")
	}

	addr := args[0]
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v\n", addr)
	}
	defer func(list net.Listener) {
		if err := lis.Close(); err != nil {
			log.Fatalf("unexpected error: %v", err)
		}
	}(lis)

	log.Printf("listening at %s\n", addr)

	opts := []grpc.ServerOption{}
	s := grpc.NewServer(opts...)
	pb.RegisterTodoServiceServer(s, &server{d: New()})
	defer s.Stop()
	// registration of endpoints
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v\n", err)
	}
}
