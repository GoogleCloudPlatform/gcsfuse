package prefetch

import "github.com/googlecloudplatform/gcsfuse/v2/internal/cache/util"

type Configuration struct {
	FirstBufferSize              uint32
	MaxBufferSize                uint32
	SequentialPrefetchMultiplier uint32

	MaxForwardSeekWaitDistance  uint32
	MaxBackwardSeekWaitDistance uint32
}

func GetDefaultPrefetchConfiguration() *Configuration {
	return &Configuration{
		FirstBufferSize:              util.MiB + 128*1024,
		MaxBufferSize:                50 * util.MiB,
		SequentialPrefetchMultiplier: 5,
		MaxForwardSeekWaitDistance:   16 * util.MiB,
		MaxBackwardSeekWaitDistance:  util.MiB,
	}
}
