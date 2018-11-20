package services

import (
	"fmt"
	jiraCommunicator "git.costrategix.net/go/jira-communicator/proto/jira-communicator"
	datasource "git.costrategix.net/go/mavenlink-jira-datasource/proto/mavenlink-jira-datasource"
	"git.costrategix.net/go/mavenlink-jira-sync/functions"
	"git.costrategix.net/go/mavenlink-jira-sync/utility"
	"github.com/pkg/errors"
	"strconv"
	"time"
)

const (
	DATE_TIME_FORMAT = "2006-01-02 03:04:05"
)

type DataSourceServiceInterface interface {
	GetSyncConfiguration() ([]*datasource.ExternalProject, error)
	SaveSprintAndTaskSyncHistory(projectId int32, sprint *jiraCommunicator.SprintWithMeta) bool
	SaveIssueAndTaskSyncHistory(projectId int32, sprintId string, parentTaskId int32, taskId int32,
		issue *jiraCommunicator.Issue) bool
	UpdateIssueAndTaskSyncHistory(externalProjectId int32, project *jiraCommunicator.Project,
		issue *jiraCommunicator.IssueWithMeta, sprintId string) error
	GetJiraSprintIdFromMavenlinkTaskId(parentId int32) string
	GetMavenlinkParentTaskIdFromMavenlinkTaskId(taskId int32) int32
	GetJiraEpicKeyFromMavenlinkTaskId(taskId int32) string
	GetTaskIdsFromSprintId(sprintId string) (string, string)
	GetJiraIssueFromTaskInSubTask(projectKey string, taskInSubTask string) <-chan jiraCommunicator.Issue
	SaveWorklogAndTimeEntrySyncHistory(issueId string, worklogId string, timeentryId string, jiraUserId string,
		mavenlinkUserId string, timeLogged int64) bool
	UpdateWorklogAndTimeEntrySyncHistory(issueId string, worklogId string,
		timeEntryId string, jiraUserId string, mavenlinkUserId string, timeLogged int64) bool
}
type DataSourceService struct {
	cf functions.CommonFunctions
	jiraService JiraService
}

func (dataSourceService *DataSourceService) GetSyncConfiguration() ([]*datasource.ExternalProject, error) {
	var projectsResponse *datasource.Response
	var projects []*datasource.ExternalProject
	projectsResponse, projectsResponseErr := utility.GetUtilitiesSingleton().ConfigurationDatasource.GetAll(
		utility.GetUtilitiesSingleton().CommsContext, &datasource.Request{})
	if projectsResponseErr != nil {
		return projects, projectsResponseErr
	}
	projects = projectsResponse.Projects
	return projects, nil
}

func (dataSourceService *DataSourceService) SaveSprintAndTaskSyncHistory(projectId int32,
	sprint *jiraCommunicator.SprintWithMeta) bool {

	var saved bool
	var tasksResponse *datasource.Response
	syncedTask := datasource.ExternalTasks{}
	syncedTask.Source1SprintId = sprint.Id
	syncedTask.Type = 0
	syncedTask.DeleteFlag = 0
	syncedTask.CreatedDtTm = time.Now().Format(DATE_TIME_FORMAT)
	syncedTask.UpdatedDtTm = time.Now().Format(DATE_TIME_FORMAT)
	syncedTask.ExternalProjectId = projectId
	if sprint.MavenlinkTaskId != 0 && sprint.MavenlinkParentTaskId != 0 {
		syncedTask.Source2TaskId = sprint.MavenlinkTaskId
		syncedTask.Source2ParentTaskId = sprint.MavenlinkParentTaskId
	}
	tasksResponse, tasksResponseErr := utility.GetUtilitiesSingleton().ConfigurationDatasource.CreateTaskAndSprint(
		utility.GetUtilitiesSingleton().CommsContext, &syncedTask)
	if tasksResponseErr == nil && tasksResponse.Error == nil && tasksResponse.Task != nil {
		saved = true
	}
	return saved
}

