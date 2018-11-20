package functions

import (
	jiraCommunicator "github.com/desertjinn/jira-communicator/proto/jira-communicator"
	mavenlinkCommunicator "github.com/desertjinn/mavenlink-communicator/proto/mavenlink-communicator"
	datasourceCommunicator "github.com/desertjinn/mavenlink-jira-datasource/proto/mavenlink-jira-datasource"
	"github.com/desertjinn/mavenlink-jira-sync/POGO"
	"github.com/desertjinn/mavenlink-jira-sync/utility"
	"regexp"
	"strconv"
	"strings"
)

type IssueFunctionsInterface interface {
	GetTasksToBeProcessedAsIssues(allTasks []*mavenlinkCommunicator.Task, jiraIssues []*jiraCommunicator.Issue,
		toBeCreated bool) ([]*mavenlinkCommunicator.Task,
		map[string]*jiraCommunicator.Issue)
	PrepareIssuesForCreation(issuesAndTasks *POGO.IssueAndTask) (<-chan jiraCommunicator.IssueWithMeta, <-chan bool)
	PrepareIssuesForUpdate(issuesAndTasks *POGO.IssueAndTask) (<-chan jiraCommunicator.IssueWithMeta, <-chan bool)
	GenerateIssueForCreation(project *jiraCommunicator.Project,
		issue *jiraCommunicator.IssueWithMeta, sprintId string) *jiraCommunicator.IssueCreate
	GenerateIssueForUpdate(project *jiraCommunicator.Project,
		issue jiraCommunicator.IssueWithMeta) *jiraCommunicator.IssueCreate
}

type IssueFunctions struct {
	cf CommonFunctions
}

// Check if JIRA issue and Mavenlink task combination exists in the datasource
func doesIssueAndTaskExistInDatasource(task int32, issue int32) bool {
	var does bool
	taskAndSprint := &datasourceCommunicator.ExternalTasks{}
	taskAndSprint.Source1TaskId = issue
	taskAndSprint.Source2TaskId = task
	taskAndSprintResponse, taskAndSprintResponseErr :=
		utility.GetUtilitiesSingleton().ConfigurationDatasource.GetTaskAndIssue(utility.GetUtilitiesSingleton().CommsContext,
			taskAndSprint)
	if taskAndSprintResponseErr == nil && taskAndSprintResponse.Error == nil && taskAndSprintResponse.Task != nil {
		if taskAndSprintResponse.Task.Id != 0 {
			does = true
		}
	}
	return does
}

// Check if a Mavenlink task that is part of a Mavenlink sub-task(ie, a JIRA Sprint) exists in the datasource
func doesTaskInSubTaskExistInDatasource(task int32) *datasourceCommunicator.ExternalTasks {
	taskAndIssue := &datasourceCommunicator.ExternalTasks{}
	taskAndIssue.Source2TaskId = task
	taskAndIssueResponse, taskAndSprintResponseErr :=
		utility.GetUtilitiesSingleton().ConfigurationDatasource.GetTaskInSubTaskFromId(utility.GetUtilitiesSingleton().CommsContext,
			taskAndIssue)
	if taskAndSprintResponseErr == nil && taskAndIssueResponse.Error == nil && taskAndIssueResponse.Task != nil {
		if taskAndIssueResponse.Task.Id != 0 {
			return taskAndIssueResponse.Task
		}
	}
	return nil
}

// Get the JIRA issue related to the Mavenlink task
func getMatchingIssueForTask(issues []*jiraCommunicator.Issue, task *mavenlinkCommunicator.Task) *jiraCommunicator.Issue {
	var taskId int32
	taskId64, taskIdErr := strconv.ParseInt(task.Id, 10, 32)
	if taskIdErr != nil {
		return nil
	}
	taskId = int32(taskId64)
	taskInDb := doesTaskInSubTaskExistInDatasource(taskId)
	if nil != taskInDb {
		for _, issue := range issues {
			var issueId int32
			issueId64, issueIdErr := strconv.ParseInt(issue.Id, 10, 32)
			if issueIdErr != nil {
				continue
			}
			issueId = int32(issueId64)
			if issueId == taskInDb.Source1TaskId {
				exists := doesIssueAndTaskExistInDatasource(taskId, issueId)
				if exists == true {
					return issue
				}
			}
		}
	}
	return nil
}

