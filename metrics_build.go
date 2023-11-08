package main

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/webdevops/go-common/prometheus/collector"
	"go.uber.org/zap"

	devopsClient "github.com/webdevops/azure-devops-exporter/azure-devops-client"
)

var cloudDevBuildId int64
var POSDevBuildId int64

type MetricsCollectorBuild struct {
	collector.Processor

	prometheus struct {
		build       *prometheus.GaugeVec
		buildStatus *prometheus.GaugeVec

		buildDefinition *prometheus.GaugeVec

		buildStage *prometheus.GaugeVec
		buildPhase *prometheus.GaugeVec
		buildJob   *prometheus.GaugeVec
		buildTask  *prometheus.GaugeVec

		buildLineCodeCoverage   *prometheus.GaugeVec
		buildBranchCodeCoverage *prometheus.GaugeVec

		buildTimeProject *prometheus.SummaryVec
		jobTimeProject   *prometheus.SummaryVec
	}
}

func (m *MetricsCollectorBuild) Setup(collector *collector.Collector) {
	m.Processor.Setup(collector)

	m.prometheus.build = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azure_devops_build_info",
			Help: "Azure DevOps build",
		},
		[]string{
			"projectID",
			"buildDefinitionID",
			"buildID",
			"agentPoolID",
			"requestedBy",
			"buildNumber",
			"buildName",
			"sourceBranch",
			"sourceVersion",
			"status",
			"reason",
			"result",
			"url",
		},
	)
	m.Collector.RegisterMetricList("build", m.prometheus.build, true)

	m.prometheus.buildLineCodeCoverage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azure_devops_build_line_code_coverage",
			Help: "Azure DevOps build line code coverage",
		},
		[]string{
			"buildID",
			"linesCoverable",
			"linesCovered",
			"pipelineName",
		},
	)
	m.Collector.RegisterMetricList("buildLineCodeCoverage", m.prometheus.buildLineCodeCoverage, true)

	m.prometheus.buildBranchCodeCoverage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azure_devops_build_branch_code_coverage",
			Help: "Azure DevOps build branch code coverage",
		},
		[]string{
			"buildID",
			"branchCoverable",
			"branchCovered",
			"pipelineName",
		},
	)
	m.Collector.RegisterMetricList("buildBranchCodeCoverage", m.prometheus.buildBranchCodeCoverage, true)

	m.prometheus.buildStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azure_devops_build_status",
			Help: "Azure DevOps build",
		},
		[]string{
			"projectID",
			"buildID",
			"buildDefinitionID",
			"buildNumber",
			"result",
			"type",
		},
	)
	m.Collector.RegisterMetricList("buildStatus", m.prometheus.buildStatus, true)

	m.prometheus.buildStage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azure_devops_build_stage",
			Help: "Azure DevOps build stages",
		},
		[]string{
			"projectID",
			"buildID",
			"buildDefinitionID",
			"buildNumber",
			"name",
			"id",
			"identifier",
			"result",
			"type",
		},
	)
	m.Collector.RegisterMetricList("buildStage", m.prometheus.buildStage, true)

	m.prometheus.buildPhase = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azure_devops_build_phase",
			Help: "Azure DevOps build phases",
		},
		[]string{
			"projectID",
			"buildID",
			"buildDefinitionID",
			"buildNumber",
			"name",
			"id",
			"parentId",
			"identifier",
			"result",
			"type",
		},
	)
	m.Collector.RegisterMetricList("buildPhase", m.prometheus.buildPhase, true)

	m.prometheus.buildJob = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azure_devops_build_job",
			Help: "Azure DevOps build jobs",
		},
		[]string{
			"projectID",
			"buildID",
			"buildDefinitionID",
			"buildNumber",
			"name",
			"id",
			"parentId",
			"identifier",
			"result",
			"type",
		},
	)
	m.Collector.RegisterMetricList("buildJob", m.prometheus.buildJob, true)

	m.prometheus.buildTask = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azure_devops_build_task",
			Help: "Azure DevOps build tasks",
		},
		[]string{
			"projectID",
			"buildID",
			"buildDefinitionID",
			"buildNumber",
			"name",
			"id",
			"parentId",
			"workerName",
			"result",
			"type",
		},
	)
	m.Collector.RegisterMetricList("buildTask", m.prometheus.buildTask, true)

	m.prometheus.buildDefinition = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azure_devops_build_definition_info",
			Help: "Azure DevOps build definition",
		},
		[]string{
			"projectID",
			"buildDefinitionID",
			"buildNameFormat",
			"buildDefinitionName",
			"path",
			"url",
		},
	)
	m.Collector.RegisterMetricList("buildDefinition", m.prometheus.buildDefinition, true)
}

