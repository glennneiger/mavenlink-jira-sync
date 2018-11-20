package main

import (
	"fmt"
	jiraCommunicator "git.costrategix.net/go/jira-communicator/proto/jira-communicator"
	mavenlinkCommunicator "git.costrategix.net/go/mavenlink-communicator/proto/mavenlink-communicator"
	datasourceCommunicator "git.costrategix.net/go/mavenlink-jira-datasource/proto/mavenlink-jira-datasource"
	"git.costrategix.net/go/mavenlink-jira-sync/POGO"
	"git.costrategix.net/go/mavenlink-jira-sync/functions"
	"git.costrategix.net/go/mavenlink-jira-sync/services"
	"git.costrategix.net/go/mavenlink-jira-sync/utility"
	"strconv"
)

type SyncOperationsInterface interface {
	IsAValidSyncConfiguration(syncConfiguration *datasourceCommunicator.ExternalProject) bool
	SyncMavenlinkToJira(externalProject *datasourceCommunicator.ExternalProject, success chan bool)
}
type SyncOperations struct {
	common     functions.CommonFunctionsInterface
	worklog    functions.WorklogFunctionsInterface
	issue      functions.IssueFunctionsInterface
	sprint     functions.SprintFunctionsInterface
	jira       services.JiraServiceInterface
	mavenlink  services.MavenlinkServiceInterface
	datasource services.DataSourceServiceInterface
}

func (syncOps *SyncOperations) retrieveAndCollateMavenlinkTasksInSubTasks(sync *datasourceCommunicator.ExternalProject,
	subTasks []*mavenlinkCommunicator.Task, tasks chan []mavenlinkCommunicator.Task) {

	var allTasks []mavenlinkCommunicator.Task
	if nil == sync || nil == subTasks {
		tasks <- allTasks
	}
	tasksInSubTasks := make(chan []mavenlinkCommunicator.Task)
	if len(subTasks) > 0 {
		processedSubTask := 0
		for _, subTask := range subTasks {
			subTaskIdInt64, subTaskIdInt64Err := strconv.ParseInt(subTask.Id, 10, 32)
			if subTaskIdInt64Err != nil {
				continue
			}
			go syncOps.mavenlink.RetrieveTasksFromSubTasksInWorkspace(sync.Source2ProjectId,
				int32(subTaskIdInt64), tasksInSubTasks)
			processedSubTask++
		}
		for i := 0; i < processedSubTask; i++ {
			currentTasks := <-tasksInSubTasks
			allTasks = append(allTasks, currentTasks...)
		}
	}
	tasks <- allTasks
}

func (syncOps *SyncOperations) retrieveAndCollateJiraTasksInSprints(jiraProject *jiraCommunicator.Project,
	sprints []*jiraCommunicator.Sprint, issues chan []jiraCommunicator.Issue) {

	var allIssues []jiraCommunicator.Issue
	if nil == jiraProject || nil == sprints {
		issues <- allIssues
	}
	issuesInSprint := make(chan []jiraCommunicator.Issue)
	if len(sprints) > 0 {
		processedSprint := 0
		for _, sprint := range sprints {
			go syncOps.jira.RetrieveIssuesFromSprintInProject(jiraProject.Key, sprint.Name, issuesInSprint)
			processedSprint++
		}
		for i := 0; i < processedSprint; i++ {
			currentIssues := <-issuesInSprint
			allIssues = append(allIssues, currentIssues...)
		}
	}
	issues <- allIssues
}

func (syncOps *SyncOperations) createSprint(externalProjectId int32, sprint jiraCommunicator.SprintWithMeta,
	created chan bool) {

	justCreated := syncOps.jira.CreateSprintInJira(sprint.RapidView)
	if justCreated != nil {
		sprint.Id = justCreated.Id
		updateErr := syncOps.jira.UpdateSprintInJira(&sprint)
		if updateErr == nil {
			saved := syncOps.datasource.SaveSprintAndTaskSyncHistory(externalProjectId, &sprint)
			if saved == true {
				utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Check,
					"Created sprint and saved sync history")
			} else {
				utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Check,
					"Created sprint")
				utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Cross,
					"FAILED to save sync history")
			}
		}
		created <- true
	} else {
		utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Cross,
			"FAILED to create sprint")
		created <- false
	}
}

