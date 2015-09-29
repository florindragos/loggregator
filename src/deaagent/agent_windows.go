// +build windows

package deaagent

import (
	"deaagent/domain"
	"path/filepath"
)

func (agent *Agent) readInstancesJson() (map[string]domain.Task, error) {

	currentTasks, err := domain.ReadTasks(filepath.Dir(agent.InstancesJsonFilePath))
	if err != nil {
		agent.logger.Warnf("Failed parsing json %s. Trying again...\n", err)
		return nil, err
	}

	return currentTasks, nil
}
