package loggingstream

import (
	"code.google.com/p/gogoprotobuf/proto"
	"fmt"
	"github.com/cloudfoundry/loggregatorlib/logmessage"
	"strconv"
	"time"
)

func (ls *LoggingStream) Stop() {
	ls.Lock()
	defer ls.Unlock()

	select {
	case <-ls.closeChan:
	default:
		close(ls.closeChan)
	}

	if ls.connection != nil {
		ls.connection.Stop()
	}
	ls.logger.Infof("Stopped reading from socket %s, %s", ls.messageType, ls.task.Identifier())
}

func (ls *LoggingStream) newLogMessage(message []byte) *logmessage.LogMessage {
	currentTime := time.Now()
	sourceName := ls.task.SourceName
	sourceId := strconv.FormatUint(ls.task.Index, 10)
	messageCopy := make([]byte, len(message))
	copyCount := copy(messageCopy, message)
	if copyCount != len(message) {
		panic(fmt.Sprintf("Didn't copy the message %d, %s", copyCount, message))
	}
	return &logmessage.LogMessage{
		Message:     messageCopy,
		AppId:       proto.String(ls.task.ApplicationId),
		DrainUrls:   ls.task.DrainUrls,
		MessageType: &ls.messageType,
		SourceName:  &sourceName,
		SourceId:    &sourceId,
		Timestamp:   proto.Int64(currentTime.UnixNano()),
	}
}
