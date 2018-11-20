package functions

import (
	"fmt"
	jiraCommunicator "github.com/desertjinn/jira-communicator/proto/jira-communicator"
	mavenlinkCommunicator "github.com/desertjinn/mavenlink-communicator/proto/mavenlink-communicator"
	datasourceCommunicator "github.com/desertjinn/mavenlink-jira-datasource/proto/mavenlink-jira-datasource"
	"github.com/desertjinn/mavenlink-jira-sync/POGO"
	"github.com/desertjinn/mavenlink-jira-sync/utility"
	"strconv"
	"strings"
)

type SprintFunctionsInterface interface {
	GetTasksToBeProcessed(subTasks []*mavenlinkCommunicator.Task, jiraSprints []*jiraCommunicator.Sprint,
		toBeCreated bool) ([]*mavenlinkCommunicator.Task,
		map[string]*jiraCommunicator.Sprint)
	PrepareSprintsForCreation(sprintsAndTasks *POGO.SprintAndTask) (<-chan jiraCommunicator.SprintWithMeta, <-chan bool)
	PrepareSprintsForUpdate(sprintsAndTasks *POGO.SprintAndTask) (<-chan jiraCommunicator.SprintWithMeta, <-chan bool)
}

type SprintFunctions struct {
	cf CommonFunctions
}

// Check if JIRA sprint and Mavenlink task combination exists in the datasource
func doesSprintAndTaskExistInDatasource(task int32, sprint int32) bool {
	var does bool
	taskAndSprint := &datasourceCommunicator.ExternalTasks{}
	taskAndSprint.Source1SprintId = sprint
	taskAndSprint.Source2TaskId = task
	taskAndSprintResponse, taskAndSprintResponseErr :=
		utility.GetUtilitiesSingleton().ConfigurationDatasource.GetTaskAndSprint(utility.GetUtilitiesSingleton().CommsContext,
			taskAndSprint)
	if taskAndSprintResponseErr == nil && taskAndSprintResponse.Error == nil && taskAndSprintResponse.Task != nil {
		if taskAndSprintResponse.Task.Id != 0 {
			does = true
		}
	}
	return does
}

// Check if Mavenlink task exists in the datasource
func doesTaskExistInDatasource(task int32) *datasourceCommunicator.ExternalTasks {
	taskAndSprint := &datasourceCommunicator.ExternalTasks{}
	taskAndSprint.Source2TaskId = task
	taskAndSprintResponse, taskAndSprintResponseErr :=
		utility.GetUtilitiesSingleton().ConfigurationDatasource.GetTaskIfExists(utility.GetUtilitiesSingleton().CommsContext,
			taskAndSprint)
	if taskAndSprintResponseErr == nil && taskAndSprintResponse.Error == nil && taskAndSprintResponse.Task != nil {
		if taskAndSprintResponse.Task.Id != 0 {
			return taskAndSprintResponse.Task
		}
	}
	return nil
}

// Get the JIRA sprint related to the Mavenlink task
func getMatchingSprintForTask(sprints []*jiraCommunicator.Sprint,
	task *mavenlinkCommunicator.Task) *jiraCommunicator.Sprint {

	var taskId int32
	taskId64, taskIdErr := strconv.ParseInt(task.Id, 10, 32)
	if taskIdErr != nil {
		return nil
	}
	taskId = int32(taskId64)
	taskInDb := doesTaskExistInDatasource(taskId)
	if nil != taskInDb {
		for _, sprint := range sprints {
			if sprint.Id == taskInDb.Source1SprintId {
				exists := doesSprintAndTaskExistInDatasource(taskId, sprint.Id)
				if exists == true {
					return sprint
				}
			}
		}
	}
	return nil
}

