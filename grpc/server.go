package grpc

import (
	"context"
	"github.com/breathbath/healthReadyChecks/health"
	"github.com/breathbath/healthReadyChecks/logging"
	readyProto "github.com/breathbath/healthReadyChecks/protos/go"
	"github.com/breathbath/healthReadyChecks/ready"
	"google.golang.org/grpc/codes"
	healthProto "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

//GRPCHealthName health check id
const GRPCHealthName = "grpc.health.v1.GRPCHealth"

//GRPCReadyName ready check id
const GRPCReadyName = "grpc.health.v1.GRPCReady"

//Server implements the https://github.com/grpc/grpc/blob/master/doc/health-checking.md health checking protocol
type Server struct {
	HealthChecker health.Checker
	ReadyChecker  ready.Checker
}

//Check implementation of pull model for the health status
func (s Server) Check(ctx context.Context, req *healthProto.HealthCheckRequest) (*healthProto.HealthCheckResponse, error) {
	return s.buildHealthResponse(req)
}

//Watch implementation of push model for the health status changes
func (s Server) Watch(req *healthProto.HealthCheckRequest, watcher healthProto.Health_WatchServer) error {
	//todo make GRPCHealthName dynamic if required by clients
	if req.Service != "" && req.Service != GRPCHealthName {
		return status.Error(codes.NotFound, "unknown service: "+req.Service+" expected name is "+GRPCHealthName)
	}

	s.HealthChecker.SubscribeToUnhealthyChange(func(reason string) {
		hr, err := s.buildHealthResponse(req)
		if err != nil {
			logging.L.ErrorF("VapGRPCServer was not able to send request to the stream: %v", err)
			return
		}

		err = watcher.Send(hr)
		if err != nil {
			logging.L.ErrorF("VapGRPCServer was not able to send request to the stream: %v", err)
		}
	})
	return nil
}

//Ready implementing ready test
func (s Server) Ready(ctx context.Context, req *readyProto.ReadyRequest) (*readyProto.ReadyResponse, error) {
	//todo make GRPCReadyName dynamic if required by clients
	if req.Service != "" && req.Service != GRPCReadyName {
		return nil, status.Error(codes.NotFound, "unknown service: "+req.Service+" expected name is "+GRPCReadyName)
	}
	isReady, err := s.ReadyChecker.IsReady(ctx)
	if !isReady {
		logging.L.WarnF("GRPC ready check failure: %v", err)
	}

	return &readyProto.ReadyResponse{Status: isReady}, err
}

func (s Server) buildHealthResponse(req *healthProto.HealthCheckRequest) (*healthProto.HealthCheckResponse, error) {
	if req.Service != "" && req.Service != GRPCHealthName {
		return nil, status.Error(codes.NotFound, "unknown service: "+req.Service+" expected name is "+GRPCHealthName)
	}

	isHealthy, errorExplanation := s.HealthChecker.IsHealthy()
	st := healthProto.HealthCheckResponse_SERVING
	if !isHealthy {
		logging.L.WarnF("GRPC health check failure: %s", errorExplanation)
		st = healthProto.HealthCheckResponse_NOT_SERVING
	}

	return &healthProto.HealthCheckResponse{Status: st}, nil
}
