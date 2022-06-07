package errs

import (
	"time"
)

//ErrPayload info about error and the time of appearance
type ErrPayload struct {
	Err       error
	Timestamp int64
}

//ErrStream wrapper for errors stream
type ErrStream chan ErrPayload

//NewErrStream err payload chan constructor
func NewErrStream(buffer int) ErrStream {
	es := make(chan ErrPayload, buffer)
	return es
}

//Send wraps sending to err channel
func (es ErrStream) Send(err error) {
	ep := ErrPayload{
		Err:       err,
		Timestamp: time.Now().UTC().Unix(),
	}
	es <- ep
}
