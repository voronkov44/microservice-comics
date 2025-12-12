package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"google.golang.org/protobuf/types/known/emptypb"

	searchpb "yadro.com/course/proto/search"
	"yadro.com/course/search/core"
)

func NewServer(service core.Search) *Server {
	return &Server{service: service}
}

type Server struct {
	searchpb.UnimplementedSearchServer
	service core.Search
}

func (s *Server) Ping(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if err := s.service.Ping(ctx); err != nil {
		return nil, status.Error(codes.Unavailable, err.Error())
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) Find(ctx context.Context, in *searchpb.SearchRequest) (*searchpb.SearchReply, error) {
	comics, err := s.service.Find(ctx, in.GetPhrase(), in.GetLimit())
	if err != nil {
		switch {
		case errors.Is(err, core.ErrEmptyPhrase),
			errors.Is(err, core.ErrBadArguments),
			errors.Is(err, core.ErrToLargeLimit):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case errors.Is(err, core.ErrUnavailable):
			return nil, status.Error(codes.Unavailable, err.Error())
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	res := &searchpb.SearchReply{
		Comics: make([]*searchpb.ComicReply, 0, len(comics)),
		Total:  uint32(len(comics)),
	}

	for _, c := range comics {
		res.Comics = append(res.Comics, &searchpb.ComicReply{
			Id:  uint32(c.ID),
			Url: c.URL,
		})
	}

	return res, nil
}

func (s *Server) IndexedSearch(ctx context.Context, in *searchpb.SearchRequest) (*searchpb.SearchReply, error) {
	comics, total, err := s.service.IndexedSearch(ctx, in.GetPhrase(), in.GetLimit())
	if err != nil {
		switch {
		case errors.Is(err, core.ErrEmptyPhrase),
			errors.Is(err, core.ErrBadArguments),
			errors.Is(err, core.ErrToLargeLimit):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case errors.Is(err, core.ErrUnavailable):
			return nil, status.Error(codes.Unavailable, err.Error())
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	res := &searchpb.SearchReply{
		Comics: make([]*searchpb.ComicReply, 0, len(comics)),
		Total:  total,
	}

	for _, c := range comics {
		res.Comics = append(res.Comics, &searchpb.ComicReply{
			Id:  uint32(c.ID),
			Url: c.URL,
		})
	}

	return res, nil
}

func (s *Server) GetIDComic(ctx context.Context, in *searchpb.ComicByIDRequest) (*searchpb.ComicReply, error) {
	id := int(in.GetId())
	comic, err := s.service.GetComicByID(ctx, id)
	if err != nil {
		switch {
		case errors.Is(err, core.ErrBadArguments):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case errors.Is(err, core.ErrComicNotFound):
			return nil, status.Error(codes.NotFound, err.Error())
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &searchpb.ComicReply{
		Id:  uint32(comic.ID),
		Url: comic.URL,
	}, nil
}

func (s *Server) GetAllComics(ctx context.Context, in *searchpb.ComicsPageRequest) (*searchpb.SearchReply, error) {
	comics, total, err := s.service.GetAllComics(ctx, in.GetPage(), in.GetPerPage())
	if err != nil {
		switch {
		case errors.Is(err, core.ErrBadArguments),
			errors.Is(err, core.ErrToLargeLimit),
			errors.Is(err, core.ErrEmptyPhrase):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case errors.Is(err, core.ErrUnavailable):
			return nil, status.Error(codes.Unavailable, err.Error())
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	res := &searchpb.SearchReply{
		Comics: make([]*searchpb.ComicReply, 0, len(comics)),
		Total:  total,
	}

	for _, c := range comics {
		res.Comics = append(res.Comics, &searchpb.ComicReply{
			Id:  uint32(c.ID),
			Url: c.URL,
		})
	}

	return res, nil
}

func (s *Server) GetRandomComic(ctx context.Context, _ *emptypb.Empty) (*searchpb.ComicReply, error) {
	comic, err := s.service.RandomComic(ctx)
	if err != nil {
		switch {
		case errors.Is(err, core.ErrUnavailable):
			return nil, status.Error(codes.Unavailable, err.Error())
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &searchpb.ComicReply{
		Id:  uint32(comic.ID),
		Url: comic.URL,
	}, nil
}
