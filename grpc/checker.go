package grpc

import (
	"context"
	"fmt"
	"github.com/breathbath/healthReadyChecks/logging"
	readyProto "github.com/breathbath/healthReadyChecks/protos/go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthProto "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"time"
)

//CheckHealth triggers a health check against health GRPC
func CheckHealth(addr, name string) error {
	logging.L.DebugF("Will check health of %s at %s", name, addr)

	ctx, cancel1 := context.WithTimeout(context.Background(), time.Second)
	defer cancel1()
	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()

	timeoutDuration := time.Second
	ctx2, cancel2 := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel2()

	cl := healthProto.NewHealthClient(conn)
	resp, err := cl.Check(
		ctx2,
		&healthProto.HealthCheckRequest{
			Service: GRPCHealthName,
		},
	)

	if err != nil {
		if stat, ok := status.FromError(err); ok && stat.Code() == codes.Unimplemented {
			err = fmt.Errorf("GRPC Health server of %s does not implement the grpc health protocol", name)
		} else if stat, ok := status.FromError(err); ok && stat.Code() == codes.DeadlineExceeded {
			err = fmt.Errorf("GRPC Health server of %s timeout: health rpc did not complete within %v", name, timeoutDuration)
		} else {
			err = fmt.Errorf("GRPC Health server of %s has failed: %+v", name, err)
		}
		logging.L.ErrorF(err.Error())
		return err
	}

	if resp.GetStatus() != healthProto.HealthCheckResponse_SERVING {
		logging.L.ErrorF("Health check Failure of %s, resp: %v", name, resp)
		err := fmt.Errorf("GRPC Health client received an unhealthy status from the server %s: %q", name, resp.GetStatus())
		return err
	}

	logging.L.DebugF("Health check of %s is OK, resp: %v", name, resp)
	return nil
}

//CheckReady triggers a ready check against ready GRPC
func CheckReady(addr, name string) error {
	logging.L.DebugF("Will check ready of %s at %s", name, addr)

	ctx, cancel1 := context.WithTimeout(context.Background(), time.Second)
	defer cancel1()
	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()

	timeoutDuration := time.Second
	ctx2, cancel2 := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel2()

	cl := readyProto.NewReadyClient(conn)
	resp, err := cl.Ready(
		ctx2,
		&readyProto.ReadyRequest{
			Service: GRPCReadyName,
		},
	)

	if err != nil {
		if stat, ok := status.FromError(err); ok && stat.Code() == codes.DeadlineExceeded {
			err = fmt.Errorf("GRPC Ready server %s timeout: ready rpc did not complete within %v", name, timeoutDuration)
		} else {
			err = fmt.Errorf("GRPC Ready server of %s failed: %+v", name, err)
		}
		return err
	}

	if !resp.Status {
		logging.L.ErrorF("Ready check failure of %s, resp: %v", name, resp)
		err := fmt.Errorf("GRPC of %s is not ready yet: %v", name, resp.GetStatus())
		return err
	}

	logging.L.DebugF("Ready check of %s OK, resp: %v", name, resp)
	return nil
}
