package main

import (
	"context"
	"fmt"
	"log"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	authTokenKey   string = "auth_token"
	authTokenValue string = "authd"
	grpcService           = 5
	grpcMethod            = 7
)

func validateAuthToken(ctx context.Context) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "incorrect auth_token")
	}
	if t, ok := md[authTokenKey]; ok {
		switch {
		case len(t) != 1:
			fmt.Printf("token: %v\n", t)
			return nil, status.Errorf(codes.InvalidArgument, "auth_token should contain only 1 value")
		case t[0] != authTokenValue:
			return nil, status.Errorf(codes.Unauthenticated, "incorrect auth_token")
		}
	} else {
		return nil, status.Errorf(codes.Unauthenticated, "failed to get auth token")
	}
	return ctx, nil
}

func logCalls(l *log.Logger) logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, level logging.Level, msg string, fields ...any) {
		switch level {
		case logging.LevelDebug:
			msg = fmt.Sprintf("DEBUG: %v", msg)
		case logging.LevelInfo:
			msg = fmt.Sprintf("INFO: %v", msg)
		case logging.LevelWarn:
			msg = fmt.Sprintf("WARN: %v", msg)
		case logging.LevelError:
			msg = fmt.Sprintf("ERROR: %v", msg)
		default:
			panic(fmt.Sprintf("unknown level %v", level))
		}
		// As long as the logging library doesn't change, it'll work
		l.Println(msg, fields[grpcService], fields[grpcMethod])
	})
}
