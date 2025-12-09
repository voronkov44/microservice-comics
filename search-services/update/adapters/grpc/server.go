package grpc

import (
	"context"
	"errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	updatepb "yadro.com/course/proto/update"
	"yadro.com/course/update/core"
)

type Notifier interface {
	NotifyDBUpdated(ctx context.Context)
}

type Server struct {
	updatepb.UnimplementedUpdateServer
	service  core.Updater
	notifier Notifier
}

func NewServer(service core.Updater, notifier Notifier) *Server {
	return &Server{
		service:  service,
		notifier: notifier,
	}
}

func (s *Server) Ping(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *Server) Status(ctx context.Context, _ *emptypb.Empty) (*updatepb.StatusReply, error) {
	st := s.service.Status(ctx)

	var protoSt updatepb.Status
	switch st {
	case core.StatusRunning:
		protoSt = updatepb.Status_STATUS_RUNNING
	case core.StatusIdle:
		protoSt = updatepb.Status_STATUS_IDLE
	default:
		protoSt = updatepb.Status_STATUS_UNSPECIFIED
	}

	return &updatepb.StatusReply{Status: protoSt}, nil
}

func (s *Server) Update(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if err := s.service.Update(ctx); err != nil {
		if errors.Is(err, core.ErrAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, "update already running")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Уведомляем брокер, что база обновилась
	if s.notifier != nil {
		s.notifier.NotifyDBUpdated(ctx)
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) Stats(ctx context.Context, _ *emptypb.Empty) (*updatepb.StatsReply, error) {
	st, err := s.service.Stats(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &updatepb.StatsReply{
		WordsTotal:    int64(st.WordsTotal),
		WordsUnique:   int64(st.WordsUnique),
		ComicsFetched: int64(st.ComicsFetched),
		ComicsTotal:   int64(st.ComicsTotal),
	}, nil
}

func (s *Server) Drop(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if err := s.service.Drop(ctx); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// При дропе так же шлем сообщение, чтобы очистить индекс
	if s.notifier != nil {
		s.notifier.NotifyDBUpdated(ctx)
	}

	return &emptypb.Empty{}, nil
}