func (dataSourceService *DataSourceService) SaveIssueAndTaskSyncHistory(projectId int32, sprintId string,
	parentTaskId int32, taskId int32, issue *jiraCommunicator.Issue) bool {

	var saved bool
	var tasksResponse *datasource.Response
	syncedTask := datasource.ExternalTasks{}
	sprintId64, sprintId64Err := strconv.ParseInt(sprintId, 10, 32)
	if nil != sprintId64Err {
		utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Cross,
			fmt.Sprintf("Error converting sprint ID: %s to int32", sprintId))
		return false
	}
	syncedTask.Source1SprintId = int32(sprintId64)
	syncedTask.Source1ParentTaskId = int32(sprintId64)
	issueId64, issueId64Err := strconv.ParseInt(issue.Id, 10, 32)
	if nil != issueId64Err {
		utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Cross,
			fmt.Sprintf("Error converting issue ID: %s to int32", issue.Id))
		return false
	}
	syncedTask.Source1TaskId = int32(issueId64)
	syncedTask.Type = 0
	syncedTask.DeleteFlag = 0
	syncedTask.CreatedDtTm = time.Now().Format(DATE_TIME_FORMAT)
	syncedTask.UpdatedDtTm = time.Now().Format(DATE_TIME_FORMAT)
	syncedTask.ExternalProjectId = projectId
	syncedTask.Source2ParentTaskId = parentTaskId
	syncedTask.Source2TaskId = taskId
	tasksResponse, tasksResponseErr := utility.GetUtilitiesSingleton().ConfigurationDatasource.CreateTaskAndIssue(
		utility.GetUtilitiesSingleton().CommsContext, &syncedTask)
	if tasksResponseErr == nil && tasksResponse.Error == nil && tasksResponse.Task != nil {
		saved = true
	}
	return saved
}

func (dataSourceService *DataSourceService) UpdateIssueAndTaskSyncHistory(externalProjectId int32,
	project *jiraCommunicator.Project, issue *jiraCommunicator.IssueWithMeta, sprintId string) error {

	var tasksResponse *datasource.Response
	existingTask := datasource.ExternalTasks{}
	existingTask.ExternalProjectId = externalProjectId
	sprintId64, sprintId64Err := strconv.ParseInt(sprintId, 10, 32)
	if nil != sprintId64Err {
		return errors.New("Failed to convert sprint ID to 32-bit integer")
	}
	existingTask.Source2TaskId = issue.MavenlinkTaskId
	existingTaskResponse, existingTaskResponseErr := utility.GetUtilitiesSingleton().ConfigurationDatasource.
		GetTaskInSubTaskFromId(utility.GetUtilitiesSingleton().CommsContext, &existingTask)
	if existingTaskResponseErr != nil || existingTaskResponse.Error != nil || existingTaskResponse.Task == nil {
		return errors.New(
			"Failed to retrieve external task information from Mavenlink task and JIRA sprint information")
	}
	parsedDate := dataSourceService.cf.ParseDateForInsertingInDb(existingTaskResponse.Task.CreatedDtTm)
	if len(parsedDate) == 0 {
		return errors.New(fmt.Sprintf(
			"Failed to parse created(%s) date to desired layout(2006-01-02 03:04:05) for update",
			existingTaskResponse.Task.CreatedDtTm))
	}
	issueId := dataSourceService.jiraService.GetJiraIssueIdFromProjectKeyAndIssueKey(project.Key, issue.ExistingIssueKey)
	if issueId == 0 {
		return errors.New("Failed to retrieve JIRA issue ID for update")
	}

	syncedTask := datasource.ExternalTasks{}
	syncedTask.Id = existingTaskResponse.Task.Id
	syncedTask.Source1SprintId = int32(sprintId64)
	syncedTask.Source1ParentTaskId = int32(sprintId64)
	syncedTask.Source1TaskId = issueId
	syncedTask.Type = 0
	syncedTask.DeleteFlag = 0
	syncedTask.CreatedDtTm = parsedDate
	syncedTask.UpdatedDtTm = time.Now().Format(DATE_TIME_FORMAT)
	syncedTask.ExternalProjectId = externalProjectId
	syncedTask.Source2ParentTaskId = issue.MavenlinkParentTaskId
	syncedTask.Source2TaskId = issue.MavenlinkTaskId
	tasksResponse, tasksResponseErr := utility.GetUtilitiesSingleton().ConfigurationDatasource.UpdateTaskAndIssue(
		utility.GetUtilitiesSingleton().CommsContext, &syncedTask)
	if tasksResponseErr != nil && tasksResponse.Error != nil {
		return errors.New(fmt.Sprintf("Failed to update external task(ID: %d)", issueId))
	}
	return nil
}

func (dataSourceService *DataSourceService) GetJiraSprintIdFromMavenlinkTaskId(parentId int32) string {
	var sprintId string
	syncedTask := datasource.ExternalTasks{}
	syncedTask.Source2TaskId = parentId
	parentTasksResponse, parentTasksResponseErr := utility.GetUtilitiesSingleton().ConfigurationDatasource.
		GetTaskIfExists(utility.GetUtilitiesSingleton().CommsContext, &syncedTask)
	if nil == parentTasksResponseErr && nil == parentTasksResponse.Error && nil != parentTasksResponse.Task {
		sprintId = fmt.Sprint(parentTasksResponse.Task.Source1SprintId)
	}
	return sprintId
}

