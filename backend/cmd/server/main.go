package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/JaterFor/sobutylnik/backend/internal/auth"
	"github.com/JaterFor/sobutylnik/backend/internal/chat"
	"github.com/JaterFor/sobutylnik/backend/internal/config"
	"github.com/JaterFor/sobutylnik/backend/internal/db"
	"github.com/JaterFor/sobutylnik/backend/internal/feed"
	matchpkg "github.com/JaterFor/sobutylnik/backend/internal/match"
	"github.com/JaterFor/sobutylnik/backend/internal/moderation"
	"github.com/JaterFor/sobutylnik/backend/internal/profile"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	pool, err := db.NewPool(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBConnTimeout)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("db unavailable"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	authRepo := auth.NewRepo(pool)
	jwtService := auth.NewJWTService(cfg.JWTAccessSecret, cfg.JWTRefreshSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	authService := auth.NewService(authRepo, jwtService, cfg.TelegramBotToken)

	r.Route("/auth", func(ar chi.Router) {
		ar.Post("/telegram", authService.HandleTelegramLogin)
		ar.Post("/register", authService.HandleRegister)
		ar.Post("/login", authService.HandleLogin)
		ar.Post("/refresh", authService.HandleRefresh)
		ar.Post("/logout", authService.HandleLogout)
	})

	profileRepo := profile.NewRepo(pool)
	r2Storage := profile.NewR2Storage(cfg.R2AccountID, cfg.R2AccessKeyID, cfg.R2SecretAccessKey, cfg.R2Bucket, cfg.R2PublicBaseURL)
	profileService := profile.NewService(profileRepo, r2Storage)

	r.Route("/profile", func(pr chi.Router) {
		pr.Use(authService.RequireAuth)
		pr.Get("/", profileService.HandleGet)
		pr.Put("/", profileService.HandleUpsert)
		pr.Put("/location", profileService.HandleUpdateLocation)
		pr.Post("/photos/presign", profileService.HandlePresignPhoto)
		pr.Post("/photos/confirm", profileService.HandleConfirmPhoto)
		pr.Delete("/photos/{photoID}", func(w http.ResponseWriter, r *http.Request) {
			profileService.HandleDeletePhoto(w, r, chi.URLParam(r, "photoID"))
		})
	})

	matchRepo := matchpkg.NewRepo(pool)
	matchService := matchpkg.NewService(matchRepo)

	moderationRepo := moderation.NewRepo(pool, cfg.ReportBanThreshold)
	moderationService := moderation.NewService(moderationRepo, matchRepo)

	feedRepo := feed.NewRepo(pool)
	feedService := feed.NewService(feedRepo, matchRepo, nil, moderationRepo) // notifier wired in once notify package exists

	r.Route("/feed", func(fr chi.Router) {
		fr.Use(authService.RequireAuth)
		fr.Get("/", feedService.HandleGetFeed)
		fr.Post("/swipe", feedService.HandleSwipe)
	})

	r.Route("/matches", func(mr chi.Router) {
		mr.Use(authService.RequireAuth)
		mr.Get("/", matchService.HandleList)
		mr.Post("/{matchID}/unmatch", func(w http.ResponseWriter, r *http.Request) {
			matchService.HandleUnmatch(w, r, chi.URLParam(r, "matchID"))
		})
	})

	r.Route("/moderation", func(mdr chi.Router) {
		mdr.Use(authService.RequireAuth)
		mdr.Post("/report", moderationService.HandleReport)
		mdr.Post("/block", moderationService.HandleBlock)
	})

	chatRepo := chat.NewRepo(pool)
	chatHub := chat.NewHub()
	chatService := chat.NewService(chatRepo, matchRepo, moderationRepo, chatHub, jwtService)

	r.Get("/chat/ws", chatService.HandleWebSocket)
	r.Route("/chat/{matchID}", func(cr chi.Router) {
		cr.Use(authService.RequireAuth)
		cr.Get("/messages", func(w http.ResponseWriter, r *http.Request) {
			chatService.HandleHistory(w, r, chi.URLParam(r, "matchID"))
		})
		cr.Post("/messages", func(w http.ResponseWriter, r *http.Request) {
			chatService.HandleSend(w, r, chi.URLParam(r, "matchID"))
		})
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