func (m *MetricsCollectorBuild) Reset() {}

func (m *MetricsCollectorBuild) Collect(callback chan<- func()) {
	ctx := m.Context()
	logger := m.Logger()

	for _, project := range AzureDevopsServiceDiscovery.ProjectList() {
		projectLogger := logger.With(zap.String("project", project.Name))
		m.collectDefinition(ctx, projectLogger, callback, project)
		m.collectBuilds(ctx, projectLogger, callback, project)
		m.collectBuildsTimeline(ctx, projectLogger, callback, project)
		m.collectBuildCodeCoverage(ctx, projectLogger, callback, project)
	}
}

func (m *MetricsCollectorBuild) collectDefinition(ctx context.Context, logger *zap.SugaredLogger, callback chan<- func(), project devopsClient.Project) {
	list, err := AzureDevopsClient.ListBuildDefinitions(project.Id)
	if err != nil {
		logger.Error(err)
		return
	}

	buildDefinitonMetric := m.Collector.GetMetricList("buildDefinition")

	for _, buildDefinition := range list.List {
		buildDefinitonMetric.Add(prometheus.Labels{
			"projectID":           project.Id,
			"buildDefinitionID":   int64ToString(buildDefinition.Id),
			"buildNameFormat":     buildDefinition.BuildNameFormat,
			"buildDefinitionName": buildDefinition.Name,
			"path":                buildDefinition.Path,
			"url":                 buildDefinition.Links.Web.Href,
		}, 1)
	}
}

func findFirstCoverageStatsWithLineLabel(coverageObj devopsClient.BuildCodeCoverage, labelType int) *struct {
	Label            string  `json:"label"`
	Position         int     `json:"position"`
	Total            int     `json:"total"`
	Covered          int     `json:"covered"`
	IsDeltaAvailable bool    `json:"isDeltaAvailable"`
	Delta            float64 `json:"delta"`
} {
	for _, coverageData := range coverageObj.CoverageData {
		for _, stat := range coverageData.CoverageStats {
			if stat.Label == "Lines" && labelType == int(devopsClient.Lines) {
				return &stat
			}
			if stat.Label == "Branches" && labelType == int(devopsClient.Branches) {
				return &stat
			}
		}
	}
	return nil
}

