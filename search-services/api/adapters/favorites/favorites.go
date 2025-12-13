package favorites

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
	favoritespb "yadro.com/course/proto/favorites"
)

type Client struct {
	log    *slog.Logger
	client favoritespb.FavoritesClient
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
		client: favoritespb.NewFavoritesClient(conn),
		conn:   conn,
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

func (c *Client) Add(ctx context.Context, userID uint32, comicID int32) error {
	_, err := c.client.Add(ctx, &favoritespb.AddRequest{
		UserId:  userID,
		ComicId: comicID,
	})
	if err != nil {
		switch status.Code(err) {
		case codes.InvalidArgument:
			return core.ErrBadArguments
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

func (c *Client) Delete(ctx context.Context, userID uint32, comicID int32) error {
	_, err := c.client.Delete(ctx, &favoritespb.DeleteRequest{
		UserId:  userID,
		ComicId: comicID,
	})
	if err != nil {
		switch status.Code(err) {
		case codes.InvalidArgument:
			return core.ErrBadArguments
		case codes.NotFound:
			return core.ErrNotFound
		case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled:
			return core.ErrUnavailable
		default:
			return err
		}
	}
	return nil
}

func (c *Client) List(ctx context.Context, userID uint32) ([]core.FavoriteItem, error) {
	resp, err := c.client.List(ctx, &favoritespb.ListRequest{
		UserId: userID,
	})
	if err != nil {
		switch status.Code(err) {
		case codes.InvalidArgument:
			return nil, core.ErrBadArguments
		case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled:
			return nil, core.ErrUnavailable
		default:
			return nil, err
		}
	}

	out := make([]core.FavoriteItem, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		out = append(out, core.FavoriteItem{
			ComicID:       it.GetComicId(),
			CreatedAtUnix: it.GetCreatedAtUnix(),
		})
	}
	return out, nil
}

var _ core.Favorites = (*Client)(nil)