func (self *SprintFunctions) prepSprint(task *mavenlinkCommunicator.Task, rapidView string, sprintId int32) *jiraCommunicator.SprintWithMeta {
	sprint := new(jiraCommunicator.SprintWithMeta)
	if sprintId > 0 {
		sprint.Id = sprintId
	}
	sprint.Name = task.Title
	sprint.StartDate = self.cf.ParseMavenlinkDateToJiraDate(task.StartDate, "")
	if task.StartDate == task.DueDate {
		sprint.EndDate = self.cf.ParseMavenlinkDateToJiraDate(task.DueDate, "01:00:00")
	} else {
		sprint.EndDate = self.cf.ParseMavenlinkDateToJiraDate(task.DueDate, "")
	}
	sprint.RapidView = rapidView
	taskIdInt64, taskIdInt64Err := strconv.ParseInt(task.Id, 10, 32)
	if taskIdInt64Err == nil {
		sprint.MavenlinkTaskId = int32(taskIdInt64)
	}
	parentTaskIdInt64, parentTaskIdInt64Err := strconv.ParseInt(task.ParentId, 10, 32)
	if parentTaskIdInt64Err == nil {
		sprint.MavenlinkParentTaskId = int32(parentTaskIdInt64)
	}
	return sprint
}



// Get the Mavenlink tasks to be processed as JIRA sprints
func (self *SprintFunctions) GetTasksToBeProcessed(subTasks []*mavenlinkCommunicator.Task, jiraSprints []*jiraCommunicator.Sprint,
	toBeCreated bool) ([]*mavenlinkCommunicator.Task,
	map[string]*jiraCommunicator.Sprint) {

	var tasks []*mavenlinkCommunicator.Task
	sprints := map[string]*jiraCommunicator.Sprint{}
	for _, task := range subTasks {
		sprint := getMatchingSprintForTask(jiraSprints, task)
		if toBeCreated == true {
			if sprint == nil {
				tasks = append(tasks, task)
			}
		} else {
			if sprint != nil {
				tasks = append(tasks, task)
				sprints[task.Id] = sprint
			}
		}
	}
	return tasks, sprints
}

// Prepare Mavenlink sub-tasks as JIRA sprints for creation purposes
func (self *SprintFunctions) PrepareSprintsForCreation(sprintsAndTasks *POGO.SprintAndTask) (
	<-chan jiraCommunicator.SprintWithMeta, <-chan bool) {

	sprintsChannel := make(chan jiraCommunicator.SprintWithMeta)
	sprintsChannelClosed := make(chan bool)
	go func() {
		toBeCreated, _ := self.GetTasksToBeProcessed(sprintsAndTasks.GetSubTasks(), sprintsAndTasks.GetSprints(),
			true)
		if toBeCreated != nil {
			for _, toBe := range toBeCreated {
				sprint := self.prepSprint(toBe, fmt.Sprint(sprintsAndTasks.GetRapidViews()[0].Id), 0)
				if sprint != nil {
					sprintsChannel <- *sprint
				}
			}
		}
		sprintsChannelClosed <- true
	}()
	return sprintsChannel, sprintsChannelClosed
}

// Prepare Mavenlink sub-tasks as existing JIRA sprints for update purposes
func (self *SprintFunctions) PrepareSprintsForUpdate(sprintsAndTasks *POGO.SprintAndTask) (
	<-chan jiraCommunicator.SprintWithMeta, <-chan bool) {

	sprintsChannel := make(chan jiraCommunicator.SprintWithMeta)
	sprintsChannelClosed := make(chan bool)
	go func() {
		toBeSynced, relatedSprints := self.GetTasksToBeProcessed(sprintsAndTasks.GetSubTasks(), sprintsAndTasks.GetSprints(),
			false)
		if toBeSynced != nil {
			for _, task := range toBeSynced {
				relatedSprint := relatedSprints[task.Id]
				if relatedSprint == nil {
					continue
				}
				if !strings.EqualFold(task.Title, relatedSprint.Name) ||
					!strings.EqualFold(task.StartDate, self.cf.ParseJiraDateToMavenlinkDate(relatedSprint.StartDate)) ||
					!strings.EqualFold(task.DueDate, self.cf.ParseJiraDateToMavenlinkDate(relatedSprint.EndDate)) {
					sprint := self.prepSprint(task, fmt.Sprint(sprintsAndTasks.GetRapidViews()[0].Id), relatedSprint.Id)
					if sprint != nil {
						sprintsChannel <- *sprint
					}
				}
			}
		}
		sprintsChannelClosed <- true
	}()
	return sprintsChannel, sprintsChannelClosed
}