func (m *MetricsCollectorBuild) collectBuilds(ctx context.Context, logger *zap.SugaredLogger, callback chan<- func(), project devopsClient.Project) {
	minTime := time.Now().Add(-opts.Limit.BuildHistoryDuration)

	list, err := AzureDevopsClient.ListBuildHistory(project.Id, minTime)
	if err != nil {
		logger.Error(err)
		return
	}

	buildMetric := m.Collector.GetMetricList("build")
	buildStatusMetric := m.Collector.GetMetricList("buildStatus")

	for _, build := range list.List {
		if build.Reason == "pullRequest" && build.Definition.Name == "Adora3.0 unit test and code coverage Test Dev Pipeline" && build.Result == "succeeded" {
			if build.Id > cloudDevBuildId {
				cloudDevBuildId = build.Id
			}
		}
		//Comment back in when we add unit test/code coverage for adora
		// if build.Reason == "pullRequest" && build.Definition.Name == "Adora-Develop" && build.Result == "succeeded" {
		// 	if build.Id > POSDevBuildId {
		// 		POSDevBuildId = build.Id
		// 	}
		// }

		buildMetric.AddInfo(prometheus.Labels{
			"projectID":         project.Id,
			"buildDefinitionID": int64ToString(build.Definition.Id),
			"buildID":           int64ToString(build.Id),
			"buildNumber":       build.BuildNumber,
			"buildName":         build.Definition.Name,
			"agentPoolID":       int64ToString(build.Queue.Pool.Id),
			"requestedBy":       build.RequestedBy.DisplayName,
			"sourceBranch":      build.SourceBranch,
			"sourceVersion":     build.SourceVersion,
			"status":            build.Status,
			"reason":            build.Reason,
			"result":            build.Result,
			"url":               build.Links.Web.Href,
		})

		buildStatusMetric.AddBool(prometheus.Labels{
			"projectID":         project.Id,
			"buildID":           int64ToString(build.Id),
			"buildDefinitionID": int64ToString(build.Definition.Id),
			"buildNumber":       build.BuildNumber,
			"result":            build.Result,
			"type":              "succeeded",
		}, build.Result == "succeeded")

		buildStatusMetric.AddTime(prometheus.Labels{
			"projectID":         project.Id,
			"buildID":           int64ToString(build.Id),
			"buildDefinitionID": int64ToString(build.Definition.Id),
			"buildNumber":       build.BuildNumber,
			"result":            build.Result,
			"type":              "queued",
		}, build.QueueTime)

		buildStatusMetric.AddTime(prometheus.Labels{
			"projectID":         project.Id,
			"buildID":           int64ToString(build.Id),
			"buildDefinitionID": int64ToString(build.Definition.Id),
			"buildNumber":       build.BuildNumber,
			"result":            build.Result,
			"type":              "started",
		}, build.StartTime)

		buildStatusMetric.AddTime(prometheus.Labels{
			"projectID":         project.Id,
			"buildID":           int64ToString(build.Id),
			"buildDefinitionID": int64ToString(build.Definition.Id),
			"buildNumber":       build.BuildNumber,
			"result":            build.Result,
			"type":              "finished",
		}, build.FinishTime)

		buildStatusMetric.AddDuration(prometheus.Labels{
			"projectID":         project.Id,
			"buildID":           int64ToString(build.Id),
			"buildDefinitionID": int64ToString(build.Definition.Id),
			"buildNumber":       build.BuildNumber,
			"result":            build.Result,
			"type":              "jobDuration",
		}, build.FinishTime.Sub(build.StartTime))
	}
}

