package logger

func NewStub() stubLogger {
	return stubLogger{}
}

type stubLogger struct{}

func (s stubLogger) With(label string) Logger {
	return s
}

func (s stubLogger) Debugf(format string, args ...any) {

}

func (s stubLogger) Infof(format string, args ...any) {

}

func (s stubLogger) Warnf(format string, args ...any) {

}

func (s stubLogger) Errorf(format string, args ...any) {

}

func (s stubLogger) Panicf(format string, args ...any) {

}

func (s stubLogger) Debug(err error) {

}

func (s stubLogger) Info(err error) {

}

func (s stubLogger) Warn(err error) {

}

func (s stubLogger) Error(err error) {

}

func (s stubLogger) Panic(err error) {

}
