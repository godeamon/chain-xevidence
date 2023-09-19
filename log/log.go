package log

import (
	"sync"

	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/lib/logs/config"
)

var (
	Log  logs.LogDriver
	once sync.Once
)

func InitLog() {
	once.Do(
		func() {
			defaultLogConf := config.LogConf{
				Module:         "xevidence",
				Filename:       "xevidence",
				Fmt:            "logfmt",
				Level:          "debug",
				RotateInterval: 60,
				RotateBackups:  168,
				Console:        false,
				Async:          false,
				BufSize:        102400,
			}
			var err error
			Log, err = logs.OpenLog(&defaultLogConf, "./logs")
			if err != nil {
				panic(err)
			}
		})
}
