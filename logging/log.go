package logging

import "log"

//L global logging object of the library
var L Logger

//Logger abstracts logging functionality of the library
type Logger interface {
	DebugF(template string, args ...interface{})
	ErrorF(template string, args ...interface{})
	WarnF(template string, args ...interface{})
	InfoF(template string, args ...interface{})
}

func init() {
	L = StdoutLogger{}
}

//SetLogger changes the global logging of the library
func SetLogger(l Logger) {
	L = l
}

//NullLogger pretends to log but in fact just does nothing
type NullLogger struct{}

//DebugF Logger interface implementation
func (dl NullLogger) DebugF(template string, args ...interface{}) {
}

//ErrorF Logger interface implementation
func (dl NullLogger) ErrorF(template string, args ...interface{}) {
}

//WarnF Logger interface implementation
func (dl NullLogger) WarnF(template string, args ...interface{}) {
}

//InfoF Logger interface implementation
func (dl NullLogger) InfoF(template string, args ...interface{}) {
}

//StdoutLogger logs to a standard library
type StdoutLogger struct{}

//DebugF Logger interface implementation
func (dl StdoutLogger) DebugF(template string, args ...interface{}) {
	log.Printf("DEBUG: "+template, args...)
}

//ErrorF Logger interface implementation
func (dl StdoutLogger) ErrorF(template string, args ...interface{}) {
	log.Printf("ERR: "+template, args...)
}

//WarnF Logger interface implementation
func (dl StdoutLogger) WarnF(template string, args ...interface{}) {
	log.Printf("WARN: "+template, args...)
}

//InfoF Logger interface implementation
func (dl StdoutLogger) InfoF(template string, args ...interface{}) {
	log.Printf("INFO: "+template, args...)
}