func prepIssue(task *mavenlinkCommunicator.Task, existingIssue *jiraCommunicator.Issue,
	users []*jiraCommunicator.Author,
	issueType *jiraCommunicator.IssueType, issueStatus *jiraCommunicator.Status,
	issuePriority *jiraCommunicator.Priority, toBeUpdated bool) *jiraCommunicator.IssueWithMeta {

	issue := new(jiraCommunicator.IssueWithMeta)
	issue.Fields = new(jiraCommunicator.Fields)
	issue.Fields.Issuetype = new(jiraCommunicator.IssueType)
	issue.Fields.Status = new(jiraCommunicator.Status)
	issue.Fields.Priority = new(jiraCommunicator.Priority)
	issue.ToBeUpdated = toBeUpdated
	issue.Fields.Summary = task.Title
	issue.Fields.Description = task.Description
	issue.Fields.Duedate = task.DueDate
	issue.Fields.Issuetype.Name = task.StoryType
	issue.Fields.Status.Name = task.State
	issue.Fields.Priority.Name = task.Priority
	if task.User != nil {
		for _, user := range users {
			if strings.EqualFold(user.EmailAddress, task.User.EmailAddress) {
				issue.Fields.Assignee = new(jiraCommunicator.Author)
				issue.Fields.Assignee.Name = user.Name
				break
			}
		}
	}

	parentTaskId64, parentTaskId64Err := strconv.ParseInt(task.ParentId, 10, 32)
	taskId64, taskId64Err := strconv.ParseInt(task.Id, 10, 32)
	if nil != parentTaskId64Err || nil != taskId64Err {
		return nil
	}
	issue.MavenlinkTaskId = int32(taskId64)
	issue.MavenlinkParentTaskId = int32(parentTaskId64)
	if issueType != nil {
		issue.ExistingIssueType = issueType.Name
	}
	if issueStatus != nil {
		issue.ExistingIssueStatus = issueStatus.Name
	}
	if issuePriority != nil {
		issue.ExistingIssuePriority = issuePriority.Name
	}
	if existingIssue != nil {
		if len(existingIssue.Id) > 0 {
			issue.Id = existingIssue.Id
		}
		if len(existingIssue.Fields.Customfield_10722[0]) > 0 {
			issue.JiraSprint = existingIssue.Fields.Customfield_10722[0]
			pat := regexp.MustCompile(`id=([0-9]+)]`)
			apiSprintIdMatch := pat.FindStringSubmatch(existingIssue.Fields.Customfield_10722[0])
			if 0 < len(apiSprintIdMatch[1]) {
				issue.ExistingIssueSprintId = apiSprintIdMatch[1]
			}
		}
		if len(existingIssue.Fields.Customfield_11022) > 0 {
			issue.JiraEpic = existingIssue.Fields.Customfield_11022
			issue.ExistingIssueEpicId = existingIssue.Fields.Customfield_11022
		}
		if len(existingIssue.Key) > 0 {
			issue.ExistingIssueKey = existingIssue.Key
		}
	}
	return issue
}


// Get the Mavenlink tasks to be processed as JIRA issues
func (self *IssueFunctions) GetTasksToBeProcessedAsIssues(allTasks []*mavenlinkCommunicator.Task, jiraIssues []*jiraCommunicator.Issue,
	toBeCreated bool) ([]*mavenlinkCommunicator.Task,
	map[string]*jiraCommunicator.Issue) {

	var tasks []*mavenlinkCommunicator.Task
	issues := map[string]*jiraCommunicator.Issue{}
	for _, task := range allTasks {
		issue := getMatchingIssueForTask(jiraIssues, task)
		if toBeCreated == true {
			if issue == nil {
				tasks = append(tasks, task)
			}
		} else {
			if issue != nil {
				tasks = append(tasks, task)
				issues[task.Id] = issue
			}
		}
	}
	return tasks, issues
}

