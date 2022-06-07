package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/breathbath/healthReadyChecks/errs"
	"github.com/breathbath/healthReadyChecks/logging"
)

// ErrsListener implements health checks based on the critical amount of errors per time unit
type ErrsListener struct {
	errs                        errs.ErrStream
	unhealthyReason             string
	subsrFunc                   func(reason string)
	lock                        sync.Mutex
	maxErrsPerTime              int
	firstErrorTimestamp         int64
	currentErrorsCountPerMinute int
	timeUnit                    time.Duration
}

// NewErrsListener constructor for ErrsListener
func NewErrsListener(maxErrsPerTime int, timeUnit time.Duration, errChan errs.ErrStream) *ErrsListener {
	return &ErrsListener{
		errs:                        errChan,
		unhealthyReason:             "",
		subsrFunc:                   nil,
		lock:                        sync.Mutex{},
		maxErrsPerTime:              maxErrsPerTime,
		firstErrorTimestamp:         0,
		currentErrorsCountPerMinute: 0,
		timeUnit:                    timeUnit,
	}
}

// Start starts listening
func (l *ErrsListener) Start(ctx context.Context) {
	defer func() {
		logging.L.DebugF("Exiting health listener")
	}()
	logging.L.DebugF("Starting health listener")
	for {
		select {
		case errPayload := <-l.errs:
			l.processErrorPayload(errPayload)
		case <-ctx.Done():
			return
		}
	}
}

func (l *ErrsListener) processErrorPayload(errPayload errs.ErrPayload) {
	if errPayload.Err == nil {
		return
	}

	logging.L.WarnF("Health check registered an error '%v', will evaluate health toleration", errPayload.Err)
	l.lock.Lock()
	defer l.lock.Unlock()

	if !l.isTooManyErrors() {
		logging.L.DebugF("The amount of errors %d in the last minute is the acceptable %d", l.currentErrorsCountPerMinute, l.maxErrsPerTime)
		return
	}
	logging.L.WarnF("The amount of critical errors %d in the last minute is not the acceptable %d, will report health failure", l.currentErrorsCountPerMinute, l.maxErrsPerTime)
	l.unhealthyReason = fmt.Sprintf("Too many critical errors %d in the last minute %d, last error: %v", l.currentErrorsCountPerMinute, l.firstErrorTimestamp, errPayload.Err)
	if l.subsrFunc != nil {
		l.subsrFunc(l.unhealthyReason)
	}
}

func (l *ErrsListener) isTooManyErrors() bool {
	l.currentErrorsCountPerMinute++

	nowTimestamp := time.Now().UTC().Unix()
	secondsAmountToCheck := l.timeUnit / time.Second
	if l.currentErrorsCountPerMinute == 0 || (nowTimestamp-l.firstErrorTimestamp > int64(secondsAmountToCheck) && l.currentErrorsCountPerMinute <= l.maxErrsPerTime) {
		l.currentErrorsCountPerMinute = 1
		l.firstErrorTimestamp = nowTimestamp
		return false
	}

	return l.currentErrorsCountPerMinute > l.maxErrsPerTime
}

// IsHealthy returns health check result
func (l *ErrsListener) IsHealthy() (isHealthy bool, unhealthyReason string) {
	l.lock.Lock()
	defer l.lock.Unlock()

	return l.unhealthyReason == "", l.unhealthyReason
}

// SubscribeToUnhealthyChange accepts the callback which will be executed on unhealthy status change
func (l *ErrsListener) SubscribeToUnhealthyChange(sf func(reason string)) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.subsrFunc = sf
}
