package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
	"unicode/utf8"
	"yadro.com/course/words/words"

	"google.golang.org/protobuf/types/known/emptypb"
	wordspb "yadro.com/course/proto/words"
)

const (
	maxPhraseLen    = 4096
	maxShutdownTime = 5 * time.Second
)

type Config struct {
	Port string `yaml:"port" env:"WORDS_ADDRESS" env-default:":80"`
}

func loadConfig() (Config, error) {
	var cfg Config
	var cfgPath string

	flag.StringVar(&cfgPath, "config", "", "path to config.yaml")
	flag.Parse()

	if cfgPath != "" {
		if err := cleanenv.ReadConfig(cfgPath, &cfg); err != nil {
			return cfg, fmt.Errorf("read config file: %w", err)
		}
	}
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return cfg, fmt.Errorf("read env: %w", err)
	}
	return cfg, nil
}

type server struct {
	wordspb.UnimplementedWordsServer
	service words.Service
}

func (s *server) Ping(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	log.Printf("ping-pong:)")
	return &emptypb.Empty{}, nil
}

func (s *server) Norm(_ context.Context, in *wordspb.WordsRequest) (*wordspb.WordsReply, error) {
	phrase := in.GetPhrase()
	log.Printf("Norm start: len_runes=%d", utf8.RuneCountInString(phrase))

	// длина входной строки не больше 4kib
	if len(phrase) > maxPhraseLen {
		log.Printf("Norm too_large: len_runes=%d", utf8.RuneCountInString(phrase))
		return nil, status.Error(codes.ResourceExhausted, "phrase too large (>4KiB)")
	}

	out, err := s.service.Norm(phrase)
	if err != nil {
		log.Printf("Normalize failde: %v", err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &wordspb.WordsReply{Words: out}, nil
}

func run(cfg Config) error {
	listener, err := net.Listen("tcp", cfg.Port)
	if err != nil {
		return fmt.Errorf("failed to listen port %s: %w", cfg.Port, err)
	}

	grpcServer := grpc.NewServer()
	wordspb.RegisterWordsServer(grpcServer, &server{
		service: words.NewService(),
	})
	reflection.Register(grpcServer)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go func() {
		log.Printf("words gRPC starting %s", cfg.Port)
		if err := grpcServer.Serve(listener); err != nil {
			log.Printf("serve failed: %v", err)
			// notify shutdown if needed
			cancel()
		}
	}()

	// got failed serve or terminate signal
	<-ctx.Done()

	// force after some time
	timer := time.AfterFunc(maxShutdownTime, func() {
		log.Println("forcing server stop")
		grpcServer.Stop()
	})
	defer timer.Stop()

	log.Println("starting graceful stop")
	grpcServer.GracefulStop()
	log.Println("server stopped gracefully")
	return nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("failed to load: %v", err)
	}

	if err := run(cfg); err != nil {
		log.Fatalf("failed to run: %v", err)
	}
}
