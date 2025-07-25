package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/pyshx/fake-batch-server/pkg/handlers"
	"github.com/pyshx/fake-batch-server/pkg/storage"
)

var (
	port    int
	verbose bool
	host    string
)

var rootCmd = &cobra.Command{
	Use:   "fake-batch-server",
	Short: "A local emulator for Google Cloud Batch API",
	Long:  `Fake Batch Server provides a lightweight, in-memory implementation of the Google Cloud Batch API for local development and testing.`,
	Run:   runServer,
}

func init() {
	defaultPort := 8080
	if envPort := os.Getenv("PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			defaultPort = p
		}
	}

	defaultHost := "0.0.0.0"
	if envHost := os.Getenv("HOST"); envHost != "" {
		defaultHost = envHost
	}

	rootCmd.Flags().IntVarP(&port, "port", "p", defaultPort, "Port to run the server on")
	rootCmd.Flags().StringVarP(&host, "host", "H", defaultHost, "Host to bind the server to")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")

	if os.Getenv("VERBOSE") == "true" {
		verbose = true
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}

func runServer(cmd *cobra.Command, args []string) {
	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}

	store := storage.NewMemoryStore()
	handler := handlers.NewHandler(store)

	router := mux.NewRouter()
	router.Use(loggingMiddleware)
	router.Use(contentTypeMiddleware)

	v1 := router.PathPrefix("/v1").Subrouter()
	
	v1.HandleFunc("/projects/{project}/locations/{location}/jobs", handler.CreateJob).Methods("POST")
	v1.HandleFunc("/projects/{project}/locations/{location}/jobs", handler.ListJobs).Methods("GET")
	v1.HandleFunc("/projects/{project}/locations/{location}/jobs/{job}", handler.GetJob).Methods("GET")
	v1.HandleFunc("/projects/{project}/locations/{location}/jobs/{job}", handler.DeleteJob).Methods("DELETE")
	v1.HandleFunc("/projects/{project}/locations/{location}/jobs/{job}/tasks", handler.ListTasks).Methods("GET")
	v1.HandleFunc("/projects/{project}/locations/{location}/jobs/{job}/tasks/{task}", handler.GetTask).Methods("GET")

	v1.HandleFunc("/health", healthCheck).Methods("GET")

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logrus.Infof("Starting Fake Batch Server on %s:%d", host, port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logrus.Fatal("Server forced to shutdown:", err)
	}

	logrus.Info("Server stopped")
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logrus.WithFields(logrus.Fields{
			"method":   r.Method,
			"path":     r.URL.Path,
			"duration": time.Since(start),
		}).Debug("Request handled")
	})
}

func contentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}
