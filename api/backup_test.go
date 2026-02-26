package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/pocketbase/pocketbase/tests"
)

func newBackupApp(t testing.TB) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("new test app: %v", err)
	}
	return app
}

func TestConfigureBackupsNoBucket(t *testing.T) {
	app := newBackupApp(t)
	defer app.Cleanup()

	getenv := func(string) string { return "" }

	if err := configureBackups(app, getenv); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if app.Settings().Backups.S3.Enabled {
		t.Fatal("S3 should stay disabled when no bucket is set")
	}
}

func TestConfigureBackupsSetsSettings(t *testing.T) {
	app := newBackupApp(t)
	defer app.Cleanup()

	env := map[string]string{
		"GCS_BACKUP_BUCKET":     "my-bucket",
		"GCS_BACKUP_REGION":     "us-central1",
		"GCS_BACKUP_ACCESS_KEY": "GOOG1234",
		"GCS_BACKUP_SECRET":     "secret-key",
	}
	getenv := func(k string) string { return env[k] }

	if err := configureBackups(app, getenv); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := app.Settings().Backups
	if s.Cron != "0 3 * * *" {
		t.Errorf("cron = %q, want %q", s.Cron, "0 3 * * *")
	}
	if s.CronMaxKeep != 7 {
		t.Errorf("cronMaxKeep = %d, want 7", s.CronMaxKeep)
	}
	if !s.S3.Enabled {
		t.Error("S3 should be enabled")
	}
	if s.S3.Bucket != "my-bucket" {
		t.Errorf("bucket = %q, want %q", s.S3.Bucket, "my-bucket")
	}
	if s.S3.Region != "us-central1" {
		t.Errorf("region = %q, want %q", s.S3.Region, "us-central1")
	}
	if s.S3.Endpoint != "https://storage.googleapis.com" {
		t.Errorf("endpoint = %q, want %q", s.S3.Endpoint, "https://storage.googleapis.com")
	}
	if s.S3.AccessKey != "GOOG1234" {
		t.Errorf("accessKey = %q, want %q", s.S3.AccessKey, "GOOG1234")
	}
	if s.S3.Secret != "secret-key" {
		t.Errorf("secret = %q, want %q", s.S3.Secret, "secret-key")
	}
	if s.S3.ForcePathStyle {
		t.Error("forcePathStyle should be false for GCS")
	}
}

func TestBackupUploadsToS3(t *testing.T) {
	var mu sync.Mutex
	var capturedPath string
	var capturedBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			mu.Lock()
			defer mu.Unlock()
			capturedPath = r.URL.Path
			capturedBody, _ = io.ReadAll(r.Body)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	app := newBackupApp(t)
	defer app.Cleanup()

	settings := app.Settings()
	settings.Backups.S3.Enabled = true
	settings.Backups.S3.Bucket = "test-backup-bucket"
	settings.Backups.S3.Region = "us-east-1"
	settings.Backups.S3.Endpoint = srv.URL
	settings.Backups.S3.AccessKey = "test-key"
	settings.Backups.S3.Secret = "test-secret"
	settings.Backups.S3.ForcePathStyle = true
	if err := app.Save(settings); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	if err := app.CreateBackup(context.Background(), "test_backup.zip"); err != nil {
		t.Fatalf("create backup: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if capturedPath != "/test-backup-bucket/test_backup.zip" {
		t.Errorf("PUT path = %q, want %q", capturedPath, "/test-backup-bucket/test_backup.zip")
	}
	if len(capturedBody) == 0 {
		t.Error("PUT body is empty, expected backup zip data")
	}
}