func (syncOps *SyncOperations) updateSprint(sprint jiraCommunicator.SprintWithMeta, updating chan bool) {
	toSync := jiraCommunicator.SprintWithMeta{}
	toSync.Id = sprint.Id
	toSync.Name = sprint.Name
	toSync.State = sprint.State
	toSync.LinkedPagesCount = sprint.LinkedPagesCount
	toSync.StartDate = sprint.StartDate
	toSync.EndDate = sprint.EndDate
	toSync.RapidView = sprint.RapidView
	updateErr := syncOps.jira.UpdateSprintInJira(&toSync)
	if updateErr == nil {
		utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Check,
			fmt.Sprintf("Update sprint successful for task with ID: %d", toSync.Id))
	} else {
		utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Cross,
			fmt.Sprintf("FAILED to update sprint for task with ID: %d", toSync.Id))
	}
	updating <- true
}

func (syncOps *SyncOperations) updateSprintOfIssueIfRequired(externalProjectId int32, project *jiraCommunicator.Project,
	issue jiraCommunicator.IssueWithMeta) string {

	var sprintId string
	recordedParentId := syncOps.datasource.GetMavenlinkParentTaskIdFromMavenlinkTaskId(issue.MavenlinkTaskId)
	if issue.MavenlinkParentTaskId != recordedParentId {
		sprintId := syncOps.datasource.GetJiraSprintIdFromMavenlinkTaskId(issue.MavenlinkParentTaskId)
		sprintUpdated := syncOps.jira.UpdateSprintInfoForJiraIssue(sprintId, issue.ExistingIssueKey)
		if sprintUpdated != true {
			utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Cross,
				fmt.Sprintf("FAILED to update sprint info for issue %s", issue.ExistingIssueKey))
		} else {
			updateErr := syncOps.datasource.UpdateIssueAndTaskSyncHistory(externalProjectId, project, &issue,
				sprintId)
			if updateErr != nil {
				utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Check,
					fmt.Sprintf("Updated sprint info %s for issue '%s'",
						sprintId, issue.ExistingIssueKey))
				utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Cross,
					"FAILED to save sync history")
				utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Cross,
					fmt.Sprintf("Error: %v", updateErr))
			} else {
				utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Check,
					fmt.Sprintf("Updated sprint info %s for issue '%s' and saved sync history",
						sprintId, issue.ExistingIssueKey))
			}
		}
	} else {
		sprintId = issue.ExistingIssueSprintId
	}
	return sprintId
}

func (syncOps *SyncOperations) recordWorklogUpdate(issueChannel <-chan jiraCommunicator.Issue,
	worklog *jiraCommunicator.WorklogWithMeta) bool {

	issue := <-issueChannel
	justUpdated := syncOps.jira.UpdateWorklogInJira(issue.Key, worklog)
	if justUpdated != nil {
		saved := syncOps.datasource.UpdateWorklogAndTimeEntrySyncHistory(issue.Id, justUpdated.Id,
			worklog.MavenlinkTimeentryId, worklog.Author.EmailAddress, worklog.MavenlinkTimeentryUserId,
			worklog.TimeSpentSeconds)
		if saved == true {
			utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Check,
				fmt.Sprintf("Update worklog %s and saved sync history", justUpdated.Id))
		} else {
			utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Check,
				fmt.Sprintf("Update worklog %s", justUpdated.Id))
			utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Cross,
				fmt.Sprintf("FAILED to save sync history"))
		}
		return true
	}
	return false
}

