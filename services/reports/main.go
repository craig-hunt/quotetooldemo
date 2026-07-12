package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(os.Getenv(EnvLogLevel)),
	}))
	slog.SetDefault(log)

	dbURL := os.Getenv(EnvDatabaseURL)
	if dbURL == "" {
		log.Error(EnvDatabaseURL + " required")
		os.Exit(1)
	}
	port := os.Getenv(EnvServicePort)
	if port == "" {
		port = DefaultPort
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Error("db pool", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Error("db ping", "err", err)
		os.Exit(1)
	}

	reports := NewReports(pool)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{StatusKey: StatusOK})
	})
	mux.HandleFunc("GET /reports/quote-to-cash", func(w http.ResponseWriter, r *http.Request) {
		out, err := reports.QuoteToCash(r.Context())
		if err != nil {
			log.Error("q2c", "err", err)
			writeError(w, http.StatusInternalServerError, ErrMsgReportFailed)
			return
		}
		writeJSON(w, http.StatusOK, out)
	})
	mux.HandleFunc("GET /reports/cycle-time", func(w http.ResponseWriter, r *http.Request) {
		out, err := reports.CycleTimes(r.Context())
		if err != nil {
			log.Error("cycle", "err", err)
			writeError(w, http.StatusInternalServerError, ErrMsgReportFailed)
			return
		}
		writeJSON(w, http.StatusOK, out)
	})
	mux.HandleFunc("GET /reports/aging", func(w http.ResponseWriter, r *http.Request) {
		out, err := reports.Aging(r.Context())
		if err != nil {
			log.Error("aging", "err", err)
			writeError(w, http.StatusInternalServerError, ErrMsgReportFailed)
			return
		}
		writeJSON(w, http.StatusOK, out)
	})
	mux.HandleFunc("GET /reports/mix-breakdown", func(w http.ResponseWriter, r *http.Request) {
		out, err := reports.MixBreakdown(r.Context())
		if err != nil {
			log.Error("mix", "err", err)
			writeError(w, http.StatusInternalServerError, ErrMsgReportFailed)
			return
		}
		writeJSON(w, http.StatusOK, out)
	})

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           withCORS(withLogging(mux, log)),
		ReadHeaderTimeout: ReadHeaderTimeout,
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Info("shutdown signal received")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), ShutdownTimeout)
		defer shutdownCancel()
		_ = srv.Shutdown(shutdownCtx)
		cancel()
	}()

	log.Info("reports service starting", "port", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("server error", "err", err)
		os.Exit(1)
	}
	log.Info("reports service stopped")
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(HeaderAccessControlAllowOrigin, CORSAllowedOrigin)
		w.Header().Set(HeaderAccessControlAllowMethods, CORSAllowedMethods)
		w.Header().Set(HeaderAccessControlAllowHeaders, CORSAllowedHeaders)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withLogging(next http.Handler, log *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Info(MetricRequest,
			"method", r.Method,
			"path", r.URL.Path,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func parseLogLevel(s string) slog.Level {
	switch s {
	case LogLevelDebug:
		return slog.LevelDebug
	case LogLevelWarn:
		return slog.LevelWarn
	case LogLevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set(HeaderContentType, ContentTypeJSON)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{ErrorKey: msg})
}