func (dataSourceService *DataSourceService) GetMavenlinkParentTaskIdFromMavenlinkTaskId(taskId int32) int32 {
	var parentTaskId int32
	syncedTask := datasource.ExternalTasks{}
	syncedTask.Source2TaskId = taskId
	parentTasksResponse, parentTasksResponseErr := utility.GetUtilitiesSingleton().ConfigurationDatasource.
		GetTaskInSubTaskFromId(utility.GetUtilitiesSingleton().CommsContext, &syncedTask)
	if nil == parentTasksResponseErr && nil == parentTasksResponse.Error && nil != parentTasksResponse.Task {
		parentTaskId = parentTasksResponse.Task.Source2ParentTaskId
	}
	return parentTaskId
}

func (dataSourceService *DataSourceService) GetJiraEpicKeyFromMavenlinkTaskId(taskId int32) string {
	var epicKey string
	syncedTask := datasource.ExternalTasks{}
	syncedProject := datasource.ExternalProject{}
	syncedTask.Source2TaskId = taskId
	syncedTaskResponse, syncedTaskResponseErr := utility.GetUtilitiesSingleton().ConfigurationDatasource.
		GetTaskInSubTaskFromId(utility.GetUtilitiesSingleton().CommsContext, &syncedTask)
	if nil == syncedTaskResponseErr && nil == syncedTaskResponse.Error && nil != syncedTaskResponse.Task {
		syncedProject.Id = syncedTaskResponse.Task.ExternalProjectId
		syncedProjectResponse, syncedProjectResponseErr := utility.GetUtilitiesSingleton().ConfigurationDatasource.
			Get(utility.GetUtilitiesSingleton().CommsContext, &syncedProject)
		if nil == syncedProjectResponseErr && nil == syncedProjectResponse.Error &&
			nil != syncedProjectResponse.Project {

			epicKey = syncedProjectResponse.Project.ProjectKey + "-" + fmt.Sprint(syncedProjectResponse.Project.EpicId)
		}
	}
	return epicKey
}

func (dataSourceService *DataSourceService) GetTaskIdsFromSprintId(sprintId string) (string, string) {
	var taskId string
	var parentTaskId string
	syncedTask := datasource.ExternalTasks{}
	sprintId64, sprintId64Err := strconv.ParseInt(sprintId, 10, 32)
	if nil == sprintId64Err {
		syncedTask.Source1SprintId = int32(sprintId64)
		parentTasksResponse, parentTasksResponseErr := utility.GetUtilitiesSingleton().ConfigurationDatasource.
			GetSprintIfExists(utility.GetUtilitiesSingleton().CommsContext, &syncedTask)
		if nil == parentTasksResponseErr && nil == parentTasksResponse.Error && nil != parentTasksResponse.Task {
			taskId = fmt.Sprint(parentTasksResponse.Task.Source2TaskId)
			parentTaskId = fmt.Sprint(parentTasksResponse.Task.Source2ParentTaskId)
		}
	}
	return taskId, parentTaskId
}

func (dataSourceService *DataSourceService) GetJiraIssueFromTaskInSubTask(projectKey string,
	taskInSubTask string) <-chan jiraCommunicator.Issue {

	issueChannel := make(chan jiraCommunicator.Issue)
	go func() {
		syncedTask := datasource.ExternalTasks{}
		taskInSubTaskId64, taskInSubTaskId64Err := strconv.ParseInt(taskInSubTask, 10, 32)
		if nil != taskInSubTaskId64Err {
			issueChannel <- jiraCommunicator.Issue{}
		}
		syncedTask.Source2TaskId = int32(taskInSubTaskId64)
		taskInSubTaskResponse, taskInSubTaskResponseErr := utility.GetUtilitiesSingleton().ConfigurationDatasource.
			GetTaskInSubTaskFromId(utility.GetUtilitiesSingleton().CommsContext, &syncedTask)
		if nil == taskInSubTaskResponseErr && nil == taskInSubTaskResponse.Error &&
			nil != taskInSubTaskResponse && nil != taskInSubTaskResponse.Task {
			issue := dataSourceService.jiraService.RetrieveIssueInProject(projectKey,
				fmt.Sprint(taskInSubTaskResponse.Task.Source1TaskId))
			if issue != nil {
				issueChannel <- *issue
			}
		}
		issueChannel <- jiraCommunicator.Issue{}
	}()
	return issueChannel
}

