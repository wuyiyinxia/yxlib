package util

import (
	"fmt"
	"testing"
	"time"
)

func TestLog(t *testing.T) {
	Logger.SetLevel(LOG_LV_DEBUG)
	Logger.StartDump("log.txt", 1024)

	for i := 0; i < 100; i++ {
		Logger.D("test", "say hello error: unknown error")
		Logger.E("test", "hello log")

		t := time.After(time.Millisecond * 100)
		<-t
	}

	Logger.StopDump()
	fmt.Println("the log result is ok")
}
