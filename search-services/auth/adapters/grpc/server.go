package grpc

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"yadro.com/course/auth/core"
	authpb "yadro.com/course/proto/auth"
)

type Server struct {
	authpb.UnimplementedAuthServer
	log     *slog.Logger
	service *core.Service
}

func NewServer(log *slog.Logger, service *core.Service) *Server {
	return &Server{
		log:     log,
		service: service,
	}
}

func (s *Server) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.TokenResponse, error) {
	if req.GetEmail() == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	token, err := s.service.Register(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		switch {
		case errors.Is(err, core.ErrUserAlreadyExists):
			return nil, status.Error(codes.AlreadyExists, err.Error())
		case errors.Is(err, core.ErrInvalidEmail):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		default:
			s.log.Error("register failed", "email", req.GetEmail(), "error", err)
			return nil, status.Error(codes.Internal, "internal error")
		}
	}
	return &authpb.TokenResponse{Token: token}, nil
}

func (s *Server) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.TokenResponse, error) {
	if req.GetEmail() == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	token, err := s.service.Login(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		switch {
		case errors.Is(err, core.ErrInvalidCredentials):
			return nil, status.Error(codes.Unauthenticated, err.Error())
		case errors.Is(err, core.ErrInvalidEmail):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		default:
			s.log.Error("login failed", "email", req.GetEmail(), "error", err)
			return nil, status.Error(codes.Internal, "internal error")
		}
	}
	return &authpb.TokenResponse{Token: token}, nil
}

func (s *Server) BotLoginTelegram(ctx context.Context, req *authpb.BotLoginTelegramRequest) (*authpb.TokenResponse, error) {
	u := req.GetUser()
	if u == nil || u.GetTgId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "tg_id is required")
	}

	token, err := s.service.BotLoginTelegram(ctx, core.TelegramProfile{
		TgID:      u.GetTgId(),
		Username:  u.GetUsername(),
		FirstName: u.GetFirstName(),
		LastName:  u.GetLastName(),
	})
	if err != nil {
		s.log.Error("bot login telegram failed", "tg_id", u.GetTgId(), "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &authpb.TokenResponse{Token: token}, nil
}

func (s *Server) Ping(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if err := s.service.Ping(ctx); err != nil {
		return nil, status.Error(codes.Unavailable, err.Error())
	}
	return &emptypb.Empty{}, nil
}
