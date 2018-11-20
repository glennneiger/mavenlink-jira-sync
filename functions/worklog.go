package functions

import (
	"fmt"
	jiraCommunicator "git.costrategix.net/go/jira-communicator/proto/jira-communicator"
	mavenlinkCommunicator "git.costrategix.net/go/mavenlink-communicator/proto/mavenlink-communicator"
	datasourceCommunicator "git.costrategix.net/go/mavenlink-jira-datasource/proto/mavenlink-jira-datasource"
	"git.costrategix.net/go/mavenlink-jira-sync/POGO"
	"git.costrategix.net/go/mavenlink-jira-sync/utility"
	"regexp"
	"strconv"
	"strings"
)

type WorklogFunctionsInterface interface {
	GetTimeEntriesToBeProcessedAsWorklogs(allTimeEntries []*mavenlinkCommunicator.Timeentry,
		jiraWorklogs []*jiraCommunicator.Worklog, toBeCreated bool) ([]*mavenlinkCommunicator.Timeentry,
		map[string]*jiraCommunicator.Worklog)
	PrepareWorklogsForCreation(issuesAndTasks *POGO.IssueAndTask) (
		chan jiraCommunicator.WorklogWithMeta, chan bool)
	PrepareWorklogsForUpdate(issuesAndTasks *POGO.IssueAndTask) (
		chan jiraCommunicator.WorklogWithMeta, chan bool)
}

type WorklogFunctions struct {
	cf CommonFunctions
}

// Check if a Mavenlink time entry exists in the datasource
func doesTimeEntryExistInDataSource(timeentry int32) *datasourceCommunicator.ExternalTimeEntries {
	externalTimeEntry := &datasourceCommunicator.ExternalTimeEntries{}
	externalTimeEntry.Source2LogId = timeentry
	worklogAndTimeEntryResponse, worklogAndTimeentryResponseErr :=
		utility.GetUtilitiesSingleton().ConfigurationDatasource.GetTimeentry(utility.GetUtilitiesSingleton().CommsContext,
			externalTimeEntry)
	if worklogAndTimeentryResponseErr == nil && worklogAndTimeEntryResponse.Error == nil &&
		worklogAndTimeEntryResponse.Timeentry != nil {
		if worklogAndTimeEntryResponse.Timeentry.Id != 0 {
			return worklogAndTimeEntryResponse.Timeentry
		}
	}
	return nil
}

// Check if JIRA worklog and Mavenlink time entry combination exists in the data source
func doesWorklogAndTimeEntryExistInDataSource(timeentry int32, worklog int32) bool {
	var does bool
	worklogAndTimeentry := &datasourceCommunicator.ExternalTimeEntries{}
	worklogAndTimeentry.Source1LogId = worklog
	worklogAndTimeentry.Source2LogId = timeentry
	worklogAndTimeentryResponse, worklogAndTimeentryResponseErr :=
		utility.GetUtilitiesSingleton().ConfigurationDatasource.GetTimeentryAndWorklog(utility.GetUtilitiesSingleton().CommsContext,
			worklogAndTimeentry)
	if worklogAndTimeentryResponseErr == nil && worklogAndTimeentryResponse.Error == nil &&
		worklogAndTimeentryResponse.Timeentry != nil {

		if worklogAndTimeentryResponse.Timeentry.Id != 0 {
			does = true
		}
	}
	return does
}

// Get the JIRA worklog related to the Mavenlink time entry
func getMatchingWorklogForTimeEntry(worklogs []*jiraCommunicator.Worklog,
	timeEntry *mavenlinkCommunicator.Timeentry) *jiraCommunicator.Worklog {

	var timeentryId int32
	timeentryId64, timeentryIdErr := strconv.ParseInt(timeEntry.Id, 10, 32)
	if timeentryIdErr != nil {
		return nil
	}
	timeentryId = int32(timeentryId64)
	timeentryInDb := doesTimeEntryExistInDataSource(timeentryId)
	if nil != timeentryInDb {
		for _, worklog := range worklogs {
			var worklogId int32
			issueId64, issueIdErr := strconv.ParseInt(worklog.Id, 10, 32)
			if issueIdErr != nil {
				continue
			}
			worklogId = int32(issueId64)
			if worklogId == timeentryInDb.Source1LogId {
				exists := doesWorklogAndTimeEntryExistInDataSource(timeentryId, worklogId)
				if exists == true {
					return worklog
				}
			}
		}
	}
	return nil
}

