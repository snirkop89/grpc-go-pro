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
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/ratelimit"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	pb "github.com/snirkop89/grpc-go-pro/proto/todo/v2"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
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

	srvMetrics := grpcprom.NewServerMetrics(
		grpcprom.WithServerHandlingTimeHistogram(
			grpcprom.WithHistogramBuckets([]float64{0.001, 0.01,
				0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120}),
		),
	)

	grpcSrv, err := newGrpcServer(lis, srvMetrics)
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

	reg := prometheus.NewRegistry()
	reg.MustRegister(srvMetrics)

	metricsServer := newMetricsServer(httpAddr, reg)
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

func newMetricsServer(httpAddr string, reg *prometheus.Registry) *http.Server {
	m := http.NewServeMux()
	m.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	return &http.Server{
		Addr:    httpAddr,
		Handler: m,
	}
}

func newGrpcServer(lis net.Listener, srvMetrics *grpcprom.ServerMetrics) (*grpc.Server, error) {
	logger := log.New(os.Stderr, "", log.Ldate|log.Ltime)

	creds, err := credentials.NewServerTLSFromFile("./certs/server_cert.pem", "./certs/server_key.pem")
	if err != nil {
		log.Fatal(err)
	}

	limiter := &simpleLimiter{
		limiter: rate.NewLimiter(2, 4),
	}
	opts := []grpc.ServerOption{
		grpc.Creds(creds),
		grpc.ChainUnaryInterceptor(
			ratelimit.UnaryServerInterceptor(limiter),
			otelgrpc.UnaryServerInterceptor(),
			srvMetrics.UnaryServerInterceptor(),
			auth.UnaryServerInterceptor(validateAuthToken),
			logging.UnaryServerInterceptor(logCalls(logger)),
		),
		grpc.ChainStreamInterceptor(
			ratelimit.StreamServerInterceptor(limiter),
			otelgrpc.StreamServerInterceptor(),
			srvMetrics.StreamServerInterceptor(),
			auth.StreamServerInterceptor(validateAuthToken),
			logging.StreamServerInterceptor(logCalls(logger)),
		),
	}
	s := grpc.NewServer(opts...)
	pb.RegisterTodoServiceServer(s, &server{d: New()})
	return s, nil
}
