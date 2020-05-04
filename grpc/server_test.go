package grpc

import (
	"context"
	"errors"
	readyProto "github.com/breathbath/healthReadyChecks/protos/go"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	healthProto "google.golang.org/grpc/health/grpc_health_v1"
	"testing"
)

type healthCheckerMock struct {
	isHealthy bool
	isHealthyReason string
	subscrF func(reason string)
}

//IsHealthy health.Checker implementation
func (hcm healthCheckerMock) IsHealthy() (bool, string) {
	return hcm.isHealthy, hcm.isHealthyReason
}

//SubscribeToUnhealthyChange health.Checker implementation
func (hcm *healthCheckerMock) SubscribeToUnhealthyChange(sf func(reason string)) {
	hcm.subscrF = sf
}

type readyCheckerMock struct {
	isReady bool
	err error
}

//IsReady ready.Checker implementation
func (rcm readyCheckerMock) IsReady(ctx context.Context) (isReady bool, err error) {
	return rcm.isReady, rcm.err
}

type watchServer struct {
	err error
	resps []*healthProto.HealthCheckResponse
	grpc.ServerStream
}

//Send healthProto.Health_WatchServer interface implementation
func (ws *watchServer) Send(resp *healthProto.HealthCheckResponse) error {
	ws.resps = append(ws.resps, resp)

	return ws.err
}

func TestUnknownServerName(t *testing.T) {
	s := Server{}
	_, err := s.Ready(context.Background(), &readyProto.ReadyRequest{
		Service: "some unknown server",
	})
	assert.EqualError(t, err, "rpc error: code = NotFound desc = unknown service: some unknown server expected name is "+GRPCReadyName)

	err2 := s.Watch(&healthProto.HealthCheckRequest{
		Service: "some unknown server",
	}, nil)
	assert.EqualError(t, err2, "rpc error: code = NotFound desc = unknown service: some unknown server expected name is "+GRPCHealthName)

	_, err3 := s.Check(context.Background(), &healthProto.HealthCheckRequest{
		Service: "some unknown server",
	})
	assert.EqualError(t, err3, "rpc error: code = NotFound desc = unknown service: some unknown server expected name is "+GRPCHealthName)
}

func TestHealthSuccess(t *testing.T) {
	hc := &healthCheckerMock{
		isHealthy: true,
	}
	s := Server{
		HealthChecker: hc,
	}

	resp, err := s.Check(context.Background(), &healthProto.HealthCheckRequest{
		Service: GRPCHealthName,
	})

	assert.NoError(t, err)
	if err != nil {
		return
	}

	assert.Equal(t, healthProto.HealthCheckResponse_SERVING, resp.Status)
}

func TestHealthFailure(t *testing.T) {
	hc := &healthCheckerMock{
		isHealthy: false,
	}
	s := Server{
		HealthChecker: hc,
	}

	resp, err := s.Check(context.Background(), &healthProto.HealthCheckRequest{
		Service: GRPCHealthName,
	})

	assert.NoError(t, err)
	if err != nil {
		return
	}

	assert.Equal(t, healthProto.HealthCheckResponse_NOT_SERVING, resp.Status)
}

func TestReadySuccess(t *testing.T) {
	rcm := readyCheckerMock{
		isReady: true,
		err: nil,
	}
	s := Server{
		ReadyChecker: rcm,
	}

	resp, err := s.Ready(context.Background(), &readyProto.ReadyRequest{
		Service: GRPCReadyName,
	})

	assert.NoError(t, err)
	if err != nil {
		return
	}

	assert.True(t, resp.Status)
}

func TestNotReady(t *testing.T) {
	rcm := readyCheckerMock{
		isReady: false,
		err: nil,
	}
	s := Server{
		ReadyChecker: rcm,
	}

	resp, err := s.Ready(context.Background(), &readyProto.ReadyRequest{
		Service: GRPCReadyName,
	})

	assert.NoError(t, err)
	if err != nil {
		return
	}

	assert.False(t, resp.Status)
}

func TestReadyError(t *testing.T) {
	rcm := readyCheckerMock{
		isReady: false,
		err: errors.New("ready failure"),
	}
	s := Server{
		ReadyChecker: rcm,
	}

	_, err := s.Ready(context.Background(), &readyProto.ReadyRequest{
		Service: GRPCReadyName,
	})

	assert.EqualError(t, err, "ready failure")
}

func TestWatch(t *testing.T) {
	hc := &healthCheckerMock{
		isHealthy: false,
	}
	s := Server{
		HealthChecker: hc,
	}

	watchSrv := &watchServer{
		resps: []*healthProto.HealthCheckResponse{},
	}

	err := s.Watch(&healthProto.HealthCheckRequest{
		Service: GRPCHealthName,
	}, watchSrv)

	assert.Len(t, watchSrv.resps, 0)

	assert.NoError(t, err)
	if err != nil {
		return
	}

	assert.NotNil(t, hc.subscrF)
	if hc.subscrF == nil {
		return
	}

	hc.subscrF("some bad reason")
	assert.Len(t, watchSrv.resps, 1)
	healthRespFromChecker := watchSrv.resps[0]
	assert.Equal(t, healthProto.HealthCheckResponse_NOT_SERVING, healthRespFromChecker.Status)
}
