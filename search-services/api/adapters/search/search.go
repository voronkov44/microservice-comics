package search

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"google.golang.org/protobuf/types/known/emptypb"

	"yadro.com/course/api/core"
	searchpb "yadro.com/course/proto/search"
)

type Client struct {
	log    *slog.Logger
	client searchpb.SearchClient
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
		client: searchpb.NewSearchClient(conn),
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

func (c *Client) Find(ctx context.Context, phrase string, limit uint32) (core.SearchResult, error) {
	res, err := c.client.Find(ctx, &searchpb.SearchRequest{
		Phrase: phrase,
		Limit:  limit,
	})
	if err != nil {
		switch status.Code(err) {
		case codes.InvalidArgument:
			return core.SearchResult{}, core.ErrBadArguments
		case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled:
			return core.SearchResult{}, core.ErrUnavailable
		default:
			return core.SearchResult{}, err
		}
	}

	out := core.SearchResult{
		Comics: make([]core.SearchComic, 0, len(res.GetComics())),
		Total:  int(res.GetTotal()),
	}

	for _, cr := range res.GetComics() {
		out.Comics = append(out.Comics, core.SearchComic{
			ID:  int(cr.GetId()),
			URL: cr.GetUrl(),
		})
	}

	return out, nil
}

func (c *Client) IndexedSearch(ctx context.Context, phrase string, limit uint32) (core.SearchResult, error) {
	res, err := c.client.IndexedSearch(ctx, &searchpb.SearchRequest{
		Phrase: phrase,
		Limit:  limit,
	})
	if err != nil {
		switch status.Code(err) {
		case codes.InvalidArgument:
			return core.SearchResult{}, core.ErrBadArguments
		case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled:
			return core.SearchResult{}, core.ErrUnavailable
		default:
			return core.SearchResult{}, err
		}
	}

	out := core.SearchResult{
		Comics: make([]core.SearchComic, 0, len(res.GetComics())),
		Total:  int(res.GetTotal()),
	}

	for _, cr := range res.GetComics() {
		out.Comics = append(out.Comics, core.SearchComic{
			ID:  int(cr.GetId()),
			URL: cr.GetUrl(),
		})
	}

	return out, nil
}
