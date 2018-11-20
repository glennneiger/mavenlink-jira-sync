package services

import (
	"fmt"
	communicator "github.com/desertjinn/jira-communicator/proto/jira-communicator"
	"github.com/desertjinn/mavenlink-jira-sync/utility"
	"github.com/pkg/errors"
	"strconv"
)

type JiraServiceInterface interface {
	GetJiraProject(projectId int32) *communicator.Project
	CreateSprintInJira(rapidViewId string) *communicator.Sprint
	CreateIssueInJira(issue *communicator.IssueCreate) *communicator.Issue
	CreateWorklogInJira(issueKey string, worklog *communicator.WorklogWithMeta) *communicator.Worklog
	UpdateWorklogInJira(issueKey string, worklog *communicator.WorklogWithMeta) *communicator.Worklog
	UpdateIssueInJira(issue *communicator.IssueCreate) bool
	UpdateSprintInJira(sprint *communicator.SprintWithMeta) error
	DoesProjectExistInJira(projectId int32, exists chan bool)
	DoesEpicExistInJiraProject(epicKey string, exists chan bool)
	GetEpicInJiraProject(epicKey string) *communicator.Issue
	RetrieveRapidViewsInProject(projectKey string, views chan []communicator.GreenhopperRapidView)
	RetrieveSprintsInProject(projectKey string, sprints chan []communicator.Sprint)
	RetrieveIssuesFromSprintInProject(projectKey string, sprintName string, issues chan []communicator.Issue)
	RetrieveIssueInProject(projectKey string, issueId string) *communicator.Issue
	UpdateSprintInfoForJiraIssue(sprintId string, issueKey string) bool
	UpdateEpicInfoForJiraIssue(epicKey string, issueKey string) bool
	GetJiraIssueIdFromProjectKeyAndIssueKey(projectKey string, issueKey string) int32
	GetWorklogsFromIssue(issueKey string, worklogs chan []communicator.Worklog)
	GetUsersInProject(projectName string, users chan []communicator.Author)
	GetJiraStatusMetadata(projectId string, statuses chan []communicator.Status)
	GetJiraPriorityMetadata(projectId string, priorities chan []communicator.Priority)
}
type JiraService struct {}

func (jiraService *JiraService) GetJiraProject(projectId int32) *communicator.Project {
	var project *communicator.Project
	var jiraProjectResponse *communicator.Response
	var projectRequest communicator.Request
	projectRequest.Project = fmt.Sprint(projectId)
	jiraProjectResponse, err := utility.GetUtilitiesSingleton().JiraClient.GetProject(
		utility.GetUtilitiesSingleton().CommsContext, &projectRequest)
	if err == nil && jiraProjectResponse.Error == nil && jiraProjectResponse.Project != nil {
		project = jiraProjectResponse.Project
	}
	return project
}

