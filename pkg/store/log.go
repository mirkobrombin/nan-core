package store

type Log interface {
	Append(rec Record) error
	Replay(fn ReplayFunc) error
	Close() error
}
