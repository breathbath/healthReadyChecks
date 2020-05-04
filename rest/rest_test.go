package rest

import (
	"context"
	"errors"
	"fmt"
	"github.com/breathbath/healthReadyChecks/errs"
	"github.com/breathbath/healthReadyChecks/health"
	"github.com/breathbath/healthReadyChecks/logging"
	"github.com/breathbath/healthReadyChecks/ready"
	"github.com/breathbath/healthReadyChecks/sleep"
	"github.com/stretchr/testify/assert"
	"log"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestHealthThreshold(t *testing.T) {
	es := errs.NewErrStream(0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	port := portToUse + 1
	go func(p int, c context.Context, e errs.ErrStream) {
		hc := health.NewErrsListener(1, time.Minute, e)
		go hc.Start(c)

		hs := WithHealth(Server{}, hc)
		err := hs.Start(c, p)
		assert.NoError(t, err)
	}(port, ctx, es)


	//give time for server to start
	time.Sleep(time.Millisecond * 500)

	es.Send(errors.New("First err"))

	addr := fmt.Sprintf("http://127.0.0.1:%d/healthz", port)
	resp, err := callAPI(addr)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	assert.Equal(t, 200, resp.StatusCode)

	es.Send(errors.New("Second err"))

	resp2, err2 := callAPI(addr)
	assert.NoError(t, err2)
	if err2 != nil {
		return
	}
	assert.Equal(t, 500, resp2.StatusCode)
}

func TestReadySuccess(t *testing.T) {
	port := portToUse + 2
	slepr := sleep.NewSleeperMock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func(c context.Context, p int, s sleep.Sleeper) {
		isError := true
		rc := ready.NewTestChecker([]ready.Test{
			{
				TestFunc: func() error {
					if isError {
						isError = false
						logging.L.WarnF("Not ready")
						return errors.New("not ready")
					}
					return nil
				},
				Name: "Some test",
			},
		}, 2, time.Second, s)
		hs := WithReady(Server{}, rc, time.Second)
		err := hs.Start(c, p)
		assert.NoError(t, err)
	}(ctx, port, slepr)

	//give time for server to start
	time.Sleep(time.Millisecond * 500)

	addr := fmt.Sprintf("http://127.0.0.1:%d/readyz", port)
	resp, err := callAPI(addr)
	assert.NoError(t, err)
	if err != nil {
		return
	}

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 1, slepr.TriggerCount)
	assert.True(t, slepr.WasTriggered)
	assert.Equal(t, time.Second, slepr.TriggeredSleepDuration)
}

func TestReadyFailure(t *testing.T) {
	port := portToUse + 3
	slepr := sleep.NewSleeperMock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func(c context.Context, p int, s sleep.Sleeper) {
		rc := ready.NewTestChecker([]ready.Test{
			{
				TestFunc: func() error {
					logging.L.WarnF("Not ready")
					return errors.New("not ready")
				},
				Name: "Some test 1",
			},
		}, 2, time.Second, s)
		hs := WithReady(Server{}, rc, time.Second)
		err := hs.Start(c, p)
		assert.NoError(t, err)
	}(ctx, port, slepr)

	//give time for server to start
	time.Sleep(time.Millisecond * 500)

	addr := fmt.Sprintf("http://127.0.0.1:%d/readyz", port)
	resp, err := callAPI(addr)
	assert.NoError(t, err)
	if err != nil {
		return
	}

	assert.Equal(t, 500, resp.StatusCode)

	assert.Equal(t, 2, slepr.TriggerCount)
	assert.True(t, slepr.WasTriggered)
	assert.Equal(t, time.Second, slepr.TriggeredSleepDuration)
}

func TestNotEquippedServer(t *testing.T) {
	port := portToUse + 4
	hs := Server{}
	err := hs.Start(context.Background(), port)
	assert.EqualError(t, err, "neither ready nor health logic was initialised")
}

func callAPI(addr string) (resp *http.Response, err error) {
	req, err := http.NewRequest(
		http.MethodGet,
		addr,
		strings.NewReader(""),
	)
	if err != nil {
		return nil, err
	}

	cl := http.Client{}
	resp, err = cl.Do(req)

	if err != nil {
		return nil, err
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	return resp, err
}
