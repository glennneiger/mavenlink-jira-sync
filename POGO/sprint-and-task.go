package POGO

import (
	jira "github.com/desertjinn/jira-communicator/proto/jira-communicator"
	mavenlink "github.com/desertjinn/mavenlink-communicator/proto/mavenlink-communicator"
)

type SprintAndTaskInterface interface {
	SetTasks(tasks []mavenlink.Task)
	GetTasks() []*mavenlink.Task
	SetSubTasks(tasks []mavenlink.Task)
	GetSubTasks() []*mavenlink.Task
	SetRapidViews(sprints []jira.GreenhopperRapidView)
	GetRapidViews() []*jira.GreenhopperRapidView
	SetSprints(sprints []jira.Sprint)
	GetSprints() []*jira.Sprint
	HasValidSprintsAndTasks() bool
	GetTasksToBeProcessed(toBeCreated bool) ([]*mavenlink.Task, map[string]*jira.Sprint)
}

type SprintAndTask struct {
	tasks      []*mavenlink.Task
	subTasks   []*mavenlink.Task
	rapidViews []*jira.GreenhopperRapidView
	sprints    []*jira.Sprint
}

func (st *SprintAndTask) GetTasks() []*mavenlink.Task {
	return st.tasks
}
func (st *SprintAndTask) SetTasks(tasks []mavenlink.Task) {
	for taskKey := range tasks {
		st.tasks = append(st.tasks, &tasks[taskKey])
	}
}
func (st *SprintAndTask) GetSubTasks() []*mavenlink.Task {
	return st.subTasks
}
func (st *SprintAndTask) SetSubTasks(tasks []mavenlink.Task) {
	for taskKey := range tasks {
		st.subTasks = append(st.subTasks, &tasks[taskKey])
	}
}
func (st *SprintAndTask) GetRapidViews() []*jira.GreenhopperRapidView {
	return st.rapidViews
}
func (st *SprintAndTask) SetRapidViews(rapidViews []jira.GreenhopperRapidView) {
	for rapidViewKey := range rapidViews {
		st.rapidViews = append(st.rapidViews, &rapidViews[rapidViewKey])
	}
}
func (st *SprintAndTask) GetSprints() []*jira.Sprint {
	return st.sprints
}
func (st *SprintAndTask) SetSprints(sprints []jira.Sprint) {
	for sprintKey := range sprints {
		st.sprints = append(st.sprints, &sprints[sprintKey])
	}
}

func (st *SprintAndTask) HasValidSprintsAndTasks() bool {
	var has bool
	if st.GetTasks() != nil && st.GetSprints() != nil && len(st.GetTasks()) != 0 && len(st.GetSprints()) != 0 {
		has = true
	}
	return has
}
