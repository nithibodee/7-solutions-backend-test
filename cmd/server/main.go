// Command server starts the user management API: HTTP + gRPC servers plus a
// background goroutine that logs the user count, all with graceful shutdown.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	userv1 "github.com/nithibodee/7-solutions-backend-test/api/proto/user/v1"
	"github.com/nithibodee/7-solutions-backend-test/internal/adapter/auth"
	grpcadapter "github.com/nithibodee/7-solutions-backend-test/internal/adapter/grpc"
	httpadapter "github.com/nithibodee/7-solutions-backend-test/internal/adapter/http"
	mongoadapter "github.com/nithibodee/7-solutions-backend-test/internal/adapter/mongo"
	appuser "github.com/nithibodee/7-solutions-backend-test/internal/app/user"
	"github.com/nithibodee/7-solutions-backend-test/internal/platform/config"
	"github.com/nithibodee/7-solutions-backend-test/internal/platform/logger"
	"github.com/nithibodee/7-solutions-backend-test/internal/platform/mongodb"
)

func main() {
	log := logger.New(os.Getenv("LOG_LEVEL"))

	if err := run(log); err != nil {
		log.Error("server exited with error", slog.Any("error", err))
		os.Exit(1)
	}
	log.Info("server stopped cleanly")
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Root context cancelled on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client, db, err := mongodb.Connect(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		return err
	}
	defer func() {
		disconnectCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Disconnect(disconnectCtx)
	}()

	repo := mongoadapter.NewUserRepository(db)
	if err := repo.EnsureIndexes(ctx); err != nil {
		return err
	}

	hasher := auth.NewBcryptHasher(cfg.BcryptCost)
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTTTL)
	svc := appuser.NewService(repo, hasher, jwtManager)

	gin.SetMode(gin.ReleaseMode)
	router := httpadapter.NewRouter(httpadapter.NewHandler(svc), jwtManager, log)
	httpServer := &http.Server{
		Addr:              net.JoinHostPort("", cfg.HTTPPort),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	var grpcOpts []googlegrpc.ServerOption
	if cfg.GRPCAuth {
		grpcOpts = append(grpcOpts, googlegrpc.UnaryInterceptor(grpcadapter.AuthUnaryInterceptor(jwtManager)))
	}
	grpcServer := googlegrpc.NewServer(grpcOpts...)
	userv1.RegisterUserServiceServer(grpcServer, grpcadapter.NewServer(svc))
	reflection.Register(grpcServer)

	g, gctx := errgroup.WithContext(ctx)

	// HTTP server.
	g.Go(func() error {
		log.Info("http server listening", slog.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	// gRPC server.
	g.Go(func() error {
		lis, err := net.Listen("tcp", net.JoinHostPort("", cfg.GRPCPort))
		if err != nil {
			return err
		}
		log.Info("grpc server listening", slog.String("addr", lis.Addr().String()))
		if err := grpcServer.Serve(lis); err != nil && !errors.Is(err, googlegrpc.ErrServerStopped) {
			return err
		}
		return nil
	})

	// Background user-count logger.
	g.Go(func() error {
		appuser.RunUserCountLogger(gctx, svc, log, cfg.CountInterval)
		return nil
	})

	// Shutdown coordinator: waits for cancellation, then stops both servers.
	g.Go(func() error {
		<-gctx.Done()
		log.Info("shutdown signal received")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		grpcServer.GracefulStop()
		return httpServer.Shutdown(shutdownCtx)
	})

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