func (dataSourceService *DataSourceService) SaveWorklogAndTimeEntrySyncHistory(issueId string, worklogId string,
	timeEntryId string, jiraUserId string, mavenlinkUserId string, timeLogged int64) bool {

	var saved bool
	var worklogsResponse *datasource.Response
	syncedWorklog := datasource.ExternalTimeEntries{}
	issueId64, issueId64Err := strconv.ParseInt(issueId, 10, 32)
	if nil != issueId64Err {
		return false
	}
	syncedWorklog.Source1TaskId = int32(issueId64)

	worklogId64, worklogId64Err := strconv.ParseInt(worklogId, 10, 32)
	if nil != worklogId64Err {
		return false
	}
	syncedWorklog.Source1LogId = int32(worklogId64)

	timeentryId64, timeentryId64Err := strconv.ParseInt(timeEntryId, 10, 32)
	if nil != timeentryId64Err {
		return false
	}
	syncedWorklog.Source2LogId = int32(timeentryId64)

	mavenlinkUserId64, mavenlinkUserId64Err := strconv.ParseInt(mavenlinkUserId, 10, 32)
	if nil != mavenlinkUserId64Err {
		return false
	}
	syncedWorklog.Source2UserId = int32(mavenlinkUserId64)

	syncedWorklog.Source1UserId = jiraUserId
	syncedWorklog.LoggedTime = timeLogged
	syncedWorklog.DeleteFlag = 0
	syncedWorklog.CreatedDtTm = time.Now().Format(DATE_TIME_FORMAT)
	syncedWorklog.UpdatedDtTm = time.Now().Format(DATE_TIME_FORMAT)

	worklogsResponse, tasksResponseErr := utility.GetUtilitiesSingleton().ConfigurationDatasource.CreateTimeentryAndWorklog(
		utility.GetUtilitiesSingleton().CommsContext, &syncedWorklog)
	if tasksResponseErr == nil && worklogsResponse.Error == nil && worklogsResponse.Timeentry != nil {
		saved = true
	}
	return saved
}

func (dataSourceService *DataSourceService) UpdateWorklogAndTimeEntrySyncHistory(issueId string, worklogId string,
	timeEntryId string, jiraUserId string, mavenlinkUserId string, timeLogged int64) bool {

	var saved bool
	var worklogsResponse *datasource.Response

	syncedWorklog := datasource.ExternalTimeEntries{}
	issueId64, issueId64Err := strconv.ParseInt(issueId, 10, 32)
	if nil != issueId64Err {
		return false
	}
	syncedWorklog.Source1TaskId = int32(issueId64)

	worklogId64, worklogId64Err := strconv.ParseInt(worklogId, 10, 32)
	if nil != worklogId64Err {
		return false
	}
	syncedWorklog.Source1LogId = int32(worklogId64)

	timeentryId64, timeentryId64Err := strconv.ParseInt(timeEntryId, 10, 32)
	if nil != timeentryId64Err {
		return false
	}
	syncedWorklog.Source2LogId = int32(timeentryId64)

	existingWorklog := datasource.ExternalTimeEntries{}
	existingWorklog.Source2LogId = int32(timeentryId64)
	existingWorklogsResponse, existingWorklogsResponseErr :=
		utility.GetUtilitiesSingleton().ConfigurationDatasource.GetTimeentry(
			utility.GetUtilitiesSingleton().CommsContext, &existingWorklog)
	if existingWorklogsResponseErr != nil || existingWorklogsResponse.Error != nil ||
		existingWorklogsResponse.Timeentry == nil || existingWorklogsResponse.Timeentry.Id == 0 {
		return false
	}

	mavenlinkUserId64, mavenlinkUserId64Err := strconv.ParseInt(mavenlinkUserId, 10, 32)
	if nil != mavenlinkUserId64Err {
		return false
	}
	syncedWorklog.Source2UserId = int32(mavenlinkUserId64)

	syncedWorklog.Id = existingWorklogsResponse.Timeentry.Id
	syncedWorklog.Source1UserId = jiraUserId
	syncedWorklog.LoggedTime = timeLogged
	syncedWorklog.DeleteFlag = 0
	syncedWorklog.CreatedDtTm = dataSourceService.cf.ParseDateForInsertingInDb(
		existingWorklogsResponse.Timeentry.CreatedDtTm)
	syncedWorklog.UpdatedDtTm = time.Now().Format(DATE_TIME_FORMAT)

	worklogsResponse, tasksResponseErr :=
		utility.GetUtilitiesSingleton().ConfigurationDatasource.UpdateTimeentryAndWorklog(
			utility.GetUtilitiesSingleton().CommsContext, &syncedWorklog)
	if tasksResponseErr == nil && worklogsResponse.Error == nil && worklogsResponse.Timeentry != nil {
		saved = true
	}
	return saved
}
