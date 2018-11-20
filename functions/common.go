package functions

import (
	"fmt"
	jiraCommunicator "github.com/desertjinn/jira-communicator/proto/jira-communicator"
	synchronizer "github.com/desertjinn/mavenlink-jira-sync/proto/mavenlink-jira-sync"
	"github.com/desertjinn/mavenlink-jira-sync/utility"
	"strconv"
	"strings"
	"time"
)

type CommonFunctionsInterface interface {
	FloatToString(input float64) string
	GetPercentOfCompletion(current int, length int) float64
	GetProgressBar(current int, length int) string
	StringInSlice(a string, list []string) bool
	IsContinuouslyTrue(current bool, next bool) bool
	FilterString(ss []string, test func(string) bool) (ret []string)
	MatchFirst(ss []string, test func(string) bool) bool
	GetIdFromString(stringId string) int32
	ParseMavenlinkDateToJiraDate(mavenlinkDate string, time string) string
	ParseJiraDateToMavenlinkDate(jiraDate string) string
	ParseDateForInsertingInDb(aDate string) string
	ChangeDetected(existing string, detected string, mavenlink string, equivalenceType *synchronizer.EquivalenceTypes) bool
	IsEquivalentToJira(jira string, mavenlink string, equivalenceType *synchronizer.EquivalenceTypes) bool
	GetDefaultEquivalentJiraIssueType() (equivalentIssueType *jiraCommunicator.IssueType)
	GetDefaultEquivalentJiraIssueStatus() (equivalentIssueStatus *jiraCommunicator.Status)
	GetDefaultEquivalentJiraIssuePriority() (equivalentIssuePriority *jiraCommunicator.Priority)
	GetJiraIssueTypeFromMetadata(mavenlinkIssueTypeName string, existingJiraIssueType string) (detectedIssueType *jiraCommunicator.IssueType)
	GetJiraStatusFromMetadata(mavenlinkStatusName string, existingJiraStatus string) (detectedStatus *jiraCommunicator.Status)
	GetJiraPriorityFromMetadata(mavenlinkPriorityName string, existingJiraPriority string) (detectedPriority *jiraCommunicator.Priority)
}

type CommonFunctions struct {}



// Convert a date from the described layout to the desired format
func convertDateWithLayoutToFormat(date string, layout string, format string) string {
	var dateString string
	if len(date) > 0 {
		dateObject, dateErr := time.Parse(layout, date)
		if dateErr != nil {
			return dateString
		}
		dateString = dateObject.Format(format)
	}
	return dateString
}


func determineEquivalenceType(equivalenceType *synchronizer.EquivalenceTypes) (equivalence map[string][]string) {
	if equivalenceType.IssueType {
		return utility.GetMavenlinkToJiraIssueTypesEquivalence()
	} else if equivalenceType.Status {
		return utility.GetMavenlinkToJiraStatusesEquivalence()
	} else if equivalenceType.Priority {
		return utility.GetMavenlinkToJiraPrioritiesEquivalence()
	}
	return
}

func mavenlinkAndJiraMatchRules(jira string, mavenlink string) bool {
	// Issue types
	if strings.EqualFold(mavenlink, "issue") {
		if strings.EqualFold(jira, "Bug") {
			return true
		}
	}
	if strings.EqualFold(mavenlink, "task") {
		return strings.EqualFold(jira, mavenlink)
	}
	// Statuses
	if strings.EqualFold(mavenlink, "not started") {
		if strings.EqualFold(jira, "open") {
			return true
		}
	}
	if strings.EqualFold(mavenlink, "started") {
		if strings.EqualFold(jira, "in progress") {
			return true
		}
	}
	if strings.EqualFold(mavenlink, "completed") {
		if strings.EqualFold(jira, "closed") {
			return true
		}
	}
	if strings.EqualFold(mavenlink, "needs info") {
		if strings.EqualFold(jira, "Require Feedback") {
			return true
		}
	}
	// Priorities
	if strings.EqualFold(mavenlink, "high") {
		if strings.EqualFold(jira, "major") {
			return true
		}
	}
	if strings.EqualFold(mavenlink, "critical") {
		if strings.EqualFold(jira, "blocker") {
			return true
		}
	}
	if strings.EqualFold(mavenlink, "normal") {
		if strings.EqualFold(jira, "minor") {
			return true
		}
	}
	if strings.EqualFold(mavenlink, "low") {
		if strings.EqualFold(jira, "trivial") {
			return true
		}
	}
	return false
}

