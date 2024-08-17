package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"strconv"
)

import (
	"google.golang.org/grpc"
)

import (
	"github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
)

var port = flag.Int("port", 5678, "The port to use to connect to the mesh")
var ip = flag.String("ip", "127.0.0.1", "The IP to use to connect to the mesh")
var servPort = flag.Int("servPort", 30000, "test service port")

func main() {

	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		conn, err := grpc.Dial(*ip+":"+strconv.Itoa(*port), grpc.WithInsecure())
		if err != nil {
			panic(err)
		}
		cli := v1alpha1.NewMDSSyncServiceClient(conn)
		req := &v1alpha1.MetaDataRegisterRequest{
			Namespace: r.Header["namespace"][0],
			PodName:   r.Header["podname"][0],
			Metadata: &v1alpha1.MetaData{
				Zone:     "",
				App:      "",
				Revision: "",
				Services: nil,
			},
		}
		logger.Debugf("[send] req: %s , err: %v", req.String(), err)
		resp, err := cli.MetadataRegister(context.Background(), req)
		logger.Debugf("[recv] rep: %s , err: %v", resp.String(), err)
	})

	// 启动HTTP服务器，在端口8080上监听

	fmt.Println("Server is listening on port " + strconv.Itoa(*servPort) + "...")
	if err := http.ListenAndServe(":"+strconv.Itoa(*servPort), nil); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
	}

}