func (m *MetricsCollectorBuild) collectBuildCodeCoverage(ctx context.Context, logger *zap.SugaredLogger, callback chan<- func(), project devopsClient.Project) {
	if cloudDevBuildId > 0 {
		coverage, err := AzureDevopsClient.GetCodeCoverageStatsOfBuild(project.Id, int64ToString(cloudDevBuildId))
		if err != nil {
			logger.Error(err)
			return
		}
		buildLineCodeCoverageMetric := m.Collector.GetMetricList("buildLineCodeCoverage")
		buildBranchCodeCoverageMetric := m.Collector.GetMetricList("buildBranchCodeCoverage")

		buildLineCodeCoverageMetric.AddIfGreaterZero(prometheus.Labels{
			"buildID":        coverage.Build.ID,
			"linesCoverable": strconv.Itoa(findFirstCoverageStatsWithLineLabel(coverage, int(devopsClient.Lines)).Total),
			"linesCovered":   strconv.Itoa(findFirstCoverageStatsWithLineLabel(coverage, int(devopsClient.Lines)).Covered),
			"pipelineName":   "Adora3.0 unit test and code coverage Test Dev Pipeline",
		}, float64(findFirstCoverageStatsWithLineLabel(coverage, int(devopsClient.Lines)).Covered)/float64(findFirstCoverageStatsWithLineLabel(coverage, int(devopsClient.Lines)).Total))

		buildBranchCodeCoverageMetric.AddIfGreaterZero(prometheus.Labels{
			"buildID":         coverage.Build.ID,
			"branchCoverable": strconv.Itoa(findFirstCoverageStatsWithLineLabel(coverage, int(devopsClient.Branches)).Total),
			"branchCovered":   strconv.Itoa(findFirstCoverageStatsWithLineLabel(coverage, int(devopsClient.Branches)).Covered),
			"pipelineName":    "Adora3.0 unit test and code coverage Test Dev Pipeline",
		}, float64(findFirstCoverageStatsWithLineLabel(coverage, int(devopsClient.Branches)).Covered)/float64(findFirstCoverageStatsWithLineLabel(coverage, int(devopsClient.Branches)).Total))
	}
	//Comment back in TODO
	// if POSDevBuildId > 0 {
	// 	coverage, err := AzureDevopsClient.GetCodeCoverageStatsOfBuild(project.Id, int64ToString(POSDevBuildId))
	// 	if err != nil {
	// 		logger.Error(err)
	// 		return
	// 	}
	// 	buildLineCodeCoverageMetric := m.Collector.GetMetricList("buildLineCodeCoverage")
	// 	buildBranchCodeCoverageMetric := m.Collector.GetMetricList("buildBranchCodeCoverage")

	// 	buildLineCodeCoverageMetric.AddIfGreaterZero(prometheus.Labels{
	// 		"buildID":        coverage.Build.ID,
	// 		"linesCoverable": strconv.Itoa(findFirstCoverageStatsWithLineLabel(coverage, int(devopsClient.Lines)).Total),
	// 		"linesCovered":   strconv.Itoa(findFirstCoverageStatsWithLineLabel(coverage, int(devopsClient.Lines)).Covered),
	//		"pipelineName":   "Adora-Develop",
	// 	}, float64(findFirstCoverageStatsWithLineLabel(coverage, 1).Covered)/float64(findFirstCoverageStatsWithLineLabel(coverage, int(devopsClient.Lines)).Total))

	// 	buildBranchCodeCoverageMetric.AddIfGreaterZero(prometheus.Labels{
	// 		"buildID":         coverage.Build.ID,
	// 		"branchCoverable": strconv.Itoa(findFirstCoverageStatsWithLineLabel(coverage, int(devopsClient.Branches)).Total),
	// 		"branchCovered":   strconv.Itoa(findFirstCoverageStatsWithLineLabel(coverage, int(devopsClient.Branches)).Covered),
	//		"pipelineName":   "Adora-Develop",
	// 	}, float64(findFirstCoverageStatsWithLineLabel(coverage, 1).Covered)/float64(findFirstCoverageStatsWithLineLabel(coverage, int(devopsClient.Branches)).Total))
	// }
}