func (syncOps *SyncOperations) recordWorklogCreation(issueChannel <-chan jiraCommunicator.Issue,
	worklog *jiraCommunicator.WorklogWithMeta) bool {

	issue := <-issueChannel
	if len(issue.Id) > 0 {
		justCreated := syncOps.jira.CreateWorklogInJira(issue.Key, worklog)
		if justCreated != nil {
			saved := syncOps.datasource.SaveWorklogAndTimeEntrySyncHistory(issue.Id, justCreated.Id,
				worklog.MavenlinkTimeentryId, worklog.Author.EmailAddress, worklog.MavenlinkTimeentryUserId,
				worklog.TimeSpentSeconds)
			if saved == true {
				utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Check,
					fmt.Sprintf("Created worklog %s and saved sync history", justCreated.Id))
			} else {
				utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Check,
					fmt.Sprintf("Created worklog %s", justCreated.Id))
				utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Cross,
					fmt.Sprintf("FAILED to save sync history"))
			}
			return true
		}
	} else {
		utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Cross,
			"FAILED to retrieve JIRA issue's key from task in sub-task")
	}
	return false
}

func (syncOps *SyncOperations) createWorklogs(project *jiraCommunicator.Project,
	worklog jiraCommunicator.WorklogWithMeta, created chan bool) {

	issue := syncOps.datasource.GetJiraIssueFromTaskInSubTask(project.Key, worklog.MavenlinkTaskInSubTaskId)
	if issue != nil {
		recorded := syncOps.recordWorklogCreation(issue, &worklog)
		if recorded {
			created <- true
		} else {
			utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Cross,
				fmt.Sprintf("FAILED to create worklog - %s", worklog.Id))
			created <- false
		}
	}
	created <- false
}

func (syncOps *SyncOperations) updateWorklogs(project *jiraCommunicator.Project,
	worklog jiraCommunicator.WorklogWithMeta, update chan bool) {

	issue := syncOps.datasource.GetJiraIssueFromTaskInSubTask(project.Key, worklog.MavenlinkTaskInSubTaskId)
	recorded := syncOps.recordWorklogUpdate(issue, &worklog)
	if recorded {
		update <- true
	} else {
		utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Cross, "Update failed !!")
		update <- false
	}
	update <- false
}

func (syncOps *SyncOperations) updateIssueAndRecordSyncHistory(externalProjectId int32, project *jiraCommunicator.Project,
	issue jiraCommunicator.IssueWithMeta, sprintId string) {

	updateIssue := syncOps.issue.GenerateIssueForUpdate(project, issue)
	if updateIssue != nil {
		justUpdated := syncOps.jira.UpdateIssueInJira(updateIssue)
		if justUpdated == true {
			updateErr := syncOps.datasource.UpdateIssueAndTaskSyncHistory(externalProjectId, project, &issue,
				sprintId)
			if updateErr != nil {
				utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Check,
					fmt.Sprintf("Updated issue %s in sprint %s via JIRA API", issue.ExistingIssueKey, sprintId))
				utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Cross,
					fmt.Sprintf("FAILED to save sync history"))
			} else {
				utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Check,
					fmt.Sprintf("Updated issue %s in sprint %s and saved sync history",
						issue.ExistingIssueKey, sprintId))
			}
		} else {
			utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Cross,
				fmt.Sprintf("FAILED to update issue %s via JIRA API", issue.ExistingIssueKey))
		}
	} else {
		utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Cross,
			fmt.Sprintf("FAILED to generate an update object for the issue %s", issue.ExistingIssueKey))
	}
}

func (syncOps *SyncOperations) updateIssue(externalProjectId int32, project *jiraCommunicator.Project,
	epic *jiraCommunicator.Issue, issue jiraCommunicator.IssueWithMeta, updates chan bool) {

	sprintId := syncOps.updateSprintOfIssueIfRequired(externalProjectId, project, issue)
	if issue.ToBeUpdated == true {
		syncOps.updateIssueAndRecordSyncHistory(externalProjectId, project, issue, sprintId)
		updates <- true
	} else {
		updates <- false
	}
}

