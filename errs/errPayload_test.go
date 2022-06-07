package errs

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSendErrStream(t *testing.T) {
	errStream := NewErrStream(1)

	go func() {
		errStream.Send(errors.New("someErr"))
	}()

	ticker := time.NewTicker(500 * time.Millisecond)

	select {
	case errItem := <-errStream:
		assert.Error(t, errItem.Err)
		if errItem.Err != nil {
			assert.Equal(t, "someErr", errItem.Err.Error())
		}

		assert.True(t, errItem.Timestamp > 0)
		return
	case <-ticker.C:
		assert.Fail(t, "Timeout no error received in the error stream")
		return
	}
}
