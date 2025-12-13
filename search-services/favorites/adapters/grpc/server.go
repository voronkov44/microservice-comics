package grpc

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"yadro.com/course/favorites/core"
	favoritespb "yadro.com/course/proto/favorites"
)

type Server struct {
	favoritespb.UnimplementedFavoritesServer
	log     *slog.Logger
	service *core.Service
}

func NewServer(log *slog.Logger, service *core.Service) *Server {
	return &Server{log: log, service: service}
}

func (s *Server) Add(ctx context.Context, req *favoritespb.AddRequest) (*emptypb.Empty, error) {
	err := s.service.Add(ctx, req.GetUserId(), req.GetComicId())
	if err != nil {
		switch {
		case errors.Is(err, core.ErrInvalidArgs):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case errors.Is(err, core.ErrAlreadyExists):
			return nil, status.Error(codes.AlreadyExists, err.Error())
		default:
			s.log.Error("add favorite failed", "error", err)
			return nil, status.Error(codes.Internal, "internal error")
		}
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) Delete(ctx context.Context, req *favoritespb.DeleteRequest) (*emptypb.Empty, error) {
	err := s.service.Delete(ctx, req.GetUserId(), req.GetComicId())
	if err != nil {
		switch {
		case errors.Is(err, core.ErrInvalidArgs):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case errors.Is(err, core.ErrNotFound):
			return nil, status.Error(codes.NotFound, err.Error())
		default:
			s.log.Error("delete favorite failed", "error", err)
			return nil, status.Error(codes.Internal, "internal error")
		}
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) List(ctx context.Context, req *favoritespb.ListRequest) (*favoritespb.ListResponse, error) {
	items, err := s.service.List(ctx, req.GetUserId())
	if err != nil {
		switch {
		case errors.Is(err, core.ErrInvalidArgs):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		default:
			s.log.Error("list favorites failed", "error", err)
			return nil, status.Error(codes.Internal, "internal error")
		}
	}

	resp := &favoritespb.ListResponse{Items: make([]*favoritespb.FavoriteItem, 0, len(items))}
	for _, it := range items {
		resp.Items = append(resp.Items, &favoritespb.FavoriteItem{
			ComicId:       it.ComicID,
			CreatedAtUnix: it.CreatedAt.Unix(),
		})
	}
	return resp, nil
}

func (s *Server) Ping(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if err := s.service.Ping(ctx); err != nil {
		return nil, status.Error(codes.Unavailable, err.Error())
	}
	return &emptypb.Empty{}, nil
}