func (syncOps *SyncOperations) createIssue(externalProjectId int32, project *jiraCommunicator.Project,
	epic *jiraCommunicator.Issue, issue jiraCommunicator.IssueWithMeta, created chan bool) {

	sprintId := syncOps.datasource.GetJiraSprintIdFromMavenlinkTaskId(issue.MavenlinkParentTaskId)
	if len(sprintId) > 0 {
		createIssue := syncOps.issue.GenerateIssueForCreation(project, &issue, sprintId)
		if nil != createIssue {
			justCreated := syncOps.jira.CreateIssueInJira(createIssue)
			if justCreated != nil {
				saved := syncOps.datasource.SaveIssueAndTaskSyncHistory(externalProjectId, sprintId,
					issue.MavenlinkParentTaskId, issue.MavenlinkTaskId, justCreated)
				if saved == true {
					utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Check,
						fmt.Sprintf("Created issue in sprint %s and saved sync history", sprintId))
					epicTagged := syncOps.jira.UpdateEpicInfoForJiraIssue(epic.Key, justCreated.Key)
					if epicTagged {
						utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Check,
							fmt.Sprintf("Added issue '%s' to epic '%s'", justCreated.Key, epic.Fields.Summary))
					} else {
						utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Cross,
							fmt.Sprintf("FAILED to add issue '%s' to epic %s", justCreated.Key,
								epic.Fields.Summary))
					}
				} else {
					utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Check, "Created issue")
					utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Cross,
						"FAILED to save sync history")
				}
				created <- true
			} else {
				utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Cross, "FAILED to create issue")
				created <- false
			}
		} else {
			utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Cross,
				"FAILED to generate issue for creation")
			created <- false
		}
	} else {
		utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Cross, "FAILED to retrieve sprint id for issue")
		created <- false
	}
}

func (syncOps *SyncOperations) syncTasksAndSprints(externalProjectId int32,
	sprintsAndTasks *POGO.SprintAndTask) <-chan bool {
	channel := make(chan bool)
	go func() {
		if len(sprintsAndTasks.GetRapidViews()) > 0 {
			toBeCreated, toBeCreatedClosed := syncOps.sprint.PrepareSprintsForCreation(sprintsAndTasks)
			toBeSynced, toBeSyncedClosed := syncOps.sprint.PrepareSprintsForUpdate(sprintsAndTasks)

			synced := make(chan bool)
			syncedCount := 0
			var quitWaiting bool
			var creationCompleted bool
			var updateCompleted bool
			for {
				if creationCompleted && updateCompleted && quitWaiting {
					break
				}
				select {
				case toBe := <-toBeCreated:
					go syncOps.createSprint(externalProjectId, toBe, synced)
					syncedCount++
				case <-toBeCreatedClosed:
					creationCompleted = true
					if updateCompleted {
						quitWaiting = true
					}
				case toBe := <-toBeSynced:
					go syncOps.updateSprint(toBe, synced)
					syncedCount++
				case <-toBeSyncedClosed:
					updateCompleted = true
					if creationCompleted {
						quitWaiting = true
					}
				}
			}
			if syncedCount > 0 {
				utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.TriangularBulletPoint,
					fmt.Sprintf("Triggered %d sprint sync jobs", syncedCount))
			}
			for syncedIndex := 0; syncedIndex < syncedCount; syncedIndex++ {
				<-synced
			}
			if syncedCount <= 0 {
				utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Check,
					"No JIRA sprints require synchronization!")
			}
		} else {
			utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Cross,
				"No JIRA rapid views found. Rejecting sync of sprints!")
		}
		channel <- true
	}()
	return channel
}