func (jiraService *JiraService) CreateSprintInJira(rapidViewId string) *communicator.Sprint {
	var created *communicator.Sprint
	toCreate := communicator.SprintWithMeta{}
	toCreate.RapidView = rapidViewId
	jiraCreateSprintResponse, err := utility.GetUtilitiesSingleton().JiraClient.CreateSprint(
		utility.GetUtilitiesSingleton().CommsContext, &toCreate)
	if err == nil && jiraCreateSprintResponse.Error == nil && jiraCreateSprintResponse.Sprint != nil {
		created = jiraCreateSprintResponse.Sprint
	}
	return created
}
func (jiraService *JiraService) CreateIssueInJira(issue *communicator.IssueCreate) *communicator.Issue {
	jiraCreateIssueResponse, err := utility.GetUtilitiesSingleton().JiraClient.CreateIssue(
		utility.GetUtilitiesSingleton().CommsContext, issue)
	if err == nil && jiraCreateIssueResponse.Error == nil && jiraCreateIssueResponse.Issue != nil {
		return jiraCreateIssueResponse.Issue
	}
	return nil
}
func (jiraService *JiraService) CreateWorklogInJira(issueKey string, worklog *communicator.WorklogWithMeta) *communicator.Worklog {
	jiraCreateWorklogRequest := new(communicator.Request)
	jiraCreateWorklogRequest.Issue = issueKey
	worklogCreate := new(communicator.WorklogCreate)
	worklogCreate.TimeSpentSeconds = worklog.TimeSpentSeconds
	worklogCreate.Comment = worklog.Comment
	worklogCreate.Started = worklog.Started
	worklogCreate.Author = new(communicator.WorklogCreateAuthor)
	worklogCreate.Author.EmailAddress = worklog.Author.EmailAddress
	jiraCreateWorklogRequest.Worklog = worklogCreate
	jiraCreateWorklogResponse, err := utility.GetUtilitiesSingleton().JiraClient.CreateWorklog(
		utility.GetUtilitiesSingleton().CommsContext, jiraCreateWorklogRequest)
	if err == nil && jiraCreateWorklogResponse.Error == nil && jiraCreateWorklogResponse.Worklog != nil {
		return jiraCreateWorklogResponse.Worklog
	}
	return nil
}
func (jiraService *JiraService) UpdateWorklogInJira(issueKey string,
	worklog *communicator.WorklogWithMeta) *communicator.Worklog {

	jiraUpdateWorklogRequest := new(communicator.Request)
	jiraUpdateWorklogRequest.KeyOrId = worklog.Id
	jiraUpdateWorklogRequest.Issue = issueKey
	worklogUpdate := new(communicator.WorklogCreate)
	worklogUpdate.TimeSpentSeconds = worklog.TimeSpentSeconds
	worklogUpdate.Comment = worklog.Comment
	worklogUpdate.Started = worklog.Started
	worklogUpdate.Author = new(communicator.WorklogCreateAuthor)
	worklogUpdate.Author.EmailAddress = worklog.Author.EmailAddress
	jiraUpdateWorklogRequest.Worklog = worklogUpdate
	jiraUpdateWorklogResponse, err := utility.GetUtilitiesSingleton().JiraClient.UpdateWorklog(
		utility.GetUtilitiesSingleton().CommsContext, jiraUpdateWorklogRequest)
	if err == nil && jiraUpdateWorklogResponse.Error == nil && jiraUpdateWorklogResponse.Worklog != nil {
		return jiraUpdateWorklogResponse.Worklog
	}
	return nil
}
func (jiraService *JiraService) UpdateIssueInJira(issue *communicator.IssueCreate) bool {
	jiraUpdateIssueResponse, err := utility.GetUtilitiesSingleton().JiraClient.UpdateIssue(
		utility.GetUtilitiesSingleton().CommsContext, issue)
	if err == nil && jiraUpdateIssueResponse.Error == nil {
		return true
	}
	return false
}

func (jiraService *JiraService) UpdateSprintInJira(sprint *communicator.SprintWithMeta) error {
	jiraCreateSprintResponse, err := utility.GetUtilitiesSingleton().JiraClient.UpdateSprint(
		utility.GetUtilitiesSingleton().CommsContext, sprint)
	if err != nil || jiraCreateSprintResponse.Error != nil || jiraCreateSprintResponse.Sprint == nil {
		return errors.New("Failed to update sprint in JIRA")
	}
	return nil
}

func (jiraService *JiraService) DoesProjectExistInJira(projectId int32, exists chan bool) {
	var does bool
	var jiraProjectResponse *communicator.Response
	var projectRequest communicator.Request
	projectRequest.Project = fmt.Sprint(projectId)
	jiraProjectResponse, err := utility.GetUtilitiesSingleton().JiraClient.GetProject(
		utility.GetUtilitiesSingleton().CommsContext, &projectRequest)
	if err == nil && jiraProjectResponse.Error == nil &&
		jiraProjectResponse != nil &&
		jiraProjectResponse.Project != nil {
		does = true
	}
	exists <- does
}

func (jiraService *JiraService) DoesEpicExistInJiraProject(epicKey string, exists chan bool) {
	var does bool
	var jiraEpicResponse *communicator.Response
	var epicRequest communicator.Request

	epicRequest.Epic = epicKey
	jiraEpicResponse, err := utility.GetUtilitiesSingleton().JiraClient.GetEpic(
		utility.GetUtilitiesSingleton().CommsContext, &epicRequest)
	if err == nil &&
		jiraEpicResponse.Error == nil &&
		jiraEpicResponse != nil &&
		jiraEpicResponse.Issue != nil {
		does = true
	}
	exists <- does
}

