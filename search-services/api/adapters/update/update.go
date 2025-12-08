package update

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"yadro.com/course/api/core"
	updatepb "yadro.com/course/proto/update"
)

type Client struct {
	log    *slog.Logger
	client updatepb.UpdateClient
	conn   *grpc.ClientConn
}

func NewClient(address string, log *slog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{
		client: updatepb.NewUpdateClient(conn),
		conn:   conn,
		log:    log,
	}, nil
}

func (c *Client) Close() error { return c.conn.Close() }

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

func (c *Client) Status(ctx context.Context) (core.UpdateStatus, error) {
	resp, err := c.client.Status(ctx, &emptypb.Empty{})
	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled:
			return core.StatusUpdateUnknown, core.ErrUnavailable
		default:
			return core.StatusUpdateUnknown, err
		}
	}
	switch resp.GetStatus() {
	case updatepb.Status_STATUS_IDLE:
		return core.StatusUpdateIdle, nil
	case updatepb.Status_STATUS_RUNNING:
		return core.StatusUpdateRunning, nil
	default:
		return core.StatusUpdateUnknown, nil
	}
}

func (c *Client) Stats(ctx context.Context) (core.UpdateStats, error) {
	resp, err := c.client.Stats(ctx, &emptypb.Empty{})
	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled:
			return core.UpdateStats{}, core.ErrUnavailable
		default:
			return core.UpdateStats{}, err
		}
	}
	return core.UpdateStats{
		WordsTotal:    int(resp.GetWordsTotal()),
		WordsUnique:   int(resp.GetWordsUnique()),
		ComicsFetched: int(resp.GetComicsFetched()),
		ComicsTotal:   int(resp.GetComicsTotal()),
	}, nil
}

func (c *Client) Update(ctx context.Context) error {
	_, err := c.client.Update(ctx, &emptypb.Empty{})
	if err != nil {
		switch status.Code(err) {
		case codes.AlreadyExists:
			return core.ErrAlreadyExists
		case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled:
			return core.ErrUnavailable
		default:
			return err
		}
	}
	return nil
}

func (c *Client) Drop(ctx context.Context) error {
	_, err := c.client.Drop(ctx, &emptypb.Empty{})
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