func (m *MetricsCollectorBuild) collectBuildsTimeline(ctx context.Context, logger *zap.SugaredLogger, callback chan<- func(), project devopsClient.Project) {
	minTime := time.Now().Add(-opts.Limit.BuildHistoryDuration)
	list, err := AzureDevopsClient.ListBuildHistoryWithStatus(project.Id, minTime, "completed")
	if err != nil {
		logger.Error(err)
		return
	}

	buildStageMetric := m.Collector.GetMetricList("buildStage")
	buildPhaseMetric := m.Collector.GetMetricList("buildPhase")
	buildJobMetric := m.Collector.GetMetricList("buildJob")
	buildTaskMetric := m.Collector.GetMetricList("buildTask")

	for _, build := range list.List {
		timelineRecordList, _ := AzureDevopsClient.ListBuildTimeline(project.Id, int64ToString(build.Id))
		for _, timelineRecord := range timelineRecordList.List {
			recordType := timelineRecord.RecordType
			switch strings.ToLower(recordType) {
			case "stage":
				buildStageMetric.Add(prometheus.Labels{
					"projectID":         project.Id,
					"buildID":           int64ToString(build.Id),
					"buildDefinitionID": int64ToString(build.Definition.Id),
					"buildNumber":       build.BuildNumber,
					"name":              timelineRecord.Name,
					"id":                timelineRecord.Id,
					"identifier":        timelineRecord.Identifier,
					"result":            timelineRecord.Result,
					"type":              "errorCount",
				}, timelineRecord.ErrorCount)

				buildStageMetric.Add(prometheus.Labels{
					"projectID":         project.Id,
					"buildID":           int64ToString(build.Id),
					"buildDefinitionID": int64ToString(build.Definition.Id),
					"buildNumber":       build.BuildNumber,
					"name":              timelineRecord.Name,
					"id":                timelineRecord.Id,
					"identifier":        timelineRecord.Identifier,
					"result":            timelineRecord.Result,
					"type":              "warningCount",
				}, timelineRecord.WarningCount)

				buildStageMetric.AddBool(prometheus.Labels{
					"projectID":         project.Id,
					"buildID":           int64ToString(build.Id),
					"buildDefinitionID": int64ToString(build.Definition.Id),
					"buildNumber":       build.BuildNumber,
					"name":              timelineRecord.Name,
					"id":                timelineRecord.Id,
					"identifier":        timelineRecord.Identifier,
					"result":            timelineRecord.Result,
					"type":              "succeeded",
				}, timelineRecord.Result == "succeeded")

				buildStageMetric.AddTime(prometheus.Labels{
					"projectID":         project.Id,
					"buildID":           int64ToString(build.Id),
					"buildDefinitionID": int64ToString(build.Definition.Id),
					"buildNumber":       build.BuildNumber,
					"name":              timelineRecord.Name,
					"id":                timelineRecord.Id,
					"identifier":        timelineRecord.Identifier,
					"result":            timelineRecord.Result,
					"type":              "started",
				}, timelineRecord.StartTime)

				buildStageMetric.AddTime(prometheus.Labels{
					"projectID":         project.Id,
					"buildID":           int64ToString(build.Id),
					"buildDefinitionID": int64ToString(build.Definition.Id),
					"buildNumber":       build.BuildNumber,
					"name":              timelineRecord.Name,
					"id":                timelineRecord.Id,
					"identifier":        timelineRecord.Identifier,
					"result":            timelineRecord.Result,
					"type":              "finished",
				}, timelineRecord.FinishTime)

				buildStageMetric.AddDuration(prometheus.Labels{
					"projectID":         project.Id,
					"buildID":           int64ToString(build.Id),
					"buildDefinitionID": int64ToString(build.Definition.Id),
					"buildNumber":       build.BuildNumber,
					"name":              timelineRecord.Name,
					"id":                timelineRecord.Id,
					"identifier":        timelineRecord.Identifier,
					"result":            timelineRecord.Result,
					"type":              "duration",
				}, timelineRecord.FinishTime.Sub(timelineRecord.StartTime))

			case "phase":
				buildPhaseMetric.Add(prometheus.Labels{
					"projectID":         project.Id,
					"buildID":           int64ToString(build.Id),
					"buildDefinitionID": int64ToString(build.Definition.Id),
					"buildNumber":       build.BuildNumber,
					"name":              timelineRecord.Name,
					"id":                timelineRecord.Id,
					"parentId":          timelineRecord.ParentId,
					"identifier":        timelineRecord.Identifier,
					"result":            timelineRecord.Result,
					"type":              "errorCount",
				}, timelineRecord.ErrorCount)

				buildPhaseMetric.Add(prometheus.Labels{
					"projectID":         project.Id,
					"buildID":           int64ToString(build.Id),
					"buildDefinitionID": int64ToString(build.Definition.Id),
					"buildNumber":       build.BuildNumber,
					"name":              timelineRecord.Name,
					"id":                timelineRecord.Id,
					"parentId":          timelineRecord.ParentId,
					"identifier":        timelineRecord.Identifier,
					"result":            timelineRecord.Result,
					"type":              "warningCount",
				}, timelineRecord.WarningCount)

				buildPhaseMetric.AddBool(prometheus.Labels{
					"projectID":         project.Id,
					"buildID":           int64ToString(build.Id),
					"buildDefinitionID": int64ToString(build.Definition.Id),
					"buildNumber":       build.BuildNumber,
					"name":              timelineRecord.Name,
					"id":                timelineRecord.Id,
					"parentId":          timelineRecord.ParentId,
					"identifier":        timelineRecord.Identifier,
					"result":            timelineRecord.Result,
					"type":              "succeeded",
				}, timelineRecord.Result == "succeeded")

				buildPhaseMetric.AddTime(prometheus.Labels{
					"projectID":         project.Id,
					"buildID":           int64ToString(build.Id),
					"buildDefinitionID": int64ToString(build.Definition.Id),
					"buildNumber":       build.BuildNumber,
					"name":              timelineRecord.Name,
					"id":                timelineRecord.Id,
					"parentId":          timelineRecord.ParentId,
					"identifier":        timelineRecord.Identifier,
					"result":            timelineRecord.Result,
					"type":              "started",
				}, timelineRecord.StartTime)

				buildPhaseMetric.AddTime(prometheus.Labels{
					"projectID":         project.Id,
					"buildID":           int64ToString(build.Id),
					"buildDefinitionID": int64ToString(build.Definition.Id),
					"buildNumber":       build.BuildNumber,
					"name":              timelineRecord.Name,
					"id":                timelineRecord.Id,
					"parentId":          timelineRecord.ParentId,
					"identifier":        timelineRecord.Identifier,
					"result":            timelineRecord.Result,
					"type":              "finished",
				}, timelineRecord.FinishTime)

				buildPhaseMetric.AddDuration(prometheus.Labels{
					"projectID":         project.Id,
					"buildID":           int64ToString(build.Id),
					"buildDefinitionID": int64ToString(build.Definition.Id),
					"buildNumber":       build.BuildNumber,
					"name":              timelineRecord.Name,
					"id":                timelineRecord.Id,
					"parentId":          timelineRecord.ParentId,
					"identifier":        timelineRecord.Identifier,
					"result":            timelineRecord.Result,
					"type":              "duration",
				}, timelineRecord.FinishTime.Sub(timelineRecord.StartTime))

			case "job":
				buildJobMetric.Add(prometheus.Labels{
					"projectID":         project.Id,
					"buildID":           int64ToString(build.Id),
					"buildDefinitionID": int64ToString(build.Definition.Id),
					"buildNumber":       build.BuildNumber,
					"name":              timelineRecord.Name,
					"id":                timelineRecord.Id,
					"parentId":          timelineRecord.ParentId,
					"identifier":        timelineRecord.Identifier,
					"result":            timelineRecord.Result,
					"type":              "errorCount",
				}, timelineRecord.ErrorCount)

				buildJobMetric.Add(prometheus.Labels{
					"projectID":         project.Id,
					"buildID":           int64ToString(build.Id),
					"buildDefinitionID": int64ToString(build.Definition.Id),
					"buildNumber":       build.BuildNumber,
					"name":              timelineRecord.Name,
					"id":                timelineRecord.Id,
					"parentId":          timelineRecord.ParentId,
					"identifier":        timelineRecord.Identifier,
					"result":            timelineRecord.Result,
					"type":              "warningCount",
				}, timelineRecord.WarningCount)

				buildJobMetric.AddBool(prometheus.Labels{
					"projectID":         project.Id,
					"buildID":           int64ToString(build.Id),
					"buildDefinitionID": int64ToString(build.Definition.Id),
					"buildNumber":       build.BuildNumber,
					"name":              timelineRecord.Name,
					"id":                timelineRecord.Id,
					"parentId":          timelineRecord.ParentId,
					"identifier":        timelineRecord.Identifier,
					"result":            timelineRecord.Result,
					"type":              "succeeded",
				}, timelineRecord.Result == "succeeded")

				buildJobMetric.AddTime(prometheus.Labels{
					"projectID":         project.Id,
					"buildID":           int64ToString(build.Id),
					"buildDefinitionID": int64ToString(build.Definition.Id),
					"buildNumber":       build.BuildNumber,
					"name":              timelineRecord.Name,
					"id":                timelineRecord.Id,
					"parentId":          timelineRecord.ParentId,
					"identifier":        timelineRecord.Identifier,
					"result":            timelineRecord.Result,
					"type":              "started",
				}, timelineRecord.StartTime)

				buildJobMetric.AddTime(prometheus.Labels{
					"projectID":         project.Id,
					"buildID":           int64ToString(build.Id),
					"buildDefinitionID": int64ToString(build.Definition.Id),
					"buildNumber":       build.BuildNumber,
					"name":              timelineRecord.Name,
					"id":                timelineRecord.Id,
					"parentId":          timelineRecord.ParentId,
					"identifier":        timelineRecord.Identifier,
					"result":            timelineRecord.Result,
					"type":              "finished",
				}, timelineRecord.FinishTime)

				buildJobMetric.AddDuration(prometheus.Labels{
					"projectID":         project.Id,
					"buildID":           int64ToString(build.Id),
					"buildDefinitionID": int64ToString(build.Definition.Id),
					"buildNumber":       build.BuildNumber,
					"name":              timelineRecord.Name,
					"id":                timelineRecord.Id,
					"parentId":          timelineRecord.ParentId,
					"identifier":        timelineRecord.Identifier,
					"result":            timelineRecord.Result,
					"type":              "duration",
				}, timelineRecord.FinishTime.Sub(timelineRecord.StartTime))

			case "task":
				buildTaskMetric.Add(prometheus.Labels{
					"projectID":         project.Id,
					"buildID":           int64ToString(build.Id),
					"buildDefinitionID": int64ToString(build.Definition.Id),
					"buildNumber":       build.BuildNumber,
					"name":              timelineRecord.Name,
					"id":                timelineRecord.Id,
					"parentId":          timelineRecord.ParentId,
					"workerName":        timelineRecord.WorkerName,
					"result":            timelineRecord.Result,
					"type":              "errorCount",
				}, timelineRecord.ErrorCount)

				buildTaskMetric.Add(prometheus.Labels{
					"projectID":         project.Id,
					"buildID":           int64ToString(build.Id),
					"buildDefinitionID": int64ToString(build.Definition.Id),
					"buildNumber":       build.BuildNumber,
					"name":              timelineRecord.Name,
					"id":                timelineRecord.Id,
					"parentId":          timelineRecord.ParentId,
					"workerName":        timelineRecord.WorkerName,
					"result":            timelineRecord.Result,
					"type":              "warningCount",
				}, timelineRecord.WarningCount)

				buildTaskMetric.AddBool(prometheus.Labels{
					"projectID":         project.Id,
					"buildID":           int64ToString(build.Id),
					"buildDefinitionID": int64ToString(build.Definition.Id),
					"buildNumber":       build.BuildNumber,
					"name":              timelineRecord.Name,
					"id":                timelineRecord.Id,
					"parentId":          timelineRecord.ParentId,
					"workerName":        timelineRecord.WorkerName,
					"result":            timelineRecord.Result,
					"type":              "succeeded",
				}, timelineRecord.Result == "succeeded")

				buildTaskMetric.AddTime(prometheus.Labels{
					"projectID":         project.Id,
					"buildID":           int64ToString(build.Id),
					"buildDefinitionID": int64ToString(build.Definition.Id),
					"buildNumber":       build.BuildNumber,
					"name":              timelineRecord.Name,
					"id":                timelineRecord.Id,
					"parentId":          timelineRecord.ParentId,
					"workerName":        timelineRecord.WorkerName,
					"result":            timelineRecord.Result,
					"type":              "started",
				}, timelineRecord.StartTime)

				buildTaskMetric.AddTime(prometheus.Labels{
					"projectID":         project.Id,
					"buildID":           int64ToString(build.Id),
					"buildDefinitionID": int64ToString(build.Definition.Id),
					"buildNumber":       build.BuildNumber,
					"name":              timelineRecord.Name,
					"id":                timelineRecord.Id,
					"parentId":          timelineRecord.ParentId,
					"workerName":        timelineRecord.WorkerName,
					"result":            timelineRecord.Result,
					"type":              "finished",
				}, timelineRecord.FinishTime)

				buildTaskMetric.AddDuration(prometheus.Labels{
					"projectID":         project.Id,
					"buildID":           int64ToString(build.Id),
					"buildDefinitionID": int64ToString(build.Definition.Id),
					"buildNumber":       build.BuildNumber,
					"name":              timelineRecord.Name,
					"id":                timelineRecord.Id,
					"parentId":          timelineRecord.ParentId,
					"workerName":        timelineRecord.WorkerName,
					"result":            timelineRecord.Result,
					"type":              "duration",
				}, timelineRecord.FinishTime.Sub(timelineRecord.StartTime))
			}
		}
	}
}
