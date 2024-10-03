package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"

	"github.com/acronis/go-cti/internal/app/command"
	"github.com/acronis/go-cti/internal/app/commands/depcmd"
	"github.com/acronis/go-cti/internal/app/commands/deploycmd"
	"github.com/acronis/go-cti/internal/app/commands/envcmd"
	"github.com/acronis/go-cti/internal/app/commands/fmtcmd"
	"github.com/acronis/go-cti/internal/app/commands/getcmd"
	"github.com/acronis/go-cti/internal/app/commands/infocmd"
	"github.com/acronis/go-cti/internal/app/commands/initcmd"
	"github.com/acronis/go-cti/internal/app/commands/lintcmd"
	"github.com/acronis/go-cti/internal/app/commands/packcmd"
	"github.com/acronis/go-cti/internal/app/commands/restcmd"
	"github.com/acronis/go-cti/internal/app/commands/testcmd"
	"github.com/acronis/go-cti/internal/app/commands/validatecmd"
	"github.com/acronis/go-cti/internal/pkg/execx"
	"github.com/acronis/go-stacktrace"

	"github.com/dusted-go/logging/prettylog"
	"github.com/mattn/go-isatty"
	slogformatter "github.com/samber/slog-formatter"
	"github.com/spf13/cobra"
)

func initLogging(verbose bool) {
	logLvl := func() slog.Level {
		if verbose {
			return slog.LevelDebug
		}
		return slog.LevelInfo
	}()
	w := os.Stderr

	logger := slog.New(
		slogformatter.NewFormatterHandler(
			slogformatter.HTTPRequestFormatter(false),
			slogformatter.HTTPResponseFormatter(false),
			slogformatter.FormatByType(func(s []string) slog.Value {
				return slog.StringValue(strings.Join(s, ","))
			}),
		)(
			prettylog.New(&slog.HandlerOptions{Level: logLvl},
				prettylog.WithDestinationWriter(w),
				func() prettylog.Option {
					if isatty.IsTerminal(w.Fd()) {
						return prettylog.WithColor()
					}
					return func(_ *prettylog.Handler) {}
				}(),
			),
		),
	)
	slog.SetDefault(logger)
}

const (
	verboseFlag = "verbose"
)

func main() {
	os.Exit(mainFn())
}

func mainFn() int {
	var ensureDuplicates bool
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	rootCmd := func() *cobra.Command {
		cmd := &cobra.Command{
			Use:           "cti",
			Short:         "cti is a tool for managing cti projects",
			SilenceUsage:  true,
			SilenceErrors: true,
			PersistentPreRun: func(cmd *cobra.Command, _ []string) {
				verbose, err := cmd.Flags().GetBool(verboseFlag)
				if err != nil {
					fmt.Printf("Failed to get verbosity flag: %v\n", err)
					os.Exit(1)
				}

				initLogging(verbose)
			},
			CompletionOptions: cobra.CompletionOptions{
				DisableDefaultCmd: true,
			},
		}

		command.AddWorkDirFlag(cmd)

		cmd.Flags().BoolP(verboseFlag, "v", false, "verbose output")
		cmd.Flags().BoolVarP(&ensureDuplicates, "ensure-duplicates", "d", false, "ensure that there are no duplicates in tracebacks")

		cmd.AddCommand(
			packcmd.New(ctx),
			depcmd.New(ctx),
			deploycmd.New(ctx),
			envcmd.New(ctx),
			fmtcmd.New(ctx),
			getcmd.New(ctx),
			infocmd.New(ctx),
			initcmd.New(ctx),
			lintcmd.New(ctx),
			restcmd.New(ctx),
			testcmd.New(ctx),
			validatecmd.New(ctx),
			&cobra.Command{
				Use:   "version",
				Short: "print a version of tool",
				Args:  cobra.MinimumNArgs(0),
				RunE: func(_ *cobra.Command, args []string) error {
					// TODO: implement in-place solution
					return nil
				},
			},
		)
		return cmd
	}()

	if err := rootCmd.Execute(); err != nil {
		var cmdErr *command.Error
		var execError *execx.ExecutionError
		if errors.As(err, &execError) {
			slog.Error(`                ^                   `)
			slog.Error(`              / | \                 `)
			slog.Error(`                |                   `)
			slog.Error(`                |                   `)
			slog.Error(` _____  ____   ____    ___   ____   `)
			slog.Error(`| ____||  _ \ |  _ \  / _ \ |  _ \  `)
			slog.Error(`|  _|  | |_) || |_) || | | || |_) | `)
			slog.Error(`| |___ |  _ < |  _ < | |_| ||  _ <  `)
			slog.Error(`|_____||_| \_\|_| \_\ \___/ |_| \_\ `)
			slog.Error(`                |                   `)
			slog.Error(`                |                   `)
			slog.Error(`                |                   `)
		}
		if errors.As(err, &cmdErr) && cmdErr.Inner != nil {
			stOpts := func() []stacktrace.TracesOpt {
				if ensureDuplicates {
					return []stacktrace.TracesOpt{stacktrace.WithEnsureDuplicates()}
				}
				return []stacktrace.TracesOpt{}
			}()

			slog.Error("Command failed", stacktrace.ErrToSlogAttr(cmdErr.Inner, stOpts...))
		} else {
			_ = rootCmd.Usage()
		}
		return 1
	}

	return 0
}
