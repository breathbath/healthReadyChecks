package sleep

import (
	"github.com/breathbath/healthz/logging"
	"time"
)

//Sleeper abstract sleeping logic
type Sleeper interface {
	Sleep(t time.Duration)
}

//RuntimeSleeper real sleeper implementation
type RuntimeSleeper struct{}

//Sleep implements sleeping logic
func (rs RuntimeSleeper) Sleep(t time.Duration) {
	logging.L.InfoF("Will sleep %v", t)
	time.Sleep(t)
	logging.L.InfoF("Woke up, will continue working")
}
