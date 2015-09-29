// +build windows

package domain

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"path/filepath"
)

func ReadTasks(dbDir string) (map[string]Task, error) {
	type instanceJson struct {
		State          string
		Droplet_id     string
		Dir            string
		Instance_index uint64
	}

	type stagingTaskJson struct {
		App_id      string
		Task_id     string
		Dir         string
		Instance_id string
	}

	var jsonApps []instanceJson
	var jsonStaging []stagingTaskJson

	applications, err := ioutil.ReadFile(filepath.Join(dbDir, "applications.json"))

	if err != nil {
		return nil, err
	}
	if len(applications) < 1 {
		return nil, errors.New("Empty data, can't parse json")
	}
	staging, err := ioutil.ReadFile(filepath.Join(dbDir, "staging.json"))

	if err != nil {
		return nil, err
	}
	if len(staging) < 1 {
		return nil, errors.New("Empty data, can't parse json")
	}

	err = json.Unmarshal(applications, &jsonApps)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(staging, &jsonStaging)
	if err != nil {
		return nil, err
	}

	tasks := make(map[string]Task, len(jsonApps))
	for _, jsonInstance := range jsonApps {
		if jsonInstance.Dir == "" {
			continue
		}
		if jsonInstance.State == "RUNNING" || jsonInstance.State == "STARTING" {
			task := Task{
				ApplicationId:       jsonInstance.Droplet_id,
				SourceName:          "App",
				WardenContainerPath: jsonInstance.Dir,
				Index:               jsonInstance.Instance_index}
			tasks[task.Identifier()] = task
		}
	}

	for _, jsonStagingTask := range jsonStaging {
		task := Task{
			ApplicationId:       jsonStagingTask.App_id,
			SourceName:          "STG",
			WardenContainerPath: jsonStagingTask.Dir}
		tasks[task.Identifier()] = task
	}

	return tasks, nil
}

func isStateTracked(state string) bool {
	return (state == "RUNNING" || state == "STARTING" || state == "STOPPING")
}
