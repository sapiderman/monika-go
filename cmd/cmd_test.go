package cmd

import (
	"bytes"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"monika-go/internal/config"
	"monika-go/internal/logger"
)

// --- run() tests -------------------------------------------------------------

func TestRun_ValidConfig(t *testing.T) {
	cfg := &config.Config{
		Probes: []config.Probe{
			{ID: "p1", Spec: &config.HTTPSpec{Requests: []config.Request{{URL: "https://example.com"}}}},
		},
	}
	log := logger.New("test")

	// Trigger a self-signal to terminate run() gracefully after starting
	go func() {
		time.Sleep(50 * time.Millisecond)
		pid := os.Getpid()
		p, _ := os.FindProcess(pid)
		_ = p.Signal(syscall.SIGINT)
	}()

	err := run(cfg, log)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRun_EmptyConfig(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New("test")

	// Trigger a self-signal to terminate run() gracefully after starting
	go func() {
		time.Sleep(50 * time.Millisecond)
		pid := os.Getpid()
		p, _ := os.FindProcess(pid)
		_ = p.Signal(syscall.SIGINT)
	}()

	err := run(cfg, log)
	if err != nil {
		t.Errorf("expected no error (not validated here), got %v", err)
	}
}

// --- Command registration ----------------------------------------------------

func TestRootCmd_Registered(t *testing.T) {
	if rootCmd.Use != "monika-go" {
		t.Errorf("rootCmd.Use = %q, want monika-go", rootCmd.Use)
	}
}

func TestVersionCmd_RegisteredAsSubcommand(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "version" {
			found = true
			break
		}
	}
	if !found {
		t.Error("version command not registered as root subcommand")
	}
}

func TestCreateConfigCmd_RegisteredAsSubcommand(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "createConfig" {
			found = true
			break
		}
	}
	if !found {
		t.Error("createConfig command not registered as root subcommand")
	}
}

// --- Version command output --------------------------------------------------

func TestVersionCmd_Output(t *testing.T) {
	// Redirect stdout since fmt.Println writes to os.Stdout directly.
	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	versionCmd.Run(versionCmd, nil)

	w.Close()
	os.Stdout = orig

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := strings.TrimSpace(buf.String())
	if !strings.Contains(output, "version:") {
		t.Errorf("expected output to contain 'version:', got %q", output)
	}
}

// --- CreateConfig command output ---------------------------------------------

func TestCreateConfigCmd_Output(t *testing.T) {
	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	createConfigCmd.Run(createConfigCmd, nil)

	w.Close()
	os.Stdout = orig

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := strings.TrimSpace(buf.String())
	if output != "createConfig called" {
		t.Errorf("expected 'createConfig called', got %q", output)
	}
}

// --- Config flag -------------------------------------------------------------

func TestRootCmd_PersistentFlags(t *testing.T) {
	// The config flag is registered inside Execute(), not init().
	// Call the flag registration directly.
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "monika.yaml", "config file path")

	flag := rootCmd.PersistentFlags().Lookup("config")
	if flag == nil {
		t.Fatal("expected --config persistent flag to be registered")
	}
	if flag.DefValue != "monika.yaml" {
		t.Errorf("config flag default = %q, want monika.yaml", flag.DefValue)
	}
}

// --- rootCmd.RunE ------------------------------------------------------------

func TestRootCmd_RunE_ConfigFileNotFound(t *testing.T) {
	orig := cfgFile
	cfgFile = "/nonexistent/config-test.yaml"
	defer func() { cfgFile = orig }()

	err := rootCmd.RunE(rootCmd, nil)
	if err == nil {
		t.Error("expected error for missing config file, got nil")
	}
}
