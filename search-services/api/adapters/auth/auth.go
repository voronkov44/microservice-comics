package auth

import (
	"context"
	"fmt"
	"google.golang.org/protobuf/types/known/emptypb"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"yadro.com/course/api/core"
	authpb "yadro.com/course/proto/auth"
)

type Client struct {
	log    *slog.Logger
	client authpb.AuthClient
	conn   *grpc.ClientConn
}

func NewClient(address string, log *slog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("new grpc client for %s: %w", address, err)
	}

	return &Client{
		log:    log,
		client: authpb.NewAuthClient(conn),
		conn:   conn,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx, &emptypb.Empty{})
	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled:
			return core.ErrUnavailable
		default:
			return err
		}
	}
	return nil
}

func (c *Client) Register(ctx context.Context, email, password string) (string, error) {
	resp, err := c.client.Register(ctx, &authpb.RegisterRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		switch status.Code(err) {
		case codes.InvalidArgument:
			return "", core.ErrInvalidEmail
		case codes.AlreadyExists:
			return "", core.ErrAlreadyExists
		case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled:
			return "", core.ErrUnavailable
		default:
			return "", err
		}
	}

	return resp.GetToken(), nil
}

func (c *Client) Login(ctx context.Context, email, password string) (string, error) {
	resp, err := c.client.Login(ctx, &authpb.LoginRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		switch status.Code(err) {
		case codes.InvalidArgument:
			return "", core.ErrInvalidEmail
		case codes.Unauthenticated:
			return "", core.ErrInvalidCredentials
		case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled:
			return "", core.ErrUnavailable
		default:
			return "", err
		}
	}

	return resp.GetToken(), nil
}