func prepWorklog(timeEntry *mavenlinkCommunicator.Timeentry, timezone string,
	worklogId string) *jiraCommunicator.WorklogWithMeta {

	worklog := new(jiraCommunicator.WorklogWithMeta)
	if len(worklogId) > 0 {
		worklog.Id = worklogId
	}
	worklog.TimeSpentSeconds = int64(timeEntry.TimeInMinutes * 60)
	worklog.Comment = timeEntry.Notes
	if len(timezone) <= 0 {
		timezone = "+0530"
	}
	worklog.Started = fmt.Sprintf("%sT%s%s", timeEntry.DatePerformed, "06:00:00.000", timezone)
	worklog.Created = timeEntry.CreatedAt
	worklog.Updated = timeEntry.UpdatedAt
	worklog.Author = new(jiraCommunicator.Author)
	worklog.Author.EmailAddress = timeEntry.User.EmailAddress
	worklog.UpdateAuthor = new(jiraCommunicator.Author)
	worklog.UpdateAuthor.EmailAddress = timeEntry.User.EmailAddress
	worklog.MavenlinkTaskInSubTaskId = timeEntry.StoryId
	worklog.MavenlinkTimeentryId = timeEntry.Id
	worklog.MavenlinkTimeentryUserId = timeEntry.User.Id

	return worklog
}

// Get the Mavenlink tasks to be processed as JIRA issues
func (self *WorklogFunctions) GetTimeEntriesToBeProcessedAsWorklogs(allTimeEntries []*mavenlinkCommunicator.Timeentry,
	jiraWorklogs []*jiraCommunicator.Worklog, toBeCreated bool) ([]*mavenlinkCommunicator.Timeentry,
	map[string]*jiraCommunicator.Worklog) {

	var timeEntries []*mavenlinkCommunicator.Timeentry
	worklogs := map[string]*jiraCommunicator.Worklog{}
	for _, timeEntry := range allTimeEntries {
		worklog := getMatchingWorklogForTimeEntry(jiraWorklogs, timeEntry)
		if toBeCreated == true {
			if worklog == nil {
				timeEntries = append(timeEntries, timeEntry)
			}
		} else {
			if worklog != nil {
				timeEntries = append(timeEntries, timeEntry)
				worklogs[timeEntry.Id] = worklog
			}
		}
	}
	return timeEntries, worklogs
}

// Prepare Mavenlink sub-task time entries as JIRA issue worklogs for creation purposes
func (self *WorklogFunctions) PrepareWorklogsForCreation(issuesAndTasks *POGO.IssueAndTask) (
	chan jiraCommunicator.WorklogWithMeta, chan bool) {

	worklogsChannel := make(chan jiraCommunicator.WorklogWithMeta)
	worklogsChannelClosed := make(chan bool)
	go func() {
		var timezone string
		toBeCreated, _ := self.GetTimeEntriesToBeProcessedAsWorklogs(issuesAndTasks.GetTimeentries(),
			issuesAndTasks.GetWorklogs(), true)
		timezone = "+0530"
		for _, toBe := range toBeCreated {
			preppedWorklog := prepWorklog(toBe, timezone, "")
			worklogsChannel <- *preppedWorklog
		}
		worklogsChannelClosed <- true
	}()
	return worklogsChannel, worklogsChannelClosed
}

// Prepare Mavenlink sub-task time entries as JIRA issue worklogs for creation purposes
func (self *WorklogFunctions) PrepareWorklogsForUpdate(issuesAndTasks *POGO.IssueAndTask) (
	chan jiraCommunicator.WorklogWithMeta, chan bool) {

	worklogsChannel := make(chan jiraCommunicator.WorklogWithMeta)
	worklogsChannelClosed := make(chan bool)
	go func() {
		toBeSynced, relatedWorklogs := self.GetTimeEntriesToBeProcessedAsWorklogs(issuesAndTasks.GetTimeentries(),
			issuesAndTasks.GetWorklogs(), false)
		for _, toBe := range toBeSynced {
			var startedDate string
			var timezone string
			existingWorklog := relatedWorklogs[toBe.Id]
			timezone = "+0530"
			//patForTimezone := regexp.MustCompile(`.*?T[0-9]+:[0-9]+:[0-9]+(.*)`)
			//timezoneMatch := patForTimezone.FindStringSubmatch(toBe.CreatedAt)
			//if 0 < len(timezoneMatch) && 0 < len(timezoneMatch[1]) {
			//	var replacer = strings.NewReplacer(":", "")
			//	timezone = replacer.Replace(timezoneMatch[1])
			//} else {
			//	timezone = "+0530"
			//}
			pat := regexp.MustCompile(`(.*?)T(.*)`)
			startedDateMatch := pat.FindStringSubmatch(existingWorklog.Started)
			if 0 < len(startedDateMatch[1]) {
				startedDate = startedDateMatch[1]
				if existingWorklog.TimeSpentSeconds != int64(toBe.TimeInMinutes*60) ||
					!strings.EqualFold(startedDate, toBe.DatePerformed) ||
					!strings.EqualFold(existingWorklog.Comment, toBe.Notes) {

					preppedWorklog := prepWorklog(toBe, timezone, existingWorklog.Id)
					worklogsChannel <- *preppedWorklog
				}
			} else {
				utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Cross,
					"Failed to find existing worklog date to simple date format(2006-01-02)")
				continue
			}
		}
		worklogsChannelClosed <- true
	}()
	return worklogsChannel, worklogsChannelClosed
}