func (syncOps *SyncOperations) syncTasksAndIssues(externalProjectId int32,
	issuesAndTasks *POGO.IssueAndTask) <-chan bool {

	channel := make(chan bool)
	go func() {

		toBeCreated, toBeCreatedClosed := syncOps.issue.PrepareIssuesForCreation(issuesAndTasks)
		toBeSynced, toBeSyncedClosed := syncOps.issue.PrepareIssuesForUpdate(issuesAndTasks)

		synced := make(chan bool)
		syncedCount := 0
		var quitWaiting bool
		var creationCompleted bool
		var updateCompleted bool
		for {
			if creationCompleted && updateCompleted && quitWaiting {
				break
			}
			select {
			case toBe := <-toBeCreated:
				go syncOps.createIssue(externalProjectId, issuesAndTasks.GetProject(), issuesAndTasks.GetEpic(), toBe, synced)
				syncedCount++
			case <-toBeCreatedClosed:
				creationCompleted = true
				if updateCompleted {
					quitWaiting = true
				}
			case toBe := <-toBeSynced:
				go syncOps.updateIssue(externalProjectId, issuesAndTasks.GetProject(), issuesAndTasks.GetEpic(), toBe, synced)
				syncedCount++
			case <-toBeSyncedClosed:
				updateCompleted = true
				if creationCompleted {
					quitWaiting = true
				}
			}
		}
		if syncedCount > 0 {
			utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.TriangularBulletPoint,
				fmt.Sprintf("Triggered %d issue sync jobs", syncedCount))
		}
		positiveSyncs := 0
		for syncedIndex := 0; syncedIndex < syncedCount; syncedIndex++ {
			if <-synced {
				positiveSyncs++
			}
		}
		if syncedCount <= 0 || positiveSyncs != syncedCount {
			utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Check,
				"No JIRA issues require synchronization!")
		}
		channel <- true
	}()
	return channel
}

func (syncOps *SyncOperations) syncWorklogsAndTimeEntries(projectId int32,
	issuesAndTasks *POGO.IssueAndTask) <-chan bool {

	channel := make(chan bool)
	go func() {
		project := syncOps.jira.GetJiraProject(projectId)
		if project == nil {
			return
		}

		toBeCreated, toBeCreatedClosed := syncOps.worklog.PrepareWorklogsForCreation(issuesAndTasks)
		toBeSynced, toBeSyncedClosed := syncOps.worklog.PrepareWorklogsForUpdate(issuesAndTasks)

		synced := make(chan bool)
		syncedCount := 0
		var quitWaiting bool
		var creationCompleted bool
		var updateCompleted bool
		for {
			if creationCompleted && updateCompleted && quitWaiting {
				break
			}
			select {
			case toBe := <-toBeCreated:
				go syncOps.createWorklogs(project, toBe, synced)
				syncedCount++
			case <-toBeCreatedClosed:
				creationCompleted = true
				if updateCompleted {
					quitWaiting = true
				}
			case toBe := <-toBeSynced:
				go syncOps.updateWorklogs(project, toBe, synced)
				syncedCount++
			case <-toBeSyncedClosed:
				updateCompleted = true
				if creationCompleted {
					quitWaiting = true
				}
			}
		}
		if syncedCount > 0 {
			utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.TriangularBulletPoint,
				fmt.Sprintf("Triggered %d worklog sync jobs", syncedCount))
		}
		for syncedIndex := 0; syncedIndex < syncedCount; syncedIndex++ {
			<-synced
		}
		if syncedCount == 0 {
			utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Check,
				"No JIRA time entries require synchronization!")
		}
		channel <- true
	}()
	return channel
}

// Check if the sync configuration is valid
func (syncOps *SyncOperations) IsAValidSyncConfiguration(syncConfiguration *datasourceCommunicator.ExternalProject) bool {
	validConfiguration := true
	validity := make(chan bool)
	go syncOps.mavenlink.DoesWorkspaceExistInMavenlink(syncConfiguration.Source2ProjectId, validity)
	go syncOps.jira.DoesProjectExistInJira(syncConfiguration.Source1ProjectId, validity)
	go syncOps.jira.DoesEpicExistInJiraProject(
		syncConfiguration.ProjectKey+"-"+fmt.Sprint(syncConfiguration.EpicId), validity)
	for i := 0; i < 3; i++ {
		validConfiguration = syncOps.common.IsContinuouslyTrue(validConfiguration, <-validity)
	}
	return validConfiguration
}

