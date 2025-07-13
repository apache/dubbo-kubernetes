package main

import (
	"context"
	"fmt"
	"os"

	"github.com/apache/dubbo-kubernetes/api/legacy"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	dubbo_log "github.com/apache/dubbo-kubernetes/pkg/log"
	"github.com/apache/dubbo-kubernetes/tools/dds-client/stream"
)

var mdsLog = core.Log.WithName("mds")

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
			if err != nil {
				return errors.Wrap(err, "failed to connect to xDS server")
			}
			defer func() {
				mdsLog.Info("closing a connection ...")
				if err := client.Close(); err != nil {
					return
				}
			}()

			ctx := context.Background()

			// register mapping and metadata
			err = client.MappingRegister(ctx, &legacy.MappingRegisterRequest{
				Namespace:       "dubbo-system",
				ApplicationName: "test",
				InterfaceNames:  []string{"a1", "a2"},
				PodName:         os.Getenv("POD_NAME"),
			})
			if err != nil {
				return err
			}
			err = client.MetadataRegister(ctx, &legacy.MetaDataRegisterRequest{
				Namespace: "dubbo-system",
				PodName:   os.Getenv("POD_NAME"),
				Metadata: &mesh_proto.MetaData{
					App:      "test",
					Revision: "11111",
				},
			})
			if err != nil {
				return err
			}

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
			err = mappingStream.MappingSyncRequest(&legacy.MappingSyncRequest{
				Namespace:     "",
				Nonce:         "",
				InterfaceName: "",
			})
			if err != nil {
				return err
			}

			err = metadataStream.MetadataSyncRequest(&legacy.MetadataSyncRequest{
				Namespace:       "",
				Nonce:           "",
				ApplicationName: "",
				Revision:        "",
			})
			if err != nil {
				return err
			}

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
