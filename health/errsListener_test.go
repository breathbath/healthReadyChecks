package health

import (
	"context"
	"errors"
	"fmt"
	"github.com/breathbath/healthz/errs"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestLessAndManyErrors(t *testing.T) {
	errStream := errs.NewErrStream(0)
	lis := NewErrsListener(1, time.Minute, errStream)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond * 100)
	defer cancel()

	go lis.Start(ctx)

	assert.Equal(t, "", lis.unhealthyReason)

	errStream.Send(errors.New("Some err1"))

	isHealthy, unhealthyReason := lis.IsHealthy()
	assert.True(t, isHealthy)
	assert.Equal(t, "", unhealthyReason)

	errStream.Send(errors.New("Some err2"))

	<-ctx.Done()

	isHealthy, unhealthyReason = lis.IsHealthy()
	assert.False(t, isHealthy)
	assert.Equal(t, fmt.Sprintf("Too many critical errors 2 in the last minute %d, last error: Some err2", lis.firstErrorTimestamp), unhealthyReason)
}

func TestSendingEmptyErrors(t *testing.T) {
	errStream := errs.NewErrStream(0)
	l := NewErrsListener(1, time.Minute, errStream)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond * 10)
	defer cancel()

	go l.Start(ctx)

	errStream.Send(nil)
	errStream.Send(nil)

	<-ctx.Done()

	assert.Equal(t, "", l.unhealthyReason)
}

func TestHealthSubscription(t *testing.T) {
	errStream := errs.NewErrStream(0)
	l := NewErrsListener(1, time.Minute, errStream)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond * 100)
	defer cancel()
	go l.Start(ctx)

	eventStream := []string{}
	l.SubscribeToUnhealthyChange(func(reason string) {
		eventStream = append(eventStream, reason)
	})

	errStream.Send(nil)

	errStream.Send(errors.New("Some err3"))
	assert.Equal(t, []string{}, eventStream)

	errStream.Send(errors.New("Some err4"))

	<-ctx.Done()

	expectedErrText := fmt.Sprintf("Too many critical errors 2 in the last minute %d, last error: Some err4", l.firstErrorTimestamp)
	assert.Equal(t, []string{expectedErrText}, eventStream)
}
