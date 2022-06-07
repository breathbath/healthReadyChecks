package health

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/breathbath/healthReadyChecks/errs"
	"github.com/stretchr/testify/assert"
)

func TestLessAndManyErrors(t *testing.T) {
	errStream := errs.NewErrStream(0)
	lis := NewErrsListener(1, time.Minute, errStream)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()

	go lis.Start(ctx)

	assert.Equal(t, "", lis.unhealthyReason)

	errStream.Send(errors.New("some err1"))

	isHealthy, unhealthyReason := lis.IsHealthy()
	assert.True(t, isHealthy)
	assert.Equal(t, "", unhealthyReason)

	errStream.Send(errors.New("some err2"))

	<-ctx.Done()

	isHealthy, unhealthyReason = lis.IsHealthy()
	assert.False(t, isHealthy)
	assert.Equal(t, fmt.Sprintf("Too many critical errors 2 in the last minute %d, last error: some err2", lis.firstErrorTimestamp), unhealthyReason)
}

func TestSendingEmptyErrors(t *testing.T) {
	errStream := errs.NewErrStream(0)
	l := NewErrsListener(1, time.Minute, errStream)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()

	receiveErrsChan := make(chan string, 3)
	l.SubscribeToUnhealthyChange(func(reason string) {
		receiveErrsChan <- reason
	})

	go l.Start(ctx)
	go func() {
		errStream.Send(nil)
		errStream.Send(errors.New("Some err3"))
		errStream.Send(errors.New("Some err4"))
	}()

	eventStream := consumeErrStream(ctx, receiveErrsChan)

	<-ctx.Done()

	expectedErrText := fmt.Sprintf("Too many critical errors 2 in the last minute %d, last error: Some err4", l.firstErrorTimestamp)
	assert.Equal(t, []string{expectedErrText}, eventStream)
}

func consumeErrStream(ctx context.Context, sourceStream chan string) []string {
	eventStream := make([]string, 0)

	for {
		select {
		case event := <-sourceStream:
			eventStream = append(eventStream, event)
		case <-ctx.Done():
			return eventStream
		}
	}
}
