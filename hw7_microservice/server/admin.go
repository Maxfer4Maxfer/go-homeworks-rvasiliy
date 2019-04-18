package server

import (
	"fmt"
	"sync"
	"time"

	"../protobuf"

	"github.com/maxfer4maxfer/goDebuger"
)

type AdminServer struct {
	mutex         *sync.RWMutex
	loggers       *[]*logReciver
	statGatherers *[]*statReciver
}

func NewAdminServer(l *[]*logReciver, sg *[]*statReciver, m *sync.RWMutex) *AdminServer {
	return &AdminServer{
		mutex:         m,
		loggers:       l,
		statGatherers: sg,
	}
}

func (as *AdminServer) Logging(n *protobuf.Nothing, logStream protobuf.Admin_LoggingServer) error {
	if DEBUG {
		fmt.Println(goDebuger.GetCurrentFunctionName())
	}

	// create and register new logger
	lr := &logReciver{
		log:   make(chan *logMessage),
		close: make(chan struct{}),
	}

	as.mutex.Lock()
	*as.loggers = append(*as.loggers, lr)
	as.mutex.Unlock()

	for {
		select {
		case logMsg := <-lr.log:
			out := &protobuf.Event{
				Timestamp: logMsg.Timestamp,
				Consumer:  logMsg.Consumer,
				Method:    logMsg.Method,
				Host:      logMsg.Host,
			}
			logStream.Send(out)
		case <-lr.close:
			return nil
		}
	}
}

func (as *AdminServer) Statistics(si *protobuf.StatInterval, statStream protobuf.Admin_StatisticsServer) error {
	if DEBUG {
		fmt.Println(goDebuger.GetCurrentFunctionName())
	}

	// create and register new stat gatherer
	sr := &statReciver{
		stat:  make(chan *statMessage),
		close: make(chan struct{}),
	}

	as.mutex.Lock()
	*as.statGatherers = append(*as.statGatherers, sr)
	as.mutex.Unlock()

	ticker := time.NewTicker(time.Duration(si.IntervalSeconds) * time.Second)
	tickChan := ticker.C

	stat := &protobuf.Stat{
		Timestamp:  0,
		ByMethod:   make(map[string]uint64),
		ByConsumer: make(map[string]uint64),
	}

	for {
		select {
		case statMsg := <-sr.stat:
			// gather statistic
			stat.ByMethod[statMsg.Method]++
			stat.ByConsumer[statMsg.Consumer]++
		case <-tickChan:
			// sent statistic
			stat.Timestamp = time.Now().UnixNano()
			statStream.Send(stat)
			// restart statistic
			stat = &protobuf.Stat{
				Timestamp:  0,
				ByMethod:   make(map[string]uint64),
				ByConsumer: make(map[string]uint64),
			}
		case <-sr.close:
			ticker.Stop()
			return nil
		}
	}
}
