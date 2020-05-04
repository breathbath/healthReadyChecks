package grpc

import (
	"fmt"
	readyProto "github.com/breathbath/healthReadyChecks/protos/go"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	healthProto "google.golang.org/grpc/health/grpc_health_v1"
	"net"
	"testing"
	"time"
)

func TestHealthChecker(t *testing.T) {
	hc := &healthCheckerMock{
		isHealthy: true,
	}
	s := Server{
		HealthChecker: hc,
	}

	address, baseSrv, err := startGRPC(s)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	defer baseSrv.Stop()

	err = CheckHealth(address, "some health")
	assert.NoError(t, err)
}

func TestHealthCheckerFail(t *testing.T) {
	hc := &healthCheckerMock{
		isHealthy: false,
	}
	s := Server{
		HealthChecker: hc,
	}
	address, baseSrv, err := startGRPC(s)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	defer baseSrv.Stop()

	err = CheckHealth(address, "some health")
	assert.EqualError(t, err, fmt.Sprintf(`GRPC Health client received an unhealthy status from the server some health: "%s"`, healthProto.HealthCheckResponse_NOT_SERVING))
}

func TestHealthCheckerWrongServerImplementation(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	if err != nil {
		return
	}

	baseSrv := grpc.NewServer()

	errChan := make(chan error)
	go func(l net.Listener, s *grpc.Server) {
		if er := s.Serve(l); er != nil {
			errChan <- er
		}
	}(lis, baseSrv)
	select {
	case err = <-errChan:
		assert.NoError(t, err)
	case <-time.After(time.Millisecond * 100):
	}

	err = CheckHealth(lis.Addr().String(), "some health")
	assert.EqualError(t, err, "GRPC Health server of some health does not implement the grpc health protocol")
}

func TestHealthCheckerWrongAddress(t *testing.T) {
	err := CheckHealth("some:8282", "somesrv")
	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "GRPC Health server of somesrv has failed")
	}
}

func TestReadyChecker(t *testing.T) {
	rc := &readyCheckerMock{
		isReady: true,
	}
	s := Server{
		ReadyChecker: rc,
	}

	address, baseSrv, err := startGRPC(s)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	defer baseSrv.Stop()

	err = CheckReady(address, "some ready")
	assert.NoError(t, err)
}

func TestReadyCheckerFail(t *testing.T) {
	rc := &readyCheckerMock{
		isReady: false,
	}
	s := Server{
		ReadyChecker: rc,
	}

	address, baseSrv, err := startGRPC(s)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	defer baseSrv.Stop()

	err = CheckReady(address, "some ready")
	assert.EqualError(t, err, "GRPC of some ready is not ready yet: false")
}

func TestReadyCheckerWrongAddress(t *testing.T) {
	err := CheckReady("some:8282", "somesrv")
	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "GRPC Ready server of somesrv failed")
	}
}

func TestReadyCheckerWrongServerImplementation(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	if err != nil {
		return
	}

	baseSrv := grpc.NewServer()

	errChan := make(chan error)
	go func(l net.Listener, s *grpc.Server) {
		if er := s.Serve(l); er != nil {
			errChan <- er
		}
	}(lis, baseSrv)
	select {
	case err = <-errChan:
		assert.NoError(t, err)
	case <-time.After(time.Millisecond * 100):
	}

	err = CheckReady(lis.Addr().String(), "some ready")
	assert.EqualError(t, err, "GRPC Ready server of some ready failed: rpc error: code = Unimplemented desc = unknown service readyProto.Ready")
}

func startGRPC(srv Server) (addr string, baseSrv *grpc.Server, err error) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}

	baseSrv = grpc.NewServer()
	readyProto.RegisterReadyServer(baseSrv, srv)
	healthProto.RegisterHealthServer(baseSrv, srv)

	errChan := make(chan error)
	go func(l net.Listener, s *grpc.Server) {
		if err := s.Serve(l); err != nil {
			errChan <- err
		}
	}(lis, baseSrv)

	select {
	case err := <-errChan:
		return "", nil, err
	case <-time.After(time.Millisecond * 500):
		return lis.Addr().String(), baseSrv, nil
	}
}
