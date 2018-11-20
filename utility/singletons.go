// Copyright Costrategix Technologies Pvt. Ltd.
// All Rights Reserved 2018

// Provide utility methods for common functionality
package utility

import (
	jiraCommunicator "github.com/desertjinn/jira-communicator/proto/jira-communicator"
	mavenlinkCommunicator "github.com/desertjinn/mavenlink-communicator/proto/mavenlink-communicator"
	mavenlinkJiraDatasource "github.com/desertjinn/mavenlink-jira-datasource/proto/mavenlink-jira-datasource"
	microclient "github.com/micro/go-micro/client"
	"golang.org/x/net/context"
	"sync"
	"time"
)

const (
	MavenlinkService  = "costrategix.service.mavenlink.communicator"
	JiraService       = "costrategix.service.jira.communicator"
	DatasourceService = "costrategix.service.mavenlink.jira.datasource"
)

const (
	ProgressBlock         = "▰"
	EmptyProgressBlock    = "▱"
	LevelOne              = "\t"
	LevelTwo              = "\t\t"
	EntryPoint            = "↳"
	CircularBulletPoint   = "☀"
	TriangularBulletPoint = "‣"
	Check                 = "✓"
	Cross                 = "❌"
	Biohazard             = "☣"
	Warning               = "⚠"
	BottomRight           = "↙"
	Refresh               = "↻"
	Because               = "∵"
	Therefore             = "∴"
	ThumbsUp              = "👍"
	ThumbsDown            = "👎"
	ErrorBlock            = "❌ ❌ ❌ ❌ ❌ ❌ ❌ ❌ ❌ ❌ ❌ ❌ ❌ ❌ ❌"
	EndBlock              = "■ ■ ■ ■ ■ ■ ■ ■ ■ ■ ■ ■ ■ ■"
	SeparationBlock       = "‣ ‣ ‣ ‣ ‣ ‣ ‣ ‣ ‣ ‣ ‣ ‣ ‣ ‣"
)

var once sync.Once
var utilities *Utilities
var logging *FormatLog

// A struct of reusable single instance functionality
// provided using a Singleton pattern
type Utilities struct {
	Logger                     FormatLogInterface
	CommsContext               context.Context
	CommsContextCancel         context.CancelFunc
	MavenlinkClient            mavenlinkCommunicator.MavenlinkCommunicatorClient
	JiraClient                 jiraCommunicator.JiraCommunicatorClient
	ConfigurationDatasource    mavenlinkJiraDatasource.MavenlinkJiraDatasourceClient
	MavenlinkToJiraEquivalence map[string]map[string][]string
	ProjectId                  string
	JiraIssueTypes             []*jiraCommunicator.IssueType
	JiraStatuses               []*jiraCommunicator.Status
	JiraPriorities             []*jiraCommunicator.Priority
}

// Retrieve the Utilities singleton struct
func GetUtilitiesSingleton() *Utilities {
	once.Do(func() {
		logging.LevelZeroLog(EntryPoint, "Initialising singletons...")
		logging.LevelOneLog(CircularBulletPoint, "Triggering Level 1 singletons...")
		temp := initialiseUtilities()
		logging.LevelOneLog(Check, "Initialised Level 1")
		logging.LevelOneLog(CircularBulletPoint, "Triggering Level 2 singletons...")
		temp = initialiseAdditionalUtilities(temp)
		utilities = &temp
		logging.LevelOneLog(Check, "Initialised Level 2")
		logging.LevelZeroLog(SeparationBlock, "")
	})
	return utilities
}

// Initialise the Utilities singleton
func initialiseUtilities() Utilities {
	commsContext, commsContextCancel := getContext()
	return Utilities{
		Logger:                     getLogger(),
		CommsContext:               commsContext,
		CommsContextCancel:         commsContextCancel,
		MavenlinkClient:            getMavenlinkCommunicator(),
		JiraClient:                 getJiraCommunicator(),
		ConfigurationDatasource:    getConfigurationDatasource(),
		MavenlinkToJiraEquivalence: getMavenlinkToJiraEquivalence(),
	}
}

// Initialise the 2nd level of the Utilities singleton
func initialiseAdditionalUtilities(utilities Utilities) Utilities {
	utilities.JiraIssueTypes = getJiraIssueTypeMetadata(utilities.JiraClient, utilities.CommsContext)
	utilities.JiraStatuses = getJiraStatusMetadata(utilities.JiraClient, utilities.CommsContext)
	utilities.JiraPriorities = getJiraPriorityMetadata(utilities.JiraClient, utilities.CommsContext)

	return utilities
}

// Retrieve the context used in various calls
func getContext() (context.Context, context.CancelFunc) {
	//return context.WithCancel(context.Background())
	return context.WithTimeout(context.Background(), time.Second*300)
}

// Retrieve a communicator instance of the datasource for sync configurations
func getConfigurationDatasource() mavenlinkJiraDatasource.MavenlinkJiraDatasourceClient {
	return mavenlinkJiraDatasource.NewMavenlinkJiraDatasourceClient(DatasourceService,
		microclient.DefaultClient)
}

