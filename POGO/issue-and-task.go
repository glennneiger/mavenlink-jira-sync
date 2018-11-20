package POGO

import (
	jira "github.com/desertjinn/jira-communicator/proto/jira-communicator"
	mavenlink "github.com/desertjinn/mavenlink-communicator/proto/mavenlink-communicator"
)

type IssueAndTaskInterface interface {
	SetProject(project jira.Project)
	GetProject() *jira.Project
	SetEpic(epic jira.Issue)
	GetEpic() *jira.Issue
	SetUsers(users []jira.Author)
	GetUsers() *jira.Project
	SetTasks(tasks []mavenlink.Task)
	GetTasks() []*mavenlink.Task
	SetTimeentries(timeentries []mavenlink.Timeentry)
	GetTimeentries() []*mavenlink.Timeentry
	AddTimeentry(timeentry mavenlink.Timeentry)
	SetIssues(issues []jira.Issue)
	GetIssues() []*jira.Issue
	SetWorklogs(worklogs []jira.Worklog)
	GetWorklogs() []*jira.Worklog
	AddWorklog(worklog jira.Worklog)
}

type IssueAndTask struct {
	project     *jira.Project
	epic        *jira.Issue
	users       []*jira.Author
	tasks       []*mavenlink.Task
	timeentries []*mavenlink.Timeentry
	issues      []*jira.Issue
	worklogs    []*jira.Worklog
}

func (st *IssueAndTask) SetProject(project *jira.Project) {
	st.project = project
}

func (st *IssueAndTask) GetProject() *jira.Project {
	return st.project
}

func (st *IssueAndTask) SetEpic(epic *jira.Issue) {
	st.epic = epic
}

func (st *IssueAndTask) GetEpic() *jira.Issue {
	return st.epic
}

func (st *IssueAndTask) SetUsers(users []jira.Author) {
	for userKey := range users {
		st.users = append(st.users, &users[userKey])
	}
}

func (st *IssueAndTask) GetUsers() []*jira.Author {
	return st.users
}

func (st *IssueAndTask) GetTasks() []*mavenlink.Task {
	return st.tasks
}
func (st *IssueAndTask) SetTasks(tasks []mavenlink.Task) {
	for taskKey := range tasks {
		st.tasks = append(st.tasks, &tasks[taskKey])
	}
}
func (st *IssueAndTask) GetTimeentries() []*mavenlink.Timeentry {
	return st.timeentries
}
func (st *IssueAndTask) SetTimeentries(timeentries []mavenlink.Timeentry) {
	for timeentryKey := range timeentries {
		st.timeentries = append(st.timeentries, &timeentries[timeentryKey])
	}
}
func (st *IssueAndTask) AddTimeentry(timeentry mavenlink.Timeentry) {
	st.timeentries = append(st.timeentries, &timeentry)
}
func (st *IssueAndTask) GetIssues() []*jira.Issue {
	return st.issues
}
func (st *IssueAndTask) SetIssues(issues []jira.Issue) {
	for issueKey := range issues {
		st.issues = append(st.issues, &issues[issueKey])
	}
}
func (st *IssueAndTask) GetWorklogs() []*jira.Worklog {
	return st.worklogs
}
func (st *IssueAndTask) SetWorklogs(worklogs []jira.Worklog) {
	for worklogKey := range worklogs {
		st.worklogs = append(st.worklogs, &worklogs[worklogKey])
	}
}
func (st *IssueAndTask) AddWorklog(worklog jira.Worklog) {
	st.worklogs = append(st.worklogs, &worklog)
}