// Get string of a float
func (cf *CommonFunctions) FloatToString(input float64) string {
	// to convert a float number to a string with the fewest digits necessary to accurately represent the float
	return strconv.FormatFloat(input, 'f', -1, 64)
}

// Get percent of completion
func (cf *CommonFunctions) GetPercentOfCompletion(current int, length int) float64 {
	return float64(current) / float64(length) * 100
}

// Get progress bar based on length and current status
func (cf *CommonFunctions) GetProgressBar(current int, length int) string {
	var progressBar string
	var i int
	for i = 0; i < length; i++ {
		if i > current {
			progressBar += utility.EmptyProgressBlock
		} else {
			progressBar += utility.ProgressBlock
		}
	}
	percentage := cf.FloatToString(cf.GetPercentOfCompletion(current+1, length))
	return progressBar + " " + percentage + "%"
}

// Check if string is present in a slice
func (cf *CommonFunctions) StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if strings.EqualFold(a, b) {
			return true
		}
	}
	return false
}

// Check the truth of the provided values and return false if either of them are false
func (cf *CommonFunctions) IsContinuouslyTrue(current bool, next bool) bool {
	var status bool
	if current == true {
		if next == true {
			status = true
		}
	}
	return status
}

func (cf *CommonFunctions) FilterString(ss []string, test func(string) bool) (ret []string) {
	for _, s := range ss {
		if test(s) {
			ret = append(ret, s)
		}
	}
	return
}

func (cf *CommonFunctions) MatchFirst(ss []string, test func(string) bool) bool {
	for _, s := range ss {
		if test(s) {
			return true
		}
	}
	return false
}

func (cf *CommonFunctions) GetIdFromString(stringId string) int32 {
	idInt64, idInt64Err := strconv.ParseInt(stringId, 10, 32)
	if idInt64Err == nil {
		return int32(idInt64)
	}
	return 0
}

// Convert a Mavenlink date(yyyy-MM-dd) layout to JIRA date(dd/MMM/yy h:m a) format
func (cf *CommonFunctions) ParseMavenlinkDateToJiraDate(mavenlinkDate string, time string) string {
	mavenlinkDateFormatLayout := "2006-01-02"
	if len(time) > 0 {
		mavenlinkDate = fmt.Sprintf("%s %s", mavenlinkDate, time)
		mavenlinkDateFormatLayout = "2006-01-02 03:04:05"
	}
	return convertDateWithLayoutToFormat(mavenlinkDate, mavenlinkDateFormatLayout, "02/Jan/06 3:04 PM")
}

// Convert a JIRA date(dd/MMM/yy h:m a) layout to Mavenlink date(yyyy-MM-dd) format
func (cf *CommonFunctions)  ParseJiraDateToMavenlinkDate(jiraDate string) string {
	return convertDateWithLayoutToFormat(jiraDate, "02/Jan/06 3:04 PM", "2006-01-02")
}

// Convert an extracted MySQL date(yyyy-MM-ddTh:m:sZ) layout to MySQL date(yyyy-MM-dd h:m:s) format for insertion
func (cf *CommonFunctions) ParseDateForInsertingInDb(aDate string) string {
	return convertDateWithLayoutToFormat(aDate, "2006-01-02T03:04:05Z", "2006-01-02 03:04:05")
}

func (cf *CommonFunctions) ChangeDetected(existing string, detected string, mavenlink string,
	equivalenceType *synchronizer.EquivalenceTypes) bool {
	equivalence := determineEquivalenceType(equivalenceType)
	if jiraIssueTypes, ok := equivalence[mavenlink]; ok {
		if !strings.EqualFold(existing, detected) && !cf.StringInSlice(existing, jiraIssueTypes) {
			return true
		}
	}
	return false
}

func (cf *CommonFunctions) IsEquivalentToJira(jira string, mavenlink string,
	equivalenceType *synchronizer.EquivalenceTypes) bool {

	equivalence := determineEquivalenceType(equivalenceType)
	if jiraIssueTypes, ok := equivalence[mavenlink]; ok {
		return cf.StringInSlice(jira, jiraIssueTypes)
	}
	return false
}

func (cf *CommonFunctions) GetDefaultEquivalentJiraIssueType() (equivalentIssueType *jiraCommunicator.IssueType) {
	for _, issuetype := range utility.GetUtilitiesSingleton().JiraIssueTypes {
		if cf.IsEquivalentToJira(issuetype.Name, "task", &synchronizer.EquivalenceTypes{IssueType: true}) {
			if mavenlinkAndJiraMatchRules(issuetype.Name, "task") {
				return issuetype
			}
		}
	}
	return
}

