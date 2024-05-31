package main

import (
	"fmt"
	"os"
	"time"
)

import (
	"github.com/spf13/cobra"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core"
	dubbo_log "github.com/apache/dubbo-kubernetes/pkg/log"
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dubbo-dds-client",
		Short: "dubbo-dds-client",
		Long:  `dubbo-dds-client`,
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			core.SetLogger(core.NewLogger(dubbo_log.DebugLevel))
		},
	}
	cmd.AddCommand(newRunCmd())
	return cmd
}

func newRunCmd() *cobra.Command {
	log := core.Log.WithName("dubbo-dds-client").WithName("run")
	args := struct {
		ddsServerAddress string
		dps              int
		rampUpPeriod     time.Duration
	}{
		ddsServerAddress: "grpc://localhost:8888",
		dps:              10,
		rampUpPeriod:     30 * time.Second,
	}
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Start dDD client(s)",
		Long:  `Start dDS client(s)`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			log.Info("going to start dDS clients", "dps", args.dps)
			return nil
		},
	}
	// flags
	cmd.PersistentFlags().StringVar(&args.ddsServerAddress, "dds-server-address", args.ddsServerAddress, "dDS server address")
	return cmd
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
