package deaagent

import (
	"deaagent/domain"
	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/loggregatorlib/emitter"
	"github.com/howeyc/fsnotify"
	"io/ioutil"
	"path/filepath"
	"time"
)

type agent struct {
	InstancesJsonFilePath string
	logger                *gosteno.Logger
	knownInstancesChan    chan<- func(map[string]*TaskListener)
}

func NewAgent(instancesJsonFilePath string, logger *gosteno.Logger) *agent {
	knownInstancesChan := atomicCacheOperator()
	return &agent{instancesJsonFilePath, logger, knownInstancesChan}
}

func (agent *agent) Start(emitter emitter.Emitter) {
	go agent.pollInstancesJson(emitter)
}

func (agent *agent) processTasks(currentTasks map[string]domain.Task, emitter emitter.Emitter) func(knownTasks map[string]*TaskListener) {
	return func(knownTasks map[string]*TaskListener) {
		agent.logger.Debug("Reading tasks data after event on instances.json")
		agent.logger.Debugf("Current known tasks are %v", knownTasks)
		for taskIdentifier, _ := range knownTasks {
			_, present := currentTasks[taskIdentifier]
			if present {
				continue
			}
			knownTasks[taskIdentifier].StopListening()
			delete(knownTasks, taskIdentifier)
			agent.logger.Debugf("Removing stale task %v", taskIdentifier)
		}

		for _, task := range currentTasks {
			identifier := task.Identifier()
			_, present := knownTasks[identifier]
			if present {
				continue
			}
			agent.logger.Debugf("Adding new task %s", task.Identifier())
			listener := NewTaskListener(task, emitter, agent.logger)
			knownTasks[identifier] = listener

			go func() {
				defer func() {
					agent.knownInstancesChan <- removeFromCache(identifier)
				}()
				listener.StartListening()
			}()
		}
	}
}

func (agent *agent) pollInstancesJson(emitter emitter.Emitter) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}

	for {
		time.Sleep(100 * time.Millisecond)
		err := watcher.Watch(filepath.Dir(agent.InstancesJsonFilePath))
		if err != nil {
			agent.logger.Warnf("Reading failed, retrying. %s\n", err)
			continue
		}
		break
	}

	agent.logger.Info("Read initial tasks data")
	agent.readInstancesJson(emitter)

	for {
		select {
		case ev := <-watcher.Event:
			agent.logger.Debugf("Got Event: %v\n", ev)
			if ev.IsDelete() {
				agent.knownInstancesChan <- resetCache
			} else {
				agent.readInstancesJson(emitter)
			}
		case err := <-watcher.Error:
			agent.logger.Warnf("Received error from file system notification: %s\n", err)
		}
	}
}

func (agent *agent) readInstancesJson(emitter emitter.Emitter) {
	json, err := ioutil.ReadFile(agent.InstancesJsonFilePath)
	if err != nil {
		agent.logger.Warnf("Reading failed, retrying. %s\n", err)
		return
	}

	currentTasks, err := domain.ReadTasks(json)
	if err != nil {
		agent.logger.Warnf("Failed parsing json %s: %v Trying again...\n", err, string(json))
		return
	}

	agent.knownInstancesChan <- agent.processTasks(currentTasks, emitter)
}

func removeFromCache(taskId string) func(knownTasks map[string]*TaskListener) {
	return func(knownTasks map[string]*TaskListener) {
		delete(knownTasks, taskId)
	}
}

func resetCache(knownTasks map[string]*TaskListener) {
	for _, task := range knownTasks {
		task.StopListening()
	}
	knownTasks = make(map[string]*TaskListener)
}

func atomicCacheOperator() chan<- func(map[string]*TaskListener) {
	operations := make(chan func(map[string]*TaskListener))
	go func() {
		knownTasks := make(map[string]*TaskListener)
		for operation := range operations {
			operation(knownTasks)
		}
	}()
	return operations
}
