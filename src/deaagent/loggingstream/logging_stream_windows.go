// +build windows

package loggingstream

import (
	"deaagent/domain"
	"github.com/ActiveState/tail"
	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent/instrumentation"
	"github.com/cloudfoundry/loggregatorlib/logmessage"
	"path/filepath"
	"sync"
	"sync/atomic"
)

type LoggingStream struct {
	connection       *tail.Tail
	task             *domain.Task
	logger           *gosteno.Logger
	messageType      logmessage.LogMessage_MessageType
	messagesReceived uint64
	bytesReceived    uint64
	sync.Mutex
	closeChan chan struct{}
}

func NewLoggingStream(task *domain.Task, logger *gosteno.Logger, messageType logmessage.LogMessage_MessageType) (ls *LoggingStream) {
	return &LoggingStream{task: task, logger: logger, messageType: messageType, closeChan: make(chan struct{})}
}

func (ls *LoggingStream) Listen() <-chan *logmessage.LogMessage {

	messageChan := make(chan *logmessage.LogMessage, 1024)

	go func() {
		defer close(messageChan)

		logFile := filepath.Join(ls.task.Identifier(), "logs", socketName(ls.messageType))

		connection, err := ls.startTail(logFile, tail.Config{Follow: true, Poll: true})

		if err != nil {
			ls.logger.Infof("Error while reading from socket %s, %s, %s", ls.messageType, ls.task.Identifier(), err)
		}

		for line := range connection.Lines {
			select {
			case <-ls.closeChan:
				connection.Stop()
				return
			default:
			}
			messageChan <- ls.newLogMessage([]byte(line.Text))
		}
	}()

	return messageChan
}

func (ls *LoggingStream) startTail(fileName string, config tail.Config) (*tail.Tail, error) {
	var err error
	var connection *tail.Tail
	connection, err = tail.TailFile(fileName, config)
	if err != nil {
		ls.logger.Debugf("Error while tailing file %s, %s", fileName, err)
	}
	ls.logger.Debugf("Opened socket %s, %s", ls.messageType, ls.task.Identifier())
	return connection, err
}

func (ls *LoggingStream) Emit() instrumentation.Context {
	return instrumentation.Context{Name: "loggingStream:" + ls.task.WardenContainerPath + " type " + socketName(ls.messageType),
		Metrics: []instrumentation.Metric{
			instrumentation.Metric{Name: "receivedMessageCount", Value: atomic.LoadUint64(&ls.messagesReceived)},
			instrumentation.Metric{Name: "receivedByteCount", Value: atomic.LoadUint64(&ls.bytesReceived)},
		},
	}
}

func socketName(messageType logmessage.LogMessage_MessageType) string {
	if messageType == logmessage.LogMessage_OUT {
		return "stdout.log"
	}
	return "stderr.log"
}
