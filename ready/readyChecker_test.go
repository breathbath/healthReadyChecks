package ready

import (
	"context"
	"errors"
	"github.com/breathbath/healthReadyChecks/sleep"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNoCheckers(t *testing.T) {
	checker := NewTestChecker([]Test{}, 10, time.Millisecond, sleep.NewSleeperMock())
	isReady, err := checker.IsReady(context.Background())
	assert.True(t, isReady)
	assert.Nil(t, err)
}

func TestCheckerSuccess(t *testing.T) {
	sleeper := sleep.NewSleeperMock()
	checker := NewTestChecker([]Test{
		{
			TestFunc: func() error {
				return nil
			},
			Name: "TestCheckerSuccess",
		},
	},
		10,
		time.Millisecond,
		sleeper,
	)

	isReady, err := checker.IsReady(context.Background())
	assert.True(t, isReady)
	assert.Nil(t, err)

	assert.False(t, sleeper.WasTriggered)
}

func TestCheckerFailure(t *testing.T) {
	sleeper := sleep.NewSleeperMock()
	checker := NewTestChecker([]Test{
		{
			TestFunc: func() error {
				return errors.New("some error")
			},
			Name: "TestCheckerFailure",
		},
	},
		1,
		time.Millisecond,
		sleeper,
	)

	isReady, err := checker.IsReady(context.Background())
	assert.False(t, isReady)
	assert.EqualError(t, err, "Readiness probe failed for TestCheckerFailure: some error")
	assert.False(t, sleeper.WasTriggered)
}

func TestMultiCheckerFailure(t *testing.T) {
	sleeper := sleep.NewSleeperMock()

	checker := NewTestChecker([]Test{
		{
			TestFunc: func() error {
				return errors.New("some error 1")
			},
			Name: "TestMultiCheckerFailure1",
		},
		{
			TestFunc: func() error {
				return errors.New("some error 2")
			},
			Name: "TestMultiCheckerFailure2",
		},
	},
		1,
		time.Millisecond,
		sleeper,
	)

	isReady, err := checker.IsReady(context.Background())
	assert.False(t, isReady)
	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "Readiness probe failed for TestMultiCheckerFailure1: some error 1")
		assert.Contains(t, err.Error(), "Readiness probe failed for TestMultiCheckerFailure2: some error 2")
	}
	assert.False(t, sleeper.WasTriggered)
}

func TestMultiCheckerFailureAndSuccess(t *testing.T) {
	checker := NewTestChecker([]Test{
		{
			TestFunc: func() error {
				return errors.New("Some error 1")
			},
			Name: "TestMultiCheckerFailureAndSuccess1",
		},
		{
			TestFunc: func() error {
				return nil
			},
			Name: "TestMultiCheckerFailureAndSuccess2",
		},
	},
		1,
		time.Millisecond,
		sleep.NewSleeperMock(),
	)

	isReady, err := checker.IsReady(context.Background())
	assert.False(t, isReady)
	assert.EqualError(t, err, "Readiness probe failed for TestMultiCheckerFailureAndSuccess1: Some error 1")
}

func TestSleepAndRepeatOnFailure(t *testing.T) {
	sleeper := sleep.NewSleeperMock()
	attempts := make(chan int, 2)
	checker := NewTestChecker([]Test{
		{
			TestFunc: func() error {
				attempts <- 1
				return errors.New("Some error 234")
			},
			Name: "TestSleepAndRepeatOnFailure",
		},
	},
		2,
		time.Second,
		sleeper,
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	isReady, err := checker.IsReady(ctx)

	assert.EqualError(t, err, "Readiness probe failed for TestSleepAndRepeatOnFailure: Some error 234")
	if err != nil {
		return
	}
	assert.Len(t, attempts, 2)
	assert.False(t, isReady)
	assert.True(t, sleeper.WasTriggered)
	assert.Equal(t, time.Second, sleeper.TriggeredSleepDuration)
	assert.Equal(t, 2, sleeper.TriggerCount)
}

func TestWaitingTimeout(t *testing.T) {
	checker := NewTestChecker([]Test{
		{
			TestFunc: func() error {
				return errors.New("Some error 1")
			},
			Name: "TestMultiCheckerFailureAndSuccess1",
		},
	},
		1,
		time.Millisecond,
		sleep.NewSleeperMock(),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	isReady, err := checker.IsReady(ctx)
	assert.False(t, isReady)
	assert.EqualError(t, err, "ready tests failed due to the context timeout")
}