func (jiraService *JiraService) GetEpicInJiraProject(epicKey string) *communicator.Issue {
	var jiraEpicResponse *communicator.Response
	var epicRequest communicator.Request

	epicRequest.Epic = epicKey
	jiraEpicResponse, err := utility.GetUtilitiesSingleton().JiraClient.GetEpic(
		utility.GetUtilitiesSingleton().CommsContext, &epicRequest)
	if err == nil &&
		jiraEpicResponse.Error == nil &&
		jiraEpicResponse != nil &&
		jiraEpicResponse.Issue != nil {
		return jiraEpicResponse.Issue
	}
	return nil
}

func (jiraService *JiraService) RetrieveRapidViewsInProject(projectKey string,
	views chan []communicator.GreenhopperRapidView) {

	var rapidViewsResponse *communicator.Response
	var rapidViewsRequest communicator.Request
	var rapidViews []communicator.GreenhopperRapidView
	rapidViewsRequest.Project = projectKey
	rapidViewsResponse, err := utility.GetUtilitiesSingleton().JiraClient.GetRapidViews(
		utility.GetUtilitiesSingleton().CommsContext, &rapidViewsRequest)
	if err == nil && rapidViewsResponse.Error == nil && rapidViewsResponse.RapidViews != nil {
		for _, rapidView := range rapidViewsResponse.RapidViews {
			rapidViews = append(rapidViews, *rapidView)
		}
	}
	views <- rapidViews
}

func (jiraService *JiraService) RetrieveSprintsInProject(projectKey string, sprints chan []communicator.Sprint) {
	var jiraSprintsResponse *communicator.Response
	var sprintsRequest communicator.Request
	var jiraSprints []communicator.Sprint
	sprintsRequest.Project = projectKey
	jiraSprintsResponse, err := utility.GetUtilitiesSingleton().JiraClient.GetSprints(
		utility.GetUtilitiesSingleton().CommsContext, &sprintsRequest)
	if err == nil && jiraSprintsResponse.Error == nil && jiraSprintsResponse.Sprints != nil {
		for _, jiraSprint := range jiraSprintsResponse.Sprints {
			jiraSprints = append(jiraSprints, *jiraSprint)
		}
	}
	sprints <- jiraSprints
}

func (jiraService *JiraService) RetrieveIssuesFromSprintInProject(projectKey string, sprintName string,
	issues chan []communicator.Issue) {

	var jiraIssuesResponse *communicator.Response
	var issuesRequest communicator.Request
	var jiraIssues []communicator.Issue
	issuesRequest.Project = projectKey
	issuesRequest.Sprint = sprintName
	jiraIssuesResponse, err := utility.GetUtilitiesSingleton().JiraClient.GetIssues(
		utility.GetUtilitiesSingleton().CommsContext, &issuesRequest)
	if err == nil && jiraIssuesResponse.Error == nil && jiraIssuesResponse.Issues != nil &&
		jiraIssuesResponse.Issues.Issues != nil {
		for _, jiraIssue := range jiraIssuesResponse.Issues.Issues {
			jiraIssues = append(jiraIssues, *jiraIssue)
		}
	}
	issues <- jiraIssues
}

func (jiraService *JiraService) RetrieveIssueInProject(projectKey string, issueId string) *communicator.Issue {
	var jiraIssuesResponse *communicator.Response
	var issuesRequest communicator.Request
	issuesRequest.Project = projectKey
	issuesRequest.Issue = issueId
	jiraIssuesResponse, err := utility.GetUtilitiesSingleton().JiraClient.GetIssueById(
		utility.GetUtilitiesSingleton().CommsContext, &issuesRequest)
	if err == nil && jiraIssuesResponse.Error == nil && jiraIssuesResponse.Issue != nil {
		return jiraIssuesResponse.Issue
	}
	return nil
}

