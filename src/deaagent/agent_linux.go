// +build !windows

package deaagent

import (
	"deaagent/domain"
	"io/ioutil"
)

func (agent *Agent) readInstancesJson() (map[string]domain.Task, error) {

	json, err := ioutil.ReadFile(agent.InstancesJsonFilePath)

	if err != nil {
		agent.logger.Warnf("Reading failed, retrying. %s\n", err)
		return nil, err
	}

	currentTasks, err := domain.ReadTasks(json)
	if err != nil {
		agent.logger.Warnf("Failed parsing json %s: %v Trying again...\n", err, string(json))
		return nil, err
	}

	return currentTasks, nil
}
