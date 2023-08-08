package signals

import (
	"os"
	"os/signal"
)

var onlyOneSignalHandler = make(chan struct{})

// SetupSignalHandler 注册信号量
func SetupSignalHandler() (stopChan <-chan struct{}) {
	close(onlyOneSignalHandler)
	stop := make(chan struct{})
	c := make(chan os.Signal, 2)
	// 监听信号量
	signal.Notify(c, shutdownSignals...)
	go func() {
		// 接收信号量
		<-c
		close(stop)

		// 再次接收信号量
		<-c
		os.Exit(1) // second signal. Exit directly.
	}()

	return stop
}
