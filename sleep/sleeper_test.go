package sleep

import (
	"testing"
	"time"
)

func TestSleep(t *testing.T) {
	s := RuntimeSleeper{}
	s.Sleep(time.Millisecond * 1)
}
