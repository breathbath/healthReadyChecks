package errs

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestSendErrStream(t *testing.T) {
	errStream := NewErrStream(1)
	
	go func() {
		errStream.Send(errors.New("SomeErr"))
	}()

	ticker := time.NewTicker(500 * time.Millisecond)

	select {
	case errItem := <- errStream:
		assert.Error(t, errItem.Err)
		if errItem.Err != nil {
			assert.Equal(t, "SomeErr", errItem.Err.Error())
		}
		
		assert.True(t, errItem.Timestamp > 0)
		return
	case <-ticker.C:
		assert.Fail(t, "Timeout no error received in the error stream")
		return
	}
}