// Retrieve a Mavenlink communicator instance
func getMavenlinkCommunicator() mavenlinkCommunicator.MavenlinkCommunicatorClient {
	return mavenlinkCommunicator.NewMavenlinkCommunicatorClient(MavenlinkService,
		microclient.DefaultClient)
}

// Retrieve a JIRA communicator instance
func getJiraCommunicator() jiraCommunicator.JiraCommunicatorClient {
	return jiraCommunicator.NewJiraCommunicatorClient(JiraService, microclient.DefaultClient)
}

// Retrieve a logger instance
func getLogger() FormatLogInterface {
	formatLog := new(FormatLog)
	return formatLog
}

// Retrieve the issue types equivalence relation between Mavenlink & JIRA(Mavenlink -> JIRA)
func GetMavenlinkToJiraIssueTypesEquivalence() (issueRelations map[string][]string) {
	issueRelations = make(map[string][]string)
	jiraTaskType := []string{"new feature", "task", "improvement", "provisioining", "sub-task", "performance",
		"support", "epic", "story", "technical task", "fulfillment", "seo", "promotion", "test"}
	issueRelations["task"] = jiraTaskType

	jiraIssueType := []string{"development bug", "bug", "defect"}
	issueRelations["issue"] = jiraIssueType

	return issueRelations
}

// Retrieve the status equivalence relation between Mavenlink & JIRA(Mavenlink -> JIRA)
func GetMavenlinkToJiraStatusesEquivalence() (statusRelations map[string][]string) {
	statusRelations = make(map[string][]string)

	statusRelations["not started"] = []string{"open"}
	statusRelations["new"] = []string{"open"}

	jiraInProgressStatus := []string{"in progress", "reopened", "review"}
	statusRelations["started"] = jiraInProgressStatus
	statusRelations["in progress"] = jiraInProgressStatus

	jiraFixedStatus := []string{"internal production validation",
		"internal staging validation", "Internal qa", "approved for prod", "approved for stage"}
	statusRelations["fixed"] = jiraFixedStatus

	statusRelations["reopened"] = []string{"reopened"}

	statusRelations["resolved"] = []string{"resolved"}

	statusRelations["completed"] = []string{"closed"}
	statusRelations["duplicate"] = []string{"closed"}
	statusRelations["can't repro"] = []string{"closed"}
	statusRelations["won't fix"] = []string{"closed"}

	statusRelations["needs info"] = []string{"require feedback"}
	statusRelations["blocked"] = []string{"require feedback"}

	return statusRelations
}

// Retrieve the status equivalence relation between Mavenlink & JIRA(Mavenlink -> JIRA)
func GetMavenlinkToJiraPrioritiesEquivalence() (prioritiesRelations map[string][]string) {
	prioritiesRelations = make(map[string][]string)
	prioritiesRelations["high"] = []string{"major"}

	jiraCriticalPriority := []string{"blocker", "critical", "roadBlocked"}
	prioritiesRelations["critical"] = jiraCriticalPriority

	prioritiesRelations["normal"] = []string{"minor"}

	prioritiesRelations["low"] = []string{"trivial"}

	return prioritiesRelations
}

// Retrieve the equivalence relations between Mavenlink & JIRA(Mavenlink -> JIRA)
func getMavenlinkToJiraEquivalence() (relations map[string]map[string][]string) {
	relations = make(map[string]map[string][]string)
	relations["IssueType"] = GetMavenlinkToJiraIssueTypesEquivalence()
	relations["Status"] = GetMavenlinkToJiraStatusesEquivalence()
	relations["Priority"] = GetMavenlinkToJiraPrioritiesEquivalence()
	return relations
}

func getJiraStatusMetadata(jiraClient jiraCommunicator.JiraCommunicatorClient,
	commsContext context.Context) (statuses []*jiraCommunicator.Status) {

	var request jiraCommunicator.Request
	response, err := jiraClient.GetIssueStatuses(commsContext, &request)
	if nil == err && nil == response.Error && nil != response.Statuses {
		for _, status := range response.Statuses {
			statuses = append(statuses, status)
		}
	}
	return statuses
}

func getJiraPriorityMetadata(jiraClient jiraCommunicator.JiraCommunicatorClient,
	commsContext context.Context) (priorities []*jiraCommunicator.Priority) {

	var request jiraCommunicator.Request
	response, err := jiraClient.GetIssuePriorities(commsContext, &request)
	if nil == err && nil == response.Error && nil != response.Priorities {
		for _, priority := range response.Priorities {
			priorities = append(priorities, priority)
		}
	}
	return priorities
}

func getJiraIssueTypeMetadata(jiraClient jiraCommunicator.JiraCommunicatorClient,
	commsContext context.Context) (issueTypes []*jiraCommunicator.IssueType) {

	var request jiraCommunicator.Request
	response, err := jiraClient.GetIssueTypes(commsContext, &request)
	if nil == err && nil == response.Error && nil != response.IssueTypes {
		for _, issueType := range response.IssueTypes {
			issueTypes = append(issueTypes, issueType)
		}
	}
	return issueTypes
}
