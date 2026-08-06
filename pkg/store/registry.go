package store

import "github.com/mirkobrombin/go-foundation/v2/core/adapters"

type OpenLogFunc func(path string) (Log, error)

var LogBackends = adapters.NewRegistry[OpenLogFunc]()

func init() {
	LogBackends.Register("wal", func(path string) (Log, error) { return OpenWAL(path) })
	LogBackends.Register("slipstream", func(path string) (Log, error) { return OpenSlipstreamLog(path) })
	LogBackends.SetDefault("wal")
}