func (syncOps *SyncOperations) SyncMavenlinkToJira(externalProject *datasourceCommunicator.ExternalProject,
	success chan bool) {

	tasks := make(chan []mavenlinkCommunicator.Task)
	subTasks := make(chan []mavenlinkCommunicator.Task)
	tasksInSubTasks := make(chan []mavenlinkCommunicator.Task)
	issuesInSprints := make(chan []jiraCommunicator.Issue)
	tasksInSubTasksTimeentries := make(chan []mavenlinkCommunicator.Timeentry)
	issuesInSprintsWorklogs := make(chan []jiraCommunicator.Worklog)
	rapidViews := make(chan []jiraCommunicator.GreenhopperRapidView)
	sprints := make(chan []jiraCommunicator.Sprint)
	users := make(chan []jiraCommunicator.Author)
	sprintsAndTasks := &POGO.SprintAndTask{}
	issuesAndTasks := &POGO.IssueAndTask{}

	jiraProject := syncOps.jira.GetJiraProject(externalProject.Source1ProjectId)
	if jiraProject == nil {
		utility.GetUtilitiesSingleton().Logger.LevelTwoLog(utility.CircularBulletPoint+utility.CircularBulletPoint,
			fmt.Sprintf("Failed to find JIRA project '%d'!!", externalProject.Source1ProjectId))
		success <- false
	}
	jiraEpic := syncOps.jira.GetEpicInJiraProject(
		externalProject.ProjectKey + "-" + fmt.Sprint(externalProject.EpicId))
	if jiraEpic == nil {
		utility.GetUtilitiesSingleton().Logger.LevelTwoLog(utility.CircularBulletPoint+utility.CircularBulletPoint,
			fmt.Sprintf("Failed to find JIRA epic '%s'!!",
				externalProject.ProjectKey+"-"+fmt.Sprint(externalProject.EpicId)))
		success <- false
	}

	utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Check,
		fmt.Sprintf("'%s' JIRA project detected", jiraProject.Name))
	utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Check,
		fmt.Sprintf("'%s' JIRA epic detected", jiraEpic.Fields.Summary))
	utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.Therefore,
		"Bootstrapping project data from Mavenlink & JIRA")

	go syncOps.mavenlink.RetrieveTasksInWorkspaceWithTitle(externalProject.Source2ProjectId, tasks,
		"Construction")
	go syncOps.jira.RetrieveRapidViewsInProject(jiraProject.Key, rapidViews)
	go syncOps.jira.RetrieveSprintsInProject(jiraProject.Key, sprints)

	sprintsAndTasks.SetTasks(<-tasks)
	sprintsAndTasks.SetRapidViews(<-rapidViews)
	sprintsAndTasks.SetSprints(<-sprints)
	utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.TriangularBulletPoint, "Retrieved")
	utility.GetUtilitiesSingleton().Logger.LevelTwoLog(utility.Check, "Milestone task - 'Construction'")
	utility.GetUtilitiesSingleton().Logger.LevelTwoLog(utility.Check,
		fmt.Sprintf("Rapid views - x%d", len(sprintsAndTasks.GetRapidViews())))
	utility.GetUtilitiesSingleton().Logger.LevelTwoLog(utility.Check,
		fmt.Sprintf("Sprints - x%d", len(sprintsAndTasks.GetSprints())))

	if sprintsAndTasks.GetRapidViews() == nil ||
		len(sprintsAndTasks.GetRapidViews()) <= 0 ||
		sprintsAndTasks.GetTasks() == nil ||
		len(sprintsAndTasks.GetTasks()) <= 0 {
		utility.GetUtilitiesSingleton().Logger.LevelTwoLog(utility.CircularBulletPoint+utility.CircularBulletPoint,
			"Failed to find valid JIRA RapidViews or Tasks !!")
		success <- false
	}
	taskIdInt64, taskIdInt64Err := strconv.ParseInt(sprintsAndTasks.GetTasks()[0].Id, 10, 32)
	if taskIdInt64Err != nil {
		utility.GetUtilitiesSingleton().Logger.LevelTwoLog(utility.CircularBulletPoint+utility.CircularBulletPoint,
			"Failed to convert task id !!")
		success <- false
	}

	go syncOps.mavenlink.RetrieveSubTasksInWorkspace(externalProject.Source2ProjectId, int32(taskIdInt64),
		subTasks)
	sprintsAndTasks.SetSubTasks(<-subTasks)

	go syncOps.retrieveAndCollateMavenlinkTasksInSubTasks(externalProject, sprintsAndTasks.GetSubTasks(),
		tasksInSubTasks)
	go syncOps.retrieveAndCollateJiraTasksInSprints(jiraProject, sprintsAndTasks.GetSprints(), issuesInSprints)
	go syncOps.jira.GetUsersInProject(jiraProject.Key, users)

	issuesAndTasks.SetProject(jiraProject)
	issuesAndTasks.SetEpic(jiraEpic)
	issuesAndTasks.SetUsers(<-users)
	issuesAndTasks.SetIssues(<-issuesInSprints)
	issuesAndTasks.SetTasks(<-tasksInSubTasks)

	utility.GetUtilitiesSingleton().Logger.LevelOneLog(utility.TriangularBulletPoint, "Prepared object with")
	utility.GetUtilitiesSingleton().Logger.LevelTwoLog(utility.Check, fmt.Sprintf("Project - %s",
		issuesAndTasks.GetProject().Name))
	utility.GetUtilitiesSingleton().Logger.LevelTwoLog(utility.Check, fmt.Sprintf(
		"Users - x%d", len(issuesAndTasks.GetUsers())))
	utility.GetUtilitiesSingleton().Logger.LevelTwoLog(utility.Check, fmt.Sprintf(
		"Mavenlink sub-tasks - x%d", len(sprintsAndTasks.GetSubTasks())))
	utility.GetUtilitiesSingleton().Logger.LevelTwoLog(utility.Check, fmt.Sprintf(
		"JIRA issues - x%d", len(issuesAndTasks.GetIssues())))
	utility.GetUtilitiesSingleton().Logger.LevelTwoLog(utility.Check, fmt.Sprintf(
		"Mavenlink tasks - x%d", len(issuesAndTasks.GetTasks())))

	timeentriesCount := 0
	worklogsCount := 0
	for _, issueInSprint := range issuesAndTasks.GetIssues() {
		go syncOps.jira.GetWorklogsFromIssue(issueInSprint.Key, issuesInSprintsWorklogs)
		worklogsCount++
	}
	for _, taskInSubTask := range issuesAndTasks.GetTasks() {
		go syncOps.mavenlink.GetTimeEntriesForIssueTask(externalProject.Source2ProjectId, taskInSubTask.Id,
			tasksInSubTasksTimeentries)
		timeentriesCount++
	}
	for i := 0; i < worklogsCount; i++ {
		for _, worklog := range <-issuesInSprintsWorklogs {
			issuesAndTasks.AddWorklog(worklog)
		}
	}
	for i := 0; i < timeentriesCount; i++ {
		for _, timeEntry := range <-tasksInSubTasksTimeentries {
			issuesAndTasks.AddTimeentry(timeEntry)
		}
	}
	utility.GetUtilitiesSingleton().Logger.LevelZeroLog(utility.SeparationBlock, "")
	completedSprintSync := syncOps.syncTasksAndSprints(externalProject.Id, sprintsAndTasks)
	<-completedSprintSync
	utility.GetUtilitiesSingleton().Logger.LevelZeroLog(utility.SeparationBlock, "")
	completedIssueSync := syncOps.syncTasksAndIssues(externalProject.Id, issuesAndTasks)
	<-completedIssueSync
	utility.GetUtilitiesSingleton().Logger.LevelZeroLog(utility.SeparationBlock, "")
	completedIssueWorklogSync := syncOps.syncWorklogsAndTimeEntries(externalProject.Source1ProjectId, issuesAndTasks)
	<-completedIssueWorklogSync
	utility.GetUtilitiesSingleton().Logger.LevelZeroLog(utility.SeparationBlock, "")
	success <- true
}
