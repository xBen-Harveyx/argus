package analyze

import (
	"testing"
	"time"

	"github.com/ben/argus/internal/model"
)

func TestBuildSummaryDetectsLocalInstability(t *testing.T) {
	targets := []model.Target{
		{Label: "local_gateway", Host: "192.168.1.1", IP: "192.168.1.1", IsGateway: true},
		{Label: "internet_1", Host: "1.1.1.1", IP: "1.1.1.1", IsInternet: true},
		{Label: "internet_2", Host: "8.8.8.8", IP: "8.8.8.8", IsInternet: true},
	}
	base := time.Date(2026, 4, 19, 14, 30, 0, 0, time.UTC)
	byTarget := map[string][]model.ProbeResult{
		"local_gateway": {
			failure(base), failure(base.Add(time.Second)), success(base.Add(2 * time.Second), 2), failure(base.Add(3 * time.Second)), failure(base.Add(4 * time.Second)),
		},
		"internet_1": {success(base, 10), success(base.Add(time.Second), 11), success(base.Add(2 * time.Second), 12), success(base.Add(3 * time.Second), 13), success(base.Add(4 * time.Second), 14)},
		"internet_2": {success(base, 10), success(base.Add(time.Second), 11), success(base.Add(2 * time.Second), 12), success(base.Add(3 * time.Second), 13), success(base.Add(4 * time.Second), 14)},
	}

	summary := BuildSummary(targets, byTarget, nil)
	if summary.Classification != "local_instability" {
		t.Fatalf("Classification = %q, want local_instability", summary.Classification)
	}
}

func TestBuildSummaryDetectsDestinationSpecificIssue(t *testing.T) {
	targets := []model.Target{
		{Label: "local_gateway", Host: "192.168.1.1", IP: "192.168.1.1", IsGateway: true},
		{Label: "internet_1", Host: "1.1.1.1", IP: "1.1.1.1", IsInternet: true},
		{Label: "internet_2", Host: "8.8.8.8", IP: "8.8.8.8", IsInternet: true},
	}
	base := time.Date(2026, 4, 19, 14, 30, 0, 0, time.UTC)
	byTarget := map[string][]model.ProbeResult{
		"local_gateway": {success(base, 1), success(base.Add(time.Second), 1), success(base.Add(2 * time.Second), 1), success(base.Add(3 * time.Second), 1), success(base.Add(4 * time.Second), 1)},
		"internet_1":    {success(base, 1), success(base.Add(time.Second), 1), success(base.Add(2 * time.Second), 1), success(base.Add(3 * time.Second), 1), success(base.Add(4 * time.Second), 1)},
		"internet_2":    {failure(base), failure(base.Add(time.Second)), success(base.Add(2 * time.Second), 1), failure(base.Add(3 * time.Second)), failure(base.Add(4 * time.Second))},
	}

	summary := BuildSummary(targets, byTarget, nil)
	if summary.Classification != "destination_specific_issue" {
		t.Fatalf("Classification = %q, want destination_specific_issue", summary.Classification)
	}
}

func success(ts time.Time, rtt int64) model.ProbeResult {
	return model.ProbeResult{Timestamp: ts, Result: "success", RTTMs: &rtt}
}

func failure(ts time.Time) model.ProbeResult {
	return model.ProbeResult{Timestamp: ts, Result: "timeout"}
}