func (cf *CommonFunctions) GetDefaultEquivalentJiraIssueStatus() (equivalentIssueStatus *jiraCommunicator.Status) {
	for _, issueStatus := range utility.GetUtilitiesSingleton().JiraStatuses {
		if cf.IsEquivalentToJira(issueStatus.Name, "not started",
			&synchronizer.EquivalenceTypes{Status: true}) {

			return issueStatus
		}
	}
	return
}

func (cf *CommonFunctions) GetDefaultEquivalentJiraIssuePriority() (equivalentIssuePriority *jiraCommunicator.Priority) {
	for _, priority := range utility.GetUtilitiesSingleton().JiraPriorities {
		if cf.IsEquivalentToJira(priority.Name, "high", &synchronizer.EquivalenceTypes{Priority: true}) {
			return priority
		}
	}
	return
}

func (cf *CommonFunctions) getEquivalentJiraIssuePriority(mavenlinkPriorityName string) (
	equivalentIssuePriority *jiraCommunicator.Priority) {

	for _, priority := range utility.GetUtilitiesSingleton().JiraPriorities {
		if cf.IsEquivalentToJira(priority.Name, mavenlinkPriorityName, &synchronizer.EquivalenceTypes{Priority: true}) {
			return priority
		}
	}
	return
}


func (cf *CommonFunctions) getEquivalentJiraIssueType(mavenlinkIssueTypeName string) (
	equivalentIssueType *jiraCommunicator.IssueType) {

	for _, issueType := range utility.GetUtilitiesSingleton().JiraIssueTypes {
		if cf.IsEquivalentToJira(issueType.Name, mavenlinkIssueTypeName, &synchronizer.EquivalenceTypes{IssueType: true}) {
			if mavenlinkAndJiraMatchRules(issueType.Name, mavenlinkIssueTypeName) {
				return issueType
			}
		}
	}
	return
}

func (cf *CommonFunctions) getEquivalentJiraIssueStatus(mavenlinkStatusName string) (
	equivalentIssueStatus *jiraCommunicator.Status) {

	for _, issueStatus := range utility.GetUtilitiesSingleton().JiraStatuses {
		if cf.IsEquivalentToJira(issueStatus.Name, mavenlinkStatusName, &synchronizer.EquivalenceTypes{Status: true}) {
			return issueStatus
		}
	}
	return
}

// Retrieve JIRA's issue type from Mavenlink task's StoryType value
func (cf *CommonFunctions) GetJiraIssueTypeFromMetadata(mavenlinkIssueTypeName string,
	existingJiraIssueType string) (detectedIssueType *jiraCommunicator.IssueType) {

	detectedIssueType = cf.getEquivalentJiraIssueType(mavenlinkIssueTypeName)
	if detectedIssueType != nil {
		if len(existingJiraIssueType) == 0 {
			return detectedIssueType
		} else {
			if cf.ChangeDetected(existingJiraIssueType, detectedIssueType.Name, mavenlinkIssueTypeName,
				&synchronizer.EquivalenceTypes{IssueType: true}) {

				return detectedIssueType
			}
		}
	}
	return detectedIssueType
}

// Retrieve JIRA's status from Mavenlink task's State value
func (cf *CommonFunctions) GetJiraStatusFromMetadata(mavenlinkStatusName string,
	existingJiraStatus string) (detectedStatus *jiraCommunicator.Status) {

	detectedStatus = cf.getEquivalentJiraIssueStatus(mavenlinkStatusName)
	if detectedStatus != nil {
		if len(existingJiraStatus) == 0 {
			return detectedStatus
		} else {
			if cf.ChangeDetected(existingJiraStatus, detectedStatus.Name, mavenlinkStatusName,
				&synchronizer.EquivalenceTypes{Status: true}) {
				return detectedStatus
			}
		}
	}
	return detectedStatus
}

// Retrieve JIRA's priority from Mavenlink task's priority value
func (cf *CommonFunctions) GetJiraPriorityFromMetadata(mavenlinkPriorityName string,
	existingJiraPriority string) (detectedPriority *jiraCommunicator.Priority) {

	detectedPriority = cf.getEquivalentJiraIssuePriority(mavenlinkPriorityName)
	if detectedPriority != nil {
		if len(existingJiraPriority) == 0 {
			return detectedPriority
		} else {
			if cf.ChangeDetected(existingJiraPriority, detectedPriority.Name, mavenlinkPriorityName,
				&synchronizer.EquivalenceTypes{Priority: true}) {

				return detectedPriority
			}
		}
	}
	return detectedPriority
}
