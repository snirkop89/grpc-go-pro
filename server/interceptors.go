package main

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	authTokenKey   string = "auth_token"
	authTokenValue string = "authd"
)

func unaryAuthInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	if err := validateAuthToken(ctx); err != nil {
		return nil, err
	}
	return handler(ctx, req)
}

func streamAuthInterceptor(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if err := validateAuthToken(ss.Context()); err != nil {
		return err
	}
	return handler(srv, ss)
}

func validateAuthToken(ctx context.Context) error {
	md, _ := metadata.FromIncomingContext(ctx)
	if t, ok := md["auth_token"]; ok {
		switch {
		case len(t) != 1:
			return status.Errorf(codes.InvalidArgument, "auth_token should contain only 1 value")
		case t[0] != "authd":
			return status.Errorf(codes.Unauthenticated, "incorrect auth_token")
		}
	} else {
		return status.Errorf(codes.Unauthenticated, "failed to get auth token")
	}
	return nil
}
