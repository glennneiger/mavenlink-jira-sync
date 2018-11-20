package main

import (
	"fmt"
	"git.costrategix.net/go/mavenlink-jira-sync/functions"
	synchronizer "git.costrategix.net/go/mavenlink-jira-sync/proto/mavenlink-jira-sync"
	"git.costrategix.net/go/mavenlink-jira-sync/services"
	"git.costrategix.net/go/mavenlink-jira-sync/utility"
	"github.com/kelseyhightower/envconfig"
	"github.com/micro/go-micro/cmd"
	"sync"
)

func main() {
	cmd.Init()

	var env synchronizer.EnvironmentConfiguration
	// Retrieve environment configuration
	err := envconfig.Process("mavenlinkCommunicator-jiraCommunicator-sync", &env)
	if err != nil {
		utility.GetUtilitiesSingleton().Logger.LevelZeroLog(utility.Warning,
			"Failure processing environment configuration")
		utility.GetUtilitiesSingleton().Logger.LevelZeroLog(utility.Cross,
			fmt.Sprintf("Error → %v", err))
		utility.GetUtilitiesSingleton().Logger.LevelZeroLog(utility.ErrorBlock, "")
	}

	var wg sync.WaitGroup
	dataSourceService := new(services.DataSourceService)
	commonFunctions := new(functions.CommonFunctions)
	syncOperations := SyncOperations{
		common:     commonFunctions,
		sprint:     new(functions.SprintFunctions),
		issue:      new(functions.IssueFunctions),
		worklog:    new(functions.WorklogFunctions),
		datasource: dataSourceService,
		jira:       new(services.JiraService),
		mavenlink:  new(services.MavenlinkService),
	}

	wg.Add(1)
	syncConfigurations, err := dataSourceService.GetSyncConfiguration()
	if err != nil {
		utility.GetUtilitiesSingleton().Logger.LevelZeroLog(utility.Warning,
			"Failure processing sync configurations")
		utility.GetUtilitiesSingleton().Logger.LevelZeroLog(utility.Warning,
			fmt.Sprintf("Error → %v", err))
		utility.GetUtilitiesSingleton().Logger.LevelZeroLog(utility.ErrorBlock, "")
	}
	utility.GetUtilitiesSingleton().Logger.LevelZeroLog(utility.CircularBulletPoint,
		fmt.Sprintf("Found %d sync configurations", len(syncConfigurations)))
	wg.Done()

	utility.GetUtilitiesSingleton().Logger.LevelZeroLog(utility.TriangularBulletPoint,
		"Syncing Mavenlink →→ JIRA")
	success := make(chan bool)
	var startedSyncing int
	for syncConfigurationKey, syncConfiguration := range syncConfigurations {
		utility.GetUtilitiesSingleton().Logger.LevelZeroLog(
			utility.TriangularBulletPoint+utility.TriangularBulletPoint,
			fmt.Sprintf("Processing configuration No.%d: %s",
				syncConfigurationKey+1, syncConfiguration.ProjectName))
		validConfiguration := syncOperations.IsAValidSyncConfiguration(syncConfiguration)
		if validConfiguration == true {
			utility.GetUtilitiesSingleton().Logger.LevelZeroLog(utility.Check, fmt.Sprintf(
				"Configuration for '%s' is valid", syncConfiguration.ProjectName))
			go syncOperations.SyncMavenlinkToJira(syncConfiguration, success)
			startedSyncing++
		} else {
			utility.GetUtilitiesSingleton().Logger.LevelZeroLog(utility.Cross,
				fmt.Sprintf("Configuration is invalid for '%s'", syncConfiguration.ProjectName))
			utility.GetUtilitiesSingleton().Logger.LevelZeroLog(utility.BottomRight, "Checking next")
		}
		if startedSyncing > 0 {
			for i := 0; i < startedSyncing; i++ {
				if true == <-success {
					utility.GetUtilitiesSingleton().Logger.LevelZeroLog(utility.Check,
						fmt.Sprintf("Successfully synced '%s'", syncConfiguration.ProjectName))
				} else {
					utility.GetUtilitiesSingleton().Logger.LevelZeroLog(utility.Cross,
						fmt.Sprintf("Failed to successfully sync '%s'", syncConfiguration.ProjectName))
				}
			}
		}
		utility.GetUtilitiesSingleton().Logger.LevelZeroLog(utility.Check,
			fmt.Sprintf("%s Completed", commonFunctions.GetProgressBar(syncConfigurationKey,
				len(syncConfigurations))))
		utility.GetUtilitiesSingleton().Logger.LevelZeroLog(utility.EndBlock, "")
	}
	wg.Wait()
	utility.GetUtilitiesSingleton().Logger.LevelZeroLog(utility.ThumbsUp, "Completed sync operation")
}
