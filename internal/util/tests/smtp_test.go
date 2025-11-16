package tests

import (
	"os"
	"testing"

	"github.com/bellezhang119/cloud-storage/internal/util"
)

func TestSendEmail_MissingEnv(t *testing.T) {
	// Clear required env vars
	os.Unsetenv("SMTP_FROM")
	os.Unsetenv("SMTP_PASSWORD")
	os.Unsetenv("SMTP_HOST")
	os.Unsetenv("SMTP_PORT")

	err := util.SendEmail("test@example.com", "Test Subject", "Test Body")
	if err == nil {
		t.Fatal("Expected error due to missing SMTP configuration, got nil")
	}
}

func TestSendEmail_InvalidHost(t *testing.T) {
	// Set env vars but with invalid SMTP_HOST to force failure in SendMail
	os.Setenv("SMTP_FROM", "from@example.com")
	os.Setenv("SMTP_PASSWORD", "password")
	os.Setenv("SMTP_HOST", "invalid.smtp.host")
	os.Setenv("SMTP_PORT", "25")

	err := util.SendEmail("to@example.com", "Test Subject", "Test Body")
	if err == nil {
		t.Fatal("Expected error due to invalid SMTP host, got nil")
	}
}
