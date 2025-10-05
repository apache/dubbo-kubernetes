package xds

import (
	"errors"
	"github.com/apache/dubbo-kubernetes/pkg/xds"
	dubbogrpc "github.com/apache/dubbo-kubernetes/sail/pkg/grpc"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
	v3 "github.com/apache/dubbo-kubernetes/sail/pkg/xds/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

func (s *DiscoveryServer) StreamDeltas(stream DeltaDiscoveryStream) error {
	if !s.IsServerReady() {
		return errors.New("server is not ready to serve discovery information")
	}

	ctx := stream.Context()
	peerAddr := "0.0.0.0"
	if peerInfo, ok := peer.FromContext(ctx); ok {
		peerAddr = peerInfo.Addr.String()
	}

	if err := s.WaitForRequestLimit(stream.Context()); err != nil {
		klog.Warningf("ADS: %q exceeded rate limit: %v", peerAddr, err)
		return status.Errorf(codes.ResourceExhausted, "request rate limit exceeded: %v", err)
	}

	ids, err := s.authenticate(ctx)
	if err != nil {
		return status.Error(codes.Unauthenticated, err.Error())
	}
	if ids != nil {
		klog.V(2).Infof("Authenticated XDS: %v with identity %v", peerAddr, ids)
	} else {
		klog.V(2).Infof("Unauthenticated XDS: %v", peerAddr)
	}

	// InitContext returns immediately if the context was already initialized.
	s.globalPushContext().InitContext(s.Env, nil, nil)
	con := newDeltaConnection(peerAddr, stream)

	// Do not call: defer close(con.pushChannel). The push channel will be garbage collected
	// when the connection is no longer used. Closing the channel can cause subtle race conditions
	// with push. According to the spec: "It's only necessary to close a channel when it is important
	// to tell the receiving goroutines that all data have been sent."

	// Block until either a request is received or a push is triggered.
	// We need 2 go routines because 'read' blocks in Recv().
	go s.receiveDelta(con, ids)

	// Wait for the proxy to be fully initialized before we start serving traffic. Because
	// initialization doesn't have dependencies that will block, there is no need to add any timeout
	// here. Prior to this explicit wait, we were implicitly waiting by receive() not sending to
	// reqChannel and the connection not being enqueued for pushes to pushChannel until the
	// initialization is complete.
	<-con.InitializedCh()

	for {
		// Go select{} statements are not ordered; the same channel can be chosen many times.
		// For requests, these are higher priority (client may be blocked on startup until these are done)
		// and often very cheap to handle (simple ACK), so we check it first.
		select {
		case req, ok := <-con.deltaReqChan:
			if ok {
				if err := s.processDeltaRequest(req, con); err != nil {
					return err
				}
			} else {
				// Remote side closed connection or error processing the request.
				return <-con.ErrorCh()
			}
		case <-con.StopCh():
			return nil
		default:
		}
		// If there wasn't already a request, poll for requests and pushes. Note: if we have a huge
		// amount of incoming requests, we may still send some pushes, as we do not `continue` above;
		// however, requests will be handled ~2x as much as pushes. This ensures a wave of requests
		// cannot completely starve pushes. However, this scenario is unlikely.
		select {
		case req, ok := <-con.deltaReqChan:
			if ok {
				if err := s.processDeltaRequest(req, con); err != nil {
					return err
				}
			} else {
				// Remote side closed connection or error processing the request.
				return <-con.ErrorCh()
			}
		case ev := <-con.PushCh():
			pushEv := ev.(*Event)
			err := s.pushConnectionDelta(con, pushEv)
			pushEv.done()
			if err != nil {
				return err
			}
		case <-con.StopCh():
			return nil
		}
	}
}

func (s *DiscoveryServer) receiveDelta(con *Connection, identities []string) {
	defer func() {
		close(con.deltaReqChan)
		close(con.ErrorCh())
		// Close the initialized channel, if its not already closed, to prevent blocking the stream
		select {
		case <-con.InitializedCh():
		default:
			close(con.InitializedCh())
		}
	}()
	firstRequest := true
	for {
		req, err := con.deltaStream.Recv()
		if err != nil {
			if dubbogrpc.GRPCErrorType(err) != dubbogrpc.UnexpectedError {
				klog.Infof("ADS: %q %s terminated", con.Peer(), con.ID())
				return
			}
			con.ErrorCh() <- err
			klog.Errorf("ADS: %q %s terminated with error: %v", con.Peer(), con.ID(), err)
			return
		}
		// This should be only set for the first request. The node id may not be set - for example malicious clients.
		if firstRequest {
			// probe happens before envoy sends first xDS request
			if req.TypeUrl == v3.HealthInfoType {
				klog.Warningf("ADS: %q %s send health check probe before normal xDS request", con.Peer(), con.ID())
				continue
			}
			firstRequest = false
			if req.Node == nil || req.Node.Id == "" {
				con.ErrorCh() <- status.New(codes.InvalidArgument, "missing node information").Err()
				return
			}
			if err := s.initConnection(req.Node, con, identities); err != nil {
				con.ErrorCh() <- err
				return
			}
			defer s.closeConnection(con)
			klog.Infof("ADS: new delta connection for node:%s", con.ID())
		}

		select {
		case con.deltaReqChan <- req:
		case <-con.deltaStream.Context().Done():
			klog.Infof("ADS: %q %s terminated with stream closed", con.Peer(), con.ID())
			return
		}
	}
}

func (s *DiscoveryServer) pushConnectionDelta(con *Connection, pushEv *Event) error {
	pushRequest := pushEv.pushRequest

	if pushRequest.Full {
		// Update Proxy with current information.
		s.computeProxyState(con.proxy, pushRequest)
	}

	pushRequest, needsPush := s.ProxyNeedsPush(con.proxy, pushRequest)
	if !needsPush {
		klog.V(2).Infof("Skipping push to %v, no updates required", con.ID())
		return nil
	}

	// Send pushes to all generators
	// Each Generator is responsible for determining if the push event requires a push
	wrl := con.watchedResourcesByOrder()
	for _, w := range wrl {
		if err := s.pushDeltaXds(con, w, pushRequest); err != nil {
			return err
		}
	}

	return nil
}

func (s *DiscoveryServer) processDeltaRequest(req *discovery.DeltaDiscoveryRequest, con *Connection) error {
	return nil
}

func (s *DiscoveryServer) computeProxyState(proxy *model.Proxy, request *model.PushRequest) {
	return
}

func (s *DiscoveryServer) pushDeltaXds(con *Connection, w *model.WatchedResource, req *model.PushRequest) error {
	return nil
}

func newDeltaConnection(peerAddr string, stream DeltaDiscoveryStream) *Connection {
	return &Connection{
		Connection:   xds.NewConnection(peerAddr, nil),
		deltaStream:  stream,
		deltaReqChan: make(chan *discovery.DeltaDiscoveryRequest, 1),
	}
}
