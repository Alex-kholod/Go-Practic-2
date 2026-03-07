package grpc

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pz1/proto/authpb"
)

type server struct {
	authpb.UnimplementedAuthServiceServer
}

func (s *server) Verify(ctx context.Context, req *authpb.VerifyRequest) (*authpb.VerifyResponse, error) {
	token := req.GetToken()
	if token == "" {
		return nil, status.Error(codes.Unauthenticated, "missing token")
	}

	if token != "demo-token" {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	return &authpb.VerifyResponse{
		Valid:   true,
		Subject: "student",
	}, nil
}

func RunGRPCServer(port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	s := grpc.NewServer()
	authpb.RegisterAuthServiceServer(s, &server{})

	log.Printf("Auth gRPC server listening on :%s", port)
	return s.Serve(lis)
}
