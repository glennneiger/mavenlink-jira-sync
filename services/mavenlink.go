package services

import (
	"fmt"
	communicator "github.com/desertjinn/mavenlink-communicator/proto/mavenlink-communicator"
	"github.com/desertjinn/mavenlink-jira-sync/utility"
	"strings"
)

type MavenlinkServiceInterface interface {
	DoesWorkspaceExistInMavenlink(keyOrId int32, exists chan bool)
	RetrieveTasksInWorkspaceWithTitle(keyOrId int32, tasks chan []communicator.Task, title string)
	RetrieveSubTasksInWorkspace(keyOrId int32, taskKeyOrId int32, tasks chan []communicator.Task)
	RetrieveTasksFromSubTasksInWorkspace(keyOrId int32, subTaskKeyOrId int32, tasks chan []communicator.Task)
	GetTimeEntriesForIssueTask(workspaceKeyOrId int32, taskKeyOrId string, timeEntries chan []communicator.Timeentry)
}
type MavenlinkService struct {}

// Check if the workspace exists in Mavenlink
func (mavenlinkService *MavenlinkService) DoesWorkspaceExistInMavenlink(keyOrId int32, exists chan bool) {
	var does bool
	var mavenlinkProjectsResponse *communicator.Response
	var projectExistsRequest communicator.Request
	projectExistsRequest.Workspace = fmt.Sprint(keyOrId)
	mavenlinkProjectsResponse, err := utility.GetUtilitiesSingleton().MavenlinkClient.GetProjectById(
		utility.GetUtilitiesSingleton().CommsContext, &projectExistsRequest)
	if err == nil && mavenlinkProjectsResponse.Error == nil &&
		mavenlinkProjectsResponse != nil &&
		mavenlinkProjectsResponse.Project != nil {
		does = true
	}
	exists <- does
}

// Retrieve the milestone task with desired title from the workspace in Mavenlink
func (mavenlinkService *MavenlinkService) RetrieveTasksInWorkspaceWithTitle(keyOrId int32,
	tasks chan []communicator.Task, title string) {

	var mavenlinkTasksResponse *communicator.Response
	var mavenlinkTasks []communicator.Task
	var taskListRequest communicator.Request
	taskListRequest.Workspace = fmt.Sprint(keyOrId)
	mavenlinkTasksResponse, err := utility.GetUtilitiesSingleton().MavenlinkClient.GetTasksByProjectId(
		utility.GetUtilitiesSingleton().CommsContext, &taskListRequest)
	if err == nil && mavenlinkTasksResponse.Error == nil &&
		mavenlinkTasksResponse != nil &&
		mavenlinkTasksResponse.Tasks != nil {
		for _, task := range mavenlinkTasksResponse.Tasks {
			if strings.EqualFold(task.Title, title) {
				mavenlinkTasks = append(mavenlinkTasks, *task)
			}
		}
	}
	tasks <- mavenlinkTasks
}

// Retrieve the all sub-tasks in milestone task from the workspace in Mavenlink
func (mavenlinkService *MavenlinkService) RetrieveSubTasksInWorkspace(keyOrId int32, taskKeyOrId int32,
	tasks chan []communicator.Task) {

	var subTasksResponse *communicator.Response
	var taskListRequest communicator.Request
	var subTasks []communicator.Task
	taskListRequest.Workspace = fmt.Sprint(keyOrId)
	taskListRequest.Task = fmt.Sprint(taskKeyOrId)
	subTasksResponse, err := utility.GetUtilitiesSingleton().MavenlinkClient.GetSubTasksByParentTaskAndProjectId(
		utility.GetUtilitiesSingleton().CommsContext, &taskListRequest)
	if err == nil && subTasksResponse.Error == nil &&
		subTasksResponse != nil &&
		subTasksResponse.Tasks != nil {
		for _, subTask := range subTasksResponse.Tasks {
			subTasks = append(subTasks, *subTask)
		}
	}
	tasks <- subTasks
}

// Retrieve the all the tasks in sub-tasks from the workspace in Mavenlink
func (mavenlinkService *MavenlinkService) RetrieveTasksFromSubTasksInWorkspace(keyOrId int32, subTaskKeyOrId int32,
	tasks chan []communicator.Task) {
	var tasksInSubTasksResponse *communicator.Response
	var tasksInSubTaskListRequest communicator.Request
	var tasksInSubTask []communicator.Task
	tasksInSubTaskListRequest.Workspace = fmt.Sprint(keyOrId)
	tasksInSubTaskListRequest.SubTask = fmt.Sprint(subTaskKeyOrId)
	tasksInSubTasksResponse, err := utility.GetUtilitiesSingleton().MavenlinkClient.GetTasksBySubTaskParentTaskAndProjectId(
		utility.GetUtilitiesSingleton().CommsContext, &tasksInSubTaskListRequest)
	if err == nil && tasksInSubTasksResponse.Error == nil &&
		tasksInSubTasksResponse != nil &&
		tasksInSubTasksResponse.Tasks != nil {
		for _, taskInSubTask := range tasksInSubTasksResponse.Tasks {
			tasksInSubTask = append(tasksInSubTask, *taskInSubTask)
		}
	}
	tasks <- tasksInSubTask
}

func (mavenlinkService *MavenlinkService) GetTimeEntriesForIssueTask(workspaceKeyOrId int32, taskKeyOrId string,
	timeEntries chan []communicator.Timeentry) {

	var timeentriesResponse *communicator.Response
	var timeentriesRequest communicator.Request
	var accumulatedTimeentries []communicator.Timeentry
	timeentriesRequest.Workspace = fmt.Sprint(workspaceKeyOrId)
	timeentriesRequest.Task = taskKeyOrId
	timeentriesResponse, err := utility.GetUtilitiesSingleton().MavenlinkClient.GetTimeentries(
		utility.GetUtilitiesSingleton().CommsContext, &timeentriesRequest)
	if err == nil && timeentriesResponse.Error == nil &&
		timeentriesResponse != nil &&
		timeentriesResponse.Timeentries != nil {
		for _, timeentry := range timeentriesResponse.Timeentries {
			accumulatedTimeentries = append(accumulatedTimeentries, *timeentry)
		}
	}
	timeEntries <- accumulatedTimeentries
}
