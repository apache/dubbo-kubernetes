package kube

import "k8s.io/client-go/tools/clientcmd"

type clientFactory struct {
	clientConfig clientcmd.ClientConfig
}

func newClientFactory(clientConfig clientcmd.ClientConfig, diskCache bool) *clientFactory {
	cf := &clientFactory{clientConfig: clientConfig}
	return cf
}
