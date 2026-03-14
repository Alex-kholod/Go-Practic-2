package authclient

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"pz1/proto/authpb"
)

type GRPCClient struct {
	conn   *grpc.ClientConn
	client authpb.AuthServiceClient
}

func NewGRPCClient(addr string) (*GRPCClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, err
	}
	client := authpb.NewAuthServiceClient(conn)
	return &GRPCClient{conn: conn, client: client}, nil
}

func (c *GRPCClient) VerifyToken(ctx context.Context, token string) (string, int, error) {
	req := &authpb.VerifyRequest{Token: token}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	resp, err := c.client.Verify(ctx, req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.Unauthenticated {
			return "", 401, err
		}
		return "", 503, err
	}

	if !resp.Valid {
		return "", 401, nil
	}

	return resp.Subject, 200, nil
}
