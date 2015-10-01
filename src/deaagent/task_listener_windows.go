// +build windows

package deaagent

import (
	"bytes"
	"deaagent/domain"
	"errors"
	"github.com/cloudfoundry/dropsonde/logs"
	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/hpcloud/tail"
	"io"

	"fmt"
	"path/filepath"
	"strconv"
	"sync"
)

type TaskListener struct {
	*gosteno.Logger
	taskIdentifier             string
	stdOutReader, stdErrReader io.ReadCloser
	task                       domain.Task
	tail                       tail.Tail
	sync.WaitGroup
}

type buffer struct {
	bytes.Buffer
}

func (b *buffer) Close() error {
	b.Buffer.Reset()
	return nil
}

func NewTaskListener(task domain.Task, logger *gosteno.Logger) (*TaskListener, error) {

	tl := &TaskListener{
		Logger:         logger,
		taskIdentifier: task.Identifier(),
		task:           task,
	}

	stdOutReader, err := tl.dial(task.Identifier(), events.LogMessage_OUT, logger)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Connection to stdout %s failed\n", task.Identifier()))
	}

	if task.SourceName == "App" {
		stdErrReader, err := tl.dial(task.Identifier(), events.LogMessage_ERR, logger)
		if err != nil {
			stdOutReader.Close()
			return nil, errors.New(fmt.Sprintf("Connection to stderr %s failed\n", task.Identifier()))
		}
		tl.stdErrReader = stdErrReader
	}

	tl.stdOutReader = stdOutReader

	return tl, nil
}

func (tl *TaskListener) Task() domain.Task {
	return tl.task
}

func (tl *TaskListener) StartListening() {
	tl.Debugf("TaskListener.StartListening: Starting to listen to %v\n", tl.taskIdentifier)
	tl.Debugf("TaskListener.StartListening: Scanning logs for %s", tl.task.ApplicationId)

	if tl.task.SourceName == "App" {
		tl.Add(2)
		go func() {
			defer tl.Done()
			defer tl.StopListening()
			logs.ScanLogStream(tl.task.ApplicationId, tl.task.SourceName, strconv.FormatUint(tl.task.Index, 10), tl.stdOutReader)
		}()
		go func() {
			defer tl.Done()
			defer tl.StopListening()
			logs.ScanErrorLogStream(tl.task.ApplicationId, tl.task.SourceName, strconv.FormatUint(tl.task.Index, 10), tl.stdErrReader)
		}()
	} else {
		tl.Add(1)
		go func() {
			defer tl.Done()
			defer tl.StopListening()
			logs.ScanLogStream(tl.task.ApplicationId, tl.task.SourceName, strconv.FormatUint(tl.task.Index, 10), tl.stdOutReader)
		}()
	}
	tl.Wait()
}

func (tl *TaskListener) StopListening() {

	go tl.tail.Stop()

	if tl.stdOutReader != nil {
		tl.stdOutReader.Close()
	}

	if tl.stdErrReader != nil {
		tl.stdErrReader.Close()
	}

	tl.Debugf("TaskListener.StopListening: Shutting down logs for %s", tl.task.ApplicationId)
}

func (tl *TaskListener) dial(taskIdentifier string, messageType events.LogMessage_MessageType, logger *gosteno.Logger) (io.ReadCloser, error) {
	var err error
	pReader, pWriter := io.Pipe()

	go func() {

		defer pWriter.Close()

		var logFile string
		if tl.task.SourceName == "App" {
			logFile = filepath.Join(taskIdentifier, "logs", socketName(messageType))
		} else {
			logFile = filepath.Join(taskIdentifier, "staging", "tmp", "logs", "staging_task.log")
		}
		tail, err := tl.startTail(logFile, tail.Config{Follow: true, Poll: true, ReOpen: false}, messageType, taskIdentifier)
		if err != nil {
			tl.Debugf("Error while reading from socket %s, %s, %s", messageType, taskIdentifier, err)
		}

		tl.tail = *tail

		for line := range tl.tail.Lines {
			_, err := pWriter.Write([]byte(line.Text + "\n"))
			if err != nil {
				tl.Logger.Warn(err.Error())
			}
		}

	}()

	return pReader, err
}

func (tl *TaskListener) startTail(fileName string, config tail.Config, messageType events.LogMessage_MessageType, taskIdentifier string) (*tail.Tail, error) {
	var err error
	var conn *tail.Tail
	conn, err = tail.TailFile(fileName, config)
	if err != nil {
		tl.Debugf("Error while tailing file %s, %s", fileName, err)
	}
	tl.Debugf("Opened socket %s, %s", messageType, taskIdentifier)
	return conn, err
}

func socketName(messageType events.LogMessage_MessageType) string {
	if messageType == events.LogMessage_OUT {
		return "stdout.log"
	}
	return "stderr.log"
}