func (jiraService *JiraService) UpdateSprintInfoForJiraIssue(sprintId string, issueKey string) bool {
	var moveRequest communicator.Request
	moveRequest.Sprint = sprintId
	moveRequest.Issue = issueKey
	response, err := utility.GetUtilitiesSingleton().JiraClient.MoveIssueToSprint(
		utility.GetUtilitiesSingleton().CommsContext, &moveRequest)
	if nil != err || nil != response.Error {
		return false
	}
	return true
}

func (jiraService *JiraService) UpdateEpicInfoForJiraIssue(epicKey string, issueKey string) bool {
	var moveRequest communicator.Request
	moveRequest.Epic = epicKey
	moveRequest.Issue = issueKey
	response, err := utility.GetUtilitiesSingleton().JiraClient.AddIssueToEpic(
		utility.GetUtilitiesSingleton().CommsContext, &moveRequest)
	if nil != err || nil != response.Error {
		return false
	}
	return true
}

func (jiraService *JiraService) GetJiraIssueIdFromProjectKeyAndIssueKey(projectKey string, issueKey string) int32 {
	var issueRequest communicator.Request
	issueRequest.Project = projectKey
	issueRequest.Issue = issueKey
	response, err := utility.GetUtilitiesSingleton().JiraClient.GetIssue(
		utility.GetUtilitiesSingleton().CommsContext, &issueRequest)
	if nil == err && nil == response.Error && nil != response.Issue {
		issueId64, issueId64Err := strconv.ParseInt(response.Issue.Id, 10, 32)
		if issueId64Err != nil {
			return 0
		}
		return int32(issueId64)
	}
	return 0
}

func (jiraService *JiraService) GetWorklogsFromIssue(issueKey string, worklogs chan []communicator.Worklog) {
	var accumulatedWorklogs []communicator.Worklog
	var issueRequest communicator.Issue
	issueRequest.Key = issueKey
	response, err := utility.GetUtilitiesSingleton().JiraClient.GetIssueWorklogs(
		utility.GetUtilitiesSingleton().CommsContext, &issueRequest)
	if nil == err && nil == response.Error && nil != response.Worklogs && nil != response.Worklogs.Worklogs {
		for _, worklog := range response.Worklogs.Worklogs {
			accumulatedWorklogs = append(accumulatedWorklogs, *worklog)
		}
	}
	worklogs <- accumulatedWorklogs
}

func (jiraService *JiraService) GetUsersInProject(projectName string, users chan []communicator.Author) {
	var availableUsers []communicator.Author
	var request communicator.Request
	request.Project = projectName
	response, err := utility.GetUtilitiesSingleton().JiraClient.GetUsers(
		utility.GetUtilitiesSingleton().CommsContext, &request)
	if nil == err && nil == response.Error && nil != response.Authors && len(response.Authors) > 0 {
		for _, author := range response.Authors {
			availableUsers = append(availableUsers, *author)
		}
	}
	users <- availableUsers
}

func (jiraService *JiraService) GetJiraStatusMetadata(projectId string, statuses chan []communicator.Status) {
	var availableStatuses []communicator.Status
	var request communicator.Request
	request.Project = projectId
	response, err := utility.GetUtilitiesSingleton().JiraClient.GetIssueStatuses(
		utility.GetUtilitiesSingleton().CommsContext, &request)
	if nil == err && nil == response.Error && nil != response.Statuses {
		for _, status := range response.Statuses {
			availableStatuses = append(availableStatuses, *status)
		}
	}
	statuses <- availableStatuses
}

func (jiraService *JiraService) GetJiraPriorityMetadata(projectId string, priorities chan []communicator.Priority) {
	var availablePriorities []communicator.Priority
	var request communicator.Request
	request.Project = projectId
	response, err := utility.GetUtilitiesSingleton().JiraClient.GetIssuePriorities(
		utility.GetUtilitiesSingleton().CommsContext, &request)
	if nil == err && nil == response.Error && nil != response.Priorities {
		for _, priority := range response.Priorities {
			availablePriorities = append(availablePriorities, *priority)
		}
	}
	priorities <- availablePriorities
}
