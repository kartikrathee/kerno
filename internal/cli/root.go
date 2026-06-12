// Copyright 2026 Optiqor contributors
// SPDX-License-Identifier: Apache-2.0

// Package cli defines all Cobra commands for the kerno CLI.
package cli

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/optiqor/kerno/internal/config"
)

var (
	cfgFile string
	cfg     *config.Config
)

// New creates the root command and registers all sub-commands.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:   "kerno",
		Short: "Kernel-level incident diagnosis engine (eBPF)",
		Long: `Kerno asks the kernel what's wrong.

When something breaks in production — slow API, OOM, retransmits,
runqueue contention — your APM dashboard is green and the kernel
already knew minutes ago. Kerno bridges the gap.

The MVP is one command:
    sudo kerno doctor

It runs for 30 seconds, collects eBPF signals across 6 dimensions
(syscalls, TCP, OOM, disk I/O, scheduler, FDs), correlates them, and
prints a ranked diagnostic report with plain-English causes, evidence,
and copy-paste fix steps.`,
		Example: `  # The golden command — what to run when production breaks
  sudo kerno doctor

  # Quick 10s check, machine-readable, exits 1 on critical findings
  sudo kerno doctor --duration 10s --output json --exit-code

  # AI-enriched root cause analysis (needs KERNO_AI_API_KEY)
  sudo kerno doctor --ai

  # Live-stream a specific kernel dimension
  sudo kerno trace syscall --pid 1234

  # Daemon mode for Prometheus + Kubernetes DaemonSet
  sudo kerno start --prometheus`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return initConfig(cmd)
		},
	}

	// Global flags.
	pf := root.PersistentFlags()
	pf.StringVar(&cfgFile, "config", "", "config file (default: $KERNO_CONFIG, /etc/kerno/config.yaml)")
	pf.String("log-level", "info", "log verbosity: debug, info, warn, error")
	pf.String("log-format", "text", "log encoding: text (human), json (structured)")
	pf.String("output", "pretty", "output format: pretty (terminal), json (machine)")
	pf.Bool("no-color", false, "disable colored output (also honors $NO_COLOR)")

	// Group sub-commands by purpose so --help reads as a workflow,
	// not an alphabetic dump.
	doctorCmd := newDoctorCmd()
	explainCmd := newExplainCmd()
	predictCmd := newPredictCmd()
	startCmd := newStartCmd()
	traceCmd := newTraceCmd()
	watchCmd := newWatchCmd()
	chaosCmd := newChaosCmd()
	versionCmd := newVersionCmd()

	root.AddGroup(
		&cobra.Group{ID: "diagnose", Title: "Incident diagnosis:"},
		&cobra.Group{ID: "observe", Title: "Live observability:"},
		&cobra.Group{ID: "ops", Title: "Operations:"},
	)
	doctorCmd.GroupID = "diagnose"
	explainCmd.GroupID = "diagnose"
	predictCmd.GroupID = "diagnose"
	traceCmd.GroupID = "observe"
	watchCmd.GroupID = "observe"
	startCmd.GroupID = "ops"
	chaosCmd.GroupID = "ops"
	versionCmd.GroupID = "ops"

	root.AddCommand(doctorCmd, explainCmd, predictCmd, traceCmd, watchCmd, startCmd, chaosCmd, versionCmd)

	return root
}

// initConfig reads the config file and environment variables.
func initConfig(cmd *cobra.Command) error {
	v := viper.New()

	// Config file discovery.
	// Precedence: --config flag > KERNO_CONFIG env > auto-discover.
	resolved := cfgFile
	if resolved == "" {
		resolved = os.Getenv("KERNO_CONFIG")
	}
	if resolved != "" {
		v.SetConfigFile(resolved)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("/etc/kerno")
		v.AddConfigPath("$HOME/.kerno")
		v.AddConfigPath(".")
	}

	// Environment variables: KERNO_LOG_LEVEL, KERNO_PROMETHEUS_ADDR, etc.
	v.SetEnvPrefix("KERNO")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()

	// Bind CLI flags to viper.
	if err := v.BindPFlag("log_level", cmd.Root().PersistentFlags().Lookup("log-level")); err != nil {
		return fmt.Errorf("binding log-level flag: %w", err)
	}
	if err := v.BindPFlag("log_format", cmd.Root().PersistentFlags().Lookup("log-format")); err != nil {
		return fmt.Errorf("binding log-format flag: %w", err)
	}

	// Read config file (not an error if it doesn't exist).
	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			// Only error if the file was explicitly requested (flag or
			// env) and can't be parsed.
			if resolved != "" {
				return fmt.Errorf("reading config file: %w", err)
			}
		}
	}

	// Unmarshal into our typed config.
	cfg = config.Default()
	if err := v.Unmarshal(cfg); err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Initialize the global logger.
	initLogger(cfg.LogLevel, cfg.LogFormat)

	return nil
}

// initLogger configures the global slog logger.
func initLogger(level, format string) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lvl}

	var handler slog.Handler
	switch format {
	case "json":
		handler = slog.NewJSONHandler(os.Stderr, opts)
	default:
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	slog.SetDefault(slog.New(handler))
}
