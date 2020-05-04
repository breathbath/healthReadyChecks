package ready

import (
	"context"
	"errors"
	"fmt"
	"github.com/breathbath/healthz/logging"
	"github.com/breathbath/healthz/sleep"
	"strings"
	"sync"
	"time"
)

//Checker abstracts readiness check behavior
type Checker interface {
	IsReady(ctx context.Context) (isReady bool, err error)
}

//Test will wrap readiness func
type Test struct {
	TestFunc func() error
	Name string
}

type result struct {
	test    Test
	isReady bool
	err     error
}

//TestChecker ready checks are based on the []Test collection where tests are run in parallel
type TestChecker struct {
	tests []Test
	maxRetries int
	sleepInterval time.Duration
	sleeper sleep.Sleeper
}

//NewTestChecker constructor, will try maxRetries and sleep sleepInterval with the sleep.Sleeper before failing ready check
func NewTestChecker(tests []Test, maxRetries int, sleepInterval time.Duration, sleeper sleep.Sleeper) TestChecker {
	return TestChecker{tests: tests, maxRetries: maxRetries, sleepInterval: sleepInterval, sleeper: sleeper}
}

//IsReady readiness implementation
func (rc TestChecker) IsReady(ctx context.Context) (isReady bool, err error) {
	logging.L.DebugF("Will execute ready scripts")
	wg := &sync.WaitGroup{}
	wg.Add(len(rc.tests))

	resultChan := make(chan result)

	for _, test := range rc.tests {
		go rc.checkTest(test, wg, resultChan)
	}

	allDone := make(chan bool)
	go func() {
		wg.Wait()
		allDone <- true
	}()

	errs := make([]string, 0, len(rc.tests))
	for {
		select {
		case <- ctx.Done():
			return false, errors.New("ready tests failed due to the context timeout")
		case res := <- resultChan:
			if !res.isReady {
				errs = append(errs, fmt.Sprintf("Readiness probe failed for %s: %v", res.test.Name, res.err))
			}
		case <- allDone:
			if len(errs) == 0 {
				return true, nil
			}
			return false, errors.New(strings.Join(errs, ", "))
		}
	}
}

func (rc TestChecker) checkTest(test Test, wg *sync.WaitGroup, resultChan chan result) {
	defer wg.Done()

	var errToGive error
	for i := 0; i < rc.maxRetries; i++ {
		logging.L.DebugF("Will check if %s is ready, attempt %d", test.Name, i+1)
		err := test.TestFunc()
		if err == nil {
			logging.L.DebugF("%s is ready", test.Name)
			resultChan <- result{test: test, isReady: true, err: nil}
			return
		}

		errToGive = err

		if rc.maxRetries > 1 {
			rc.sleeper.Sleep(rc.sleepInterval)
		}

		logging.L.WarnF("%s is not ready: %v", test.Name, err)
	}

	resultChan <- result{test: test, isReady: false, err: errToGive}
}
