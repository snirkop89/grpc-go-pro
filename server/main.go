package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	pb "github.com/snirkop89/grpc-go-pro/proto/todo/v2"
)

type server struct {
	d db
	pb.UnimplementedTodoServiceServer
}

func main() {
	args := os.Args[1:]
	if len(args) != 2 {
		log.Fatalln("usage: server [GRPC_IP_ADDR] [METRICS_IP_ADDR]")
	}
	grpcAddr := args[0]
	httpAddr := args[1]

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v\n", grpcAddr)
	}
	defer lis.Close()

	g, ctx := errgroup.WithContext(ctx)

	grpcSrv, err := newGrpcServer(lis)
	if err != nil {
		log.Fatal(err)
	}
	g.Go(func() error {
		log.Printf("gRPC server listening at %s\n", grpcAddr)
		if err := grpcSrv.Serve(lis); err != nil {
			log.Printf("failed starting gRPC server: %v\n", err)
			return err
		}
		log.Println("gRPC server shutdown")
		return nil
	})
	metricsServer := newMetricsServer(httpAddr)
	g.Go(func() error {
		log.Printf("metrics server listening at %s\n", httpAddr)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("failed to serve metrics: %v\n", err)
			return err
		}
		log.Println("metrics server shutdown")
		return nil
	})

	<-ctx.Done()
	cancel()
	timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer timeoutCancel()
	log.Println("Shutting down server, please wait...")
	grpcSrv.GracefulStop()
	metricsServer.Shutdown(timeoutCtx)
	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}
}

func newMetricsServer(httpAddr string) *http.Server {
	m := http.NewServeMux()
	return &http.Server{
		Addr:    httpAddr,
		Handler: m,
	}
}

func newGrpcServer(lis net.Listener) (*grpc.Server, error) {
	logger := log.New(os.Stderr, "", log.Ldate|log.Ltime)

	creds, err := credentials.NewServerTLSFromFile("./certs/server_cert.pem", "./certs/server_key.pem")
	if err != nil {
		log.Fatal(err)
	}

	opts := []grpc.ServerOption{
		grpc.Creds(creds),
		grpc.ChainUnaryInterceptor(
			auth.UnaryServerInterceptor(validateAuthToken),
			logging.UnaryServerInterceptor(logCalls(logger)),
		),
		grpc.ChainStreamInterceptor(
			auth.StreamServerInterceptor(validateAuthToken),
			logging.StreamServerInterceptor(logCalls(logger)),
		),
	}
	s := grpc.NewServer(opts...)
	pb.RegisterTodoServiceServer(s, &server{d: New()})
	return s, nil
}
