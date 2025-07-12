package main

import (
	"context"
	"log"
	"os"
	"runtime/pprof"

	"github.com/sirprodigle/linkpatrol/internal/app"
	"github.com/sirprodigle/linkpatrol/internal/config"
	"github.com/spf13/cobra"
)

var cfg config.Config

var rootCmd = &cobra.Command{
	Use:           "linkpatrol",
	Short:         "Markdown/HTML link checker",
	Long:          `LinkPatrol is a tool for checking that links in markdown/html files are accessible and valid`,
	RunE:          run,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	cfg = config.NewConfig()
	cfg.InitFlags(rootCmd)
}

func run(cmd *cobra.Command, args []string) error {
	cfg.LoadFromViper()

	if cfg.CPUProfile != "" {
		f, err := os.Create(cfg.CPUProfile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal(err)
		}
		defer pprof.StopCPUProfile()
	}

	if cfg.MemProfile != "" {
		f, err := os.Create(cfg.MemProfile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal(err)
		}
	}

	application := app.New(&cfg)
	err := application.Run(context.Background())

	return err
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