// Prepare Mavenlink sub-tasks as JIRA issues for creation purposes
func (self *IssueFunctions) PrepareIssuesForCreation(issuesAndTasks *POGO.IssueAndTask) (
	<-chan jiraCommunicator.IssueWithMeta, <-chan bool) {

	issueChannel := make(chan jiraCommunicator.IssueWithMeta)
	issueChannelClosed := make(chan bool)
	go func() {
		toBeCreated, _ := self.GetTasksToBeProcessedAsIssues(issuesAndTasks.GetTasks(), issuesAndTasks.GetIssues(),
			true)
		for _, toBe := range toBeCreated {
			issueType := self.cf.GetJiraIssueTypeFromMetadata(toBe.StoryType, "")
			if issueType == nil {
				issueType = self.cf.GetDefaultEquivalentJiraIssueType()
			}
			status := self.cf.GetJiraStatusFromMetadata(toBe.State, "")
			if status == nil {
				status = self.cf.GetDefaultEquivalentJiraIssueStatus()
			}
			priority := self.cf.GetJiraPriorityFromMetadata(toBe.Priority, "")
			if priority == nil {
				priority = self.cf.GetDefaultEquivalentJiraIssuePriority()
			}
			issue := prepIssue(toBe, nil, issuesAndTasks.GetUsers(), issueType,
				status, priority, false)
			if issue != nil {
				issueChannel <- *issue
			}
		}
		issueChannelClosed <- true
	}()
	return issueChannel, issueChannelClosed
}

// Prepare Mavenlink sub-tasks as existing JIRA issues for update purposes
func (self *IssueFunctions) PrepareIssuesForUpdate(issuesAndTasks *POGO.IssueAndTask) (
	<-chan jiraCommunicator.IssueWithMeta, <-chan bool) {

	issueChannel := make(chan jiraCommunicator.IssueWithMeta)
	issueChannelClosed := make(chan bool)
	go func() {
		toBeSynced, relatedIssues := self.GetTasksToBeProcessedAsIssues(issuesAndTasks.GetTasks(),
			issuesAndTasks.GetIssues(), false)
		for _, toBe := range toBeSynced {
			existingIssue := relatedIssues[toBe.Id]
			//issueType := GetJiraIssueTypeFromMetadata(toBe.StoryType, existingIssue.Fields.Issuetype.Name)
			status := self.cf.GetJiraStatusFromMetadata(toBe.State, existingIssue.Fields.Status.Name)
			priority := self.cf.GetJiraPriorityFromMetadata(toBe.Priority, existingIssue.Fields.Priority.Name)
			var toBeUpdated bool
			if !strings.EqualFold(existingIssue.Fields.Summary, toBe.Title) ||
				!strings.EqualFold(existingIssue.Fields.Description, toBe.Description) {

				toBeUpdated = true
			}
			if existingIssue.Fields.Assignee != nil && toBe.User != nil &&
				!strings.EqualFold(existingIssue.Fields.Assignee.EmailAddress, toBe.User.EmailAddress) {
				toBeUpdated = true
			}
			if existingIssue.Fields.Assignee == nil && toBe.User != nil {
				toBeUpdated = true
			}
			//if issueType != nil && !strings.EqualFold(issueType.Name, toBe.StoryType) {
			//	toBeUpdated = true
			//}
			//issue := prepIssue(toBe, existingIssue, issuesAndTasks.GetUsers(), issueType,
			//	status, priority, toBeUpdated)
			issue := prepIssue(toBe, existingIssue, issuesAndTasks.GetUsers(), nil,
				status, priority, toBeUpdated)
			if issue != nil {
				issueChannel <- *issue
			}
		}
		issueChannelClosed <- true
	}()
	return issueChannel, issueChannelClosed
}

