package main

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/tools/dds-client/stream"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"os"
)

import (
	"github.com/spf13/cobra"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core"
	dubbo_log "github.com/apache/dubbo-kubernetes/pkg/log"
)

var mdsLog = core.Log.WithName("md")

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dubbo-mds-client",
		Short: "dubbo-mds-client",
		Long:  `dubbo-mds-client`,
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			core.SetLogger(core.NewLogger(dubbo_log.DebugLevel))
		},
	}
	cmd.AddCommand(newRunCmd())
	return cmd
}

func newRunCmd() *cobra.Command {
	args := struct {
		ddsServerAddress string
	}{
		ddsServerAddress: "grpc://localhost:8888",
	}
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Start dDD client(s)",
		Long:  `Start dDS client(s)`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := stream.New(args.ddsServerAddress)

			// register mapping and metadata
			client.MappingRegister()
			client.MetadataRegister()

			if err != nil {
				return errors.Wrap(err, "failed to connect to xDS server")
			}
			defer func() {
				mdsLog.Info("closing a connection ...")
				if err := client.Close(); err != nil {
					return
				}
			}()

			mdsLog.Info("opening a dds stream ...")
			mappingStream, err := client.StartMappingStream()
			if err != nil {
				return errors.Wrap(err, "failed to start an mDS stream")
			}
			defer func() {
				mdsLog.Info("closing a mds mapping steam ... ")
				if err := mappingStream.Close(); err != nil {
					return
				}
			}()

			metadataStream, err := client.StartMetadataSteam()
			if err != nil {
				return errors.Wrap(err, "failed to start an mDS stream")
			}
			defer func() {
				mdsLog.Info("closing a mds metadata stream ... ")
				if err := metadataStream.Close(); err != nil {
					return
				}
			}()

			// mapping and metadata request
			mappingStream.MappingSyncRequest()

			metadataStream.MetadataSyncRequest()

			var eg errgroup.Group

			eg.Go(func() error {
				for {
					mdsLog.Info("waiting for a mapping response ...")
					resp, err := mappingStream.WaitForMappingResource()
					if err != nil {
						return errors.Wrap(err, "failed to receive a mapping response")
					}

					mdsLog.Info("recv mapping", resp)

					if err := mappingStream.MappingACK(); err != nil {
						return errors.Wrap(err, "failed to ACK a mapping response")
					}
				}
			})

			eg.Go(func() error {
				for {
					mdsLog.Info("waiting for a metadata response ...")
					resp, err := metadataStream.WaitForMetadataResource()
					if err != nil {
						return errors.Wrap(err, "failed to receive a metadata response")
					}

					mdsLog.Info("recv metadata", resp)

					if err := metadataStream.MetadataACK(); err != nil {
						return errors.Wrap(err, "failed to ACK a metadata response")
					}
				}
			})

			err = eg.Wait()
			if err != nil {
				return err
			}

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
