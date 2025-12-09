package words

import (
	"context"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log/slog"
	"yadro.com/course/update/core"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	wordspb "yadro.com/course/proto/words"
)

type Client struct {
	log    *slog.Logger
	client wordspb.WordsClient
	conn   *grpc.ClientConn
}

func NewClient(address string, log *slog.Logger) (*Client, error) {
	// ClientConnection - создаем подключение для локальной сети/compose
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("new grpc client for  %s: %w", address, err)
	}

	return &Client{
		client: wordspb.NewWordsClient(conn),
		conn:   conn,
		log:    log,
	}, nil

}

// Close grpc connection
func (c *Client) Close() error { return c.conn.Close() }

// Norm реализация порта normalizer
// Делает grpc вызов Norm и маппит ошибки в доменные
func (c *Client) Norm(ctx context.Context, phrase string) ([]string, error) {
	resp, err := c.client.Norm(ctx, &wordspb.WordsRequest{Phrase: phrase})
	if err != nil {
		switch status.Code(err) {
		case codes.ResourceExhausted:
			return nil, core.ErrBadArguments
		case codes.Unavailable, codes.DeadlineExceeded:
			return nil, core.ErrUnavailable
		default:
			return nil, err
		}
	}
	return resp.GetWords(), nil
}
