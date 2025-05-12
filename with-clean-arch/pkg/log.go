package pkg

type Log interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}
