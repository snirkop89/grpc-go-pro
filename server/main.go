package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	pb "github.com/snirkop89/grpc-go-pro/proto/todo/v2"
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

	creds, err := credentials.NewServerTLSFromFile("./certs/server_cert.pem", "./certs/server_key.pem")
	if err != nil {
		log.Fatal(err)
	}

	opts := []grpc.ServerOption{
		grpc.Creds(creds),
		grpc.ChainUnaryInterceptor(auth.UnaryServerInterceptor(validateAuthToken), unaryLogInterceptor),
		grpc.ChainStreamInterceptor(auth.StreamServerInterceptor(validateAuthToken), streamLogInterceptor),
	}
	s := grpc.NewServer(opts...)
	pb.RegisterTodoServiceServer(s, &server{d: New()})
	defer s.Stop()
	// registration of endpoints
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v\n", err)
	}
}
