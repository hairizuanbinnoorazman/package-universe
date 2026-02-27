package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/hairizuanbinnoorazman/package-universe/cmd/server/handlers"
	"github.com/hairizuanbinnoorazman/package-universe/logger"
	"github.com/hairizuanbinnoorazman/package-universe/oci"
	"github.com/hairizuanbinnoorazman/package-universe/storage"
	"github.com/spf13/cobra"
)

var configFile string

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP server",
	RunE:  runServer,
}

func init() {
	serveCmd.Flags().StringVarP(&configFile, "config", "c", "", "config file path")
	rootCmd.AddCommand(serveCmd)
}

func runServer(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load configuration
	cfg, err := LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger
	log := logger.NewLogrusLogger(cfg.Log.Level)
	log.Info(ctx, "starting server", map[string]interface{}{
		"version": Version,
		"commit":  Commit,
		"date":    BuildDate,
	})

	// Initialize storage
	storageConfig := map[string]interface{}{
		"base_dir":       cfg.Storage.BaseDir,
		"bucket":         cfg.Storage.S3Bucket,
		"region":         cfg.Storage.S3Region,
		"presign_expiry": cfg.Storage.S3PresignExpiry,
	}

	blobStorage, err := storage.NewBlobStorage(cfg.Storage.Type, storageConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Log storage initialization
	logFields := map[string]interface{}{"type": cfg.Storage.Type}
	if cfg.Storage.Type == "local" {
		logFields["base_dir"] = cfg.Storage.BaseDir
	} else if cfg.Storage.Type == "s3" {
		logFields["bucket"] = cfg.Storage.S3Bucket
		logFields["region"] = cfg.Storage.S3Region
	}
	log.Info(ctx, "storage initialized", logFields)

	// Setup router
	router := mux.NewRouter()

	// Health and readiness endpoints
	router.HandleFunc("/healthz", handlers.HealthHandler).Methods("GET")
	router.HandleFunc("/readyz", handlers.ReadyHandler).Methods("GET")

	// OCI container registry endpoints
	if cfg.Registry.Enabled {
		sessionMgr := oci.NewSessionManager(cfg.Registry.UploadSessionTimeout)
		ociStorage := oci.NewOCIStorage(blobStorage, sessionMgr)
		ociHandler := &handlers.OCIHandler{
			Storage: ociStorage,
			Logger:  log,
		}

		log.Info(ctx, "OCI container registry enabled", nil)

		// /v2/ base route
		router.HandleFunc("/v2/", ociHandler.V2Check).Methods("GET")

		// Blob upload routes (must be before blob routes since they have longer paths)
		router.HandleFunc("/v2/{name:.+}/blobs/uploads/", ociHandler.InitiateBlobUpload).Methods("POST")
		router.HandleFunc("/v2/{name:.+}/blobs/uploads/{uuid}", ociHandler.PatchBlobUpload).Methods("PATCH")
		router.HandleFunc("/v2/{name:.+}/blobs/uploads/{uuid}", ociHandler.CompleteBlobUpload).Methods("PUT")
		router.HandleFunc("/v2/{name:.+}/blobs/uploads/{uuid}", ociHandler.CancelBlobUpload).Methods("DELETE")

		// Blob routes
		router.HandleFunc("/v2/{name:.+}/blobs/{digest}", ociHandler.HeadBlob).Methods("HEAD")
		router.HandleFunc("/v2/{name:.+}/blobs/{digest}", ociHandler.GetBlob).Methods("GET")

		// Manifest routes
		router.HandleFunc("/v2/{name:.+}/manifests/{reference}", ociHandler.HeadManifest).Methods("HEAD")
		router.HandleFunc("/v2/{name:.+}/manifests/{reference}", ociHandler.GetManifest).Methods("GET")
		router.HandleFunc("/v2/{name:.+}/manifests/{reference}", ociHandler.PutManifest).Methods("PUT")

		// Tags route
		router.HandleFunc("/v2/{name:.+}/tags/list", ociHandler.TagsList).Methods("GET")
	}

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in a goroutine
	go func() {
		log.Info(ctx, "server listening", map[string]interface{}{
			"address": addr,
		})
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(ctx, "server error", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info(ctx, "shutting down server", nil)

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	log.Info(ctx, "server stopped", nil)
	return nil
}
