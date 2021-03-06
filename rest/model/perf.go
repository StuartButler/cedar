package model

import (
	dbmodel "github.com/evergreen-ci/cedar/model"
	"github.com/mongodb/ftdc/events"
	"github.com/pkg/errors"
)

type APIPerformanceResult struct {
	Name        APIString                `json:"name"`
	Info        APIPerformanceResultInfo `json:"info"`
	CreatedAt   APITime                  `json:"create_at"`
	CompletedAt APITime                  `json:"completed_at"`
	Version     int                      `json:"version"`
	Artifacts   []APIArtifactInfo        `json:"artifacts"`
	Total       *APIPerformanceEvent     `json:"total"`
	Rollups     *APIPerfRollups          `json:"rollups"`
}

func (apiResult *APIPerformanceResult) Import(i interface{}) error {
	switch r := i.(type) {
	case dbmodel.PerformanceResult:
		apiResult.Name = ToAPIString(r.ID)
		apiResult.CreatedAt = NewTime(r.CreatedAt)
		apiResult.CompletedAt = NewTime(r.CompletedAt)
		apiResult.Version = r.Version
		apiResult.Info = getPerformanceResultInfo(r.Info)

		if r.Total != nil {
			total := getPerformanceEvent(r.Total)
			apiResult.Total = &total
		}
		if r.Rollups != nil {
			rollups := getPerfRollups(r.Rollups)
			apiResult.Rollups = &rollups
		}

		var apiArtifacts []APIArtifactInfo
		for _, artifactInfo := range r.Artifacts {
			apiArtifacts = append(apiArtifacts, getArtifactInfo(artifactInfo))
		}
		apiResult.Artifacts = apiArtifacts
	default:
		return errors.New("incorrect type when fetching converting PerformanceResult type")
	}
	return nil
}

func (apiResult *APIPerformanceResult) Export(i interface{}) (interface{}, error) {
	return nil, errors.Errorf("Export is not implemented for APIPerformanceResult")
}

type APIPerformanceResultInfo struct {
	Project   APIString        `json:"project"`
	Version   APIString        `json:"version"`
	TaskName  APIString        `json:"task_name"`
	TaskID    APIString        `json:"task_id"`
	Execution int              `json:"execution"`
	TestName  APIString        `json:"test_name"`
	Trial     int              `json:"trial"`
	Parent    APIString        `json:"parent"`
	Tags      []string         `json:"tags"`
	Arguments map[string]int32 `json:"args"`
	Schema    int              `json:"schema"`
}

func getPerformanceResultInfo(r dbmodel.PerformanceResultInfo) APIPerformanceResultInfo {
	return APIPerformanceResultInfo{
		Project:   ToAPIString(r.Project),
		Version:   ToAPIString(r.Version),
		TaskName:  ToAPIString(r.TaskName),
		TaskID:    ToAPIString(r.TaskID),
		Execution: r.Execution,
		TestName:  ToAPIString(r.TestName),
		Trial:     r.Trial,
		Parent:    ToAPIString(r.Parent),
		Tags:      r.Tags,
		Arguments: r.Arguments,
		Schema:    r.Schema,
	}
}

type APIArtifactInfo struct {
	Type        APIString `json:"type"`
	Bucket      APIString `json:"bucket"`
	Path        APIString `json:"path"`
	Format      APIString `json:"format"`
	Compression APIString `json:"compression"`
	Schema      APIString `bson:"schema"`
	Tags        []string  `json:"tags"`
	CreatedAt   APITime   `bson:"created_at"`
}

func getArtifactInfo(r dbmodel.ArtifactInfo) APIArtifactInfo {
	return APIArtifactInfo{
		Type:        ToAPIString(string(r.Type)),
		Bucket:      ToAPIString(r.Bucket),
		Path:        ToAPIString(r.Path),
		Format:      ToAPIString(string(r.Format)),
		Compression: ToAPIString(string(r.Compression)),
		Schema:      ToAPIString(string(r.Schema)),
		Tags:        r.Tags,
		CreatedAt:   NewTime(r.CreatedAt),
	}
}

type APIPerformanceEvent struct {
	Timestamp APITime `json:"ts"`
	Counters  APIPerformanceCounters
	Timers    APIPerformanceTimers
	Gauges    APIPerformanceGauges
}

type APIPerformanceCounters struct {
	Number     int64 `json:"n"`
	Operations int64 `json:"ops"`
	Size       int64 `json:"size"`
	Errors     int64 `json:"errors"`
}

type APIPerformanceTimers struct {
	Duration APIDuration `json:"dur"`
	Total    APIDuration `json:"total"`
}

type APIPerformanceGauges struct {
	State   int64 `json:"state"`
	Workers int64 `json:"workers"`
	Failed  bool  `json:"failed"`
}

func getPerformanceEvent(r *events.Performance) APIPerformanceEvent {
	return APIPerformanceEvent{
		Timestamp: NewTime(r.Timestamp),
		Counters: APIPerformanceCounters{
			Number:     r.Counters.Number,
			Operations: r.Counters.Operations,
			Size:       r.Counters.Size,
			Errors:     r.Counters.Errors,
		},
		Timers: APIPerformanceTimers{
			Duration: NewAPIDuration(r.Timers.Duration),
			Total:    NewAPIDuration(r.Timers.Total),
		},
		Gauges: APIPerformanceGauges{
			State:   r.Gauges.State,
			Workers: r.Gauges.Workers,
			Failed:  r.Gauges.Failed,
		},
	}
}

type APIPerfRollups struct {
	Stats       []APIPerfRollupValue `json:"stats"`
	ProcessedAt APITime              `json:"processed_at"`
	Count       int                  `json:"count"`
	Valid       bool                 `json:"valid"`
}

type APIPerfRollupValue struct {
	Name          APIString   `json:"name"`
	Value         interface{} `json:"val"`
	Version       int         `json:"version"`
	UserSubmitted bool        `json:"user"`
}

func getPerfRollups(r *dbmodel.PerfRollups) APIPerfRollups {
	rollups := APIPerfRollups{
		ProcessedAt: NewTime(r.ProcessedAt),
		Count:       r.Count,
		Valid:       r.Valid,
	}

	var apiStats []APIPerfRollupValue
	for _, stat := range r.Stats {
		apiStats = append(apiStats, getPerfRollupValue(stat))
	}
	rollups.Stats = apiStats

	return rollups
}

func getPerfRollupValue(r dbmodel.PerfRollupValue) APIPerfRollupValue {
	return APIPerfRollupValue{
		Name:          ToAPIString(r.Name),
		Value:         r.Value,
		Version:       r.Version,
		UserSubmitted: r.UserSubmitted,
	}
}
