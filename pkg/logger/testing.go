package logger

type EmptyLogger struct{}

func (*EmptyLogger) Debug(_ any, _ ...any)            {}
func (*EmptyLogger) Info(_ any, _ ...any)             {}
func (*EmptyLogger) Warn(_ any, _ ...any)             {}
func (*EmptyLogger) Error(_ any, _ ...any)            {}
func (*EmptyLogger) Panic(_ any, _ ...any)            {}
func (*EmptyLogger) Fatal(_ any, _ ...any)            {}
func (el *EmptyLogger) With(_ Field, _ any) Interface { return el }
