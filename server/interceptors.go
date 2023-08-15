package main

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	authTokenKey   string = "auth_token"
	authTokenValue string = "authd"
)

func unaryLogInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	log.Println(info.FullMethod, "called")
	return handler(ctx, req)
}

func streamLogInterceptor(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	log.Println(info.FullMethod, "called")
	return handler(srv, ss)
}
func validateAuthToken(ctx context.Context) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "incorrect auth_token")
	}
	if t, ok := md["auth_token"]; ok {
		switch {
		case len(t) != 1:
			fmt.Printf("token: %v\n", t)
			return nil, status.Errorf(codes.InvalidArgument, "auth_token should contain only 1 value")
		case t[0] != "authd":
			return nil, status.Errorf(codes.Unauthenticated, "incorrect auth_token")
		}
	} else {
		return nil, status.Errorf(codes.Unauthenticated, "failed to get auth token")
	}
	return ctx, nil
}