// Generate the JIRA issue object to be used for creating an issue
func (self *IssueFunctions) GenerateIssueForCreation(project *jiraCommunicator.Project,
	issue *jiraCommunicator.IssueWithMeta, sprintId string) *jiraCommunicator.IssueCreate {

	issueType := self.cf.GetJiraIssueTypeFromMetadata(issue.Fields.Issuetype.Name, issue.ExistingIssueType)
	status := self.cf.GetJiraStatusFromMetadata(issue.Fields.Status.Name, issue.ExistingIssueStatus)
	priority := self.cf.GetJiraPriorityFromMetadata(issue.Fields.Priority.Name, issue.ExistingIssuePriority)
	if issueType != nil && priority != nil && status != nil {
		createIssue := new(jiraCommunicator.IssueCreate)

		createIssue.Fields = new(jiraCommunicator.FieldsForCreate)
		createIssue.Fields.Summary = issue.Fields.Summary
		createIssue.Fields.Description = issue.Fields.Description
		createIssue.Fields.Duedate = issue.Fields.Duedate
		createIssue.Fields.Customfield_10722 = sprintId

		if issue.Fields.Assignee != nil {
			createIssue.Fields.Assignee = new(jiraCommunicator.Author)
			createIssue.Fields.Assignee.Name = issue.Fields.Assignee.Name
		}

		createIssue.Fields.Issuetype = new(jiraCommunicator.IssueType)
		createIssue.Fields.Issuetype.Name = ""
		createIssue.Fields.Issuetype.Id = issueType.Id

		createIssue.Fields.Priority = new(jiraCommunicator.Priority)
		createIssue.Fields.Priority.Name = ""
		createIssue.Fields.Priority.Id = priority.Id

		createIssue.Fields.Status = new(jiraCommunicator.Status)
		createIssue.Fields.Status.Name = ""
		createIssue.Fields.Status.Id = status.Id

		createIssue.Fields.Project = new(jiraCommunicator.Project)
		createIssue.Fields.Project.Key = project.Key

		return createIssue
	}
	return nil
}

// Generate the JIRA issue object to be used to update an issue
func (self *IssueFunctions) GenerateIssueForUpdate(project *jiraCommunicator.Project,
	issue jiraCommunicator.IssueWithMeta) *jiraCommunicator.IssueCreate {

	issueType := self.cf.GetJiraIssueTypeFromMetadata(issue.Fields.Issuetype.Name, issue.ExistingIssueType)
	status := self.cf.GetJiraStatusFromMetadata(issue.Fields.Status.Name, issue.ExistingIssueStatus)
	priority := self.cf.GetJiraPriorityFromMetadata(issue.Fields.Priority.Name, issue.ExistingIssuePriority)
	if issueType != nil && priority != nil && status != nil {
		updateIssue := new(jiraCommunicator.IssueCreate)

		updateIssue.Id = issue.Id

		updateIssue.Fields = new(jiraCommunicator.FieldsForCreate)
		updateIssue.Key = issue.ExistingIssueKey
		updateIssue.Fields.Summary = issue.Fields.Summary
		updateIssue.Fields.Description = issue.Fields.Description
		updateIssue.Fields.Duedate = issue.Fields.Duedate

		if issue.Fields.Assignee != nil {
			updateIssue.Fields.Assignee = new(jiraCommunicator.Author)
			updateIssue.Fields.Assignee.Name = issue.Fields.Assignee.Name
		}

		updateIssue.Fields.Issuetype = new(jiraCommunicator.IssueType)
		updateIssue.Fields.Issuetype.Name = ""
		updateIssue.Fields.Issuetype.Id = issueType.Id

		updateIssue.Fields.Priority = new(jiraCommunicator.Priority)
		updateIssue.Fields.Priority.Name = ""
		updateIssue.Fields.Priority.Id = priority.Id

		updateIssue.Fields.Status = new(jiraCommunicator.Status)
		updateIssue.Fields.Status.Name = ""
		updateIssue.Fields.Status.Id = status.Id

		return updateIssue
	}
	return nil
}
