package analyze

import (
	"math"
	"sort"

	"github.com/ben/argus/internal/model"
)

const (
	minSamplesForClassification = 5
	negligibleLossPercent       = 1.0
	materialLossPercent         = 5.0
	outageWindowThreshold       = 2
)

func BuildSummary(targets []model.Target, byTarget map[string][]model.ProbeResult, warnings []string) model.Summary {
	summary := model.Summary{Warnings: append([]string(nil), warnings...)}

	for _, target := range targets {
		results := byTarget[target.Label]
		targetSummary := summarizeTarget(target, results)
		summary.Targets = append(summary.Targets, targetSummary)
	}

	sort.Slice(summary.Targets, func(i, j int) bool {
		return summary.Targets[i].TargetLabel < summary.Targets[j].TargetLabel
	})
	summary.Classification = classify(summary.Targets)
	return summary
}

func summarizeTarget(target model.Target, results []model.ProbeResult) model.TargetSummary {
	out := model.TargetSummary{
		TargetLabel: target.Label,
		TargetHost:  target.Host,
		TargetIP:    target.IP,
		TargetKind:  kindForTarget(target),
	}

	var rtts []int64
	currentFailure := 0
	var outageStart string
	var failureSequenceStart string

	for _, result := range results {
		out.PacketsSent++
		if result.Result == "success" {
			if outageStart != "" {
				out.OutageWindows = append(out.OutageWindows, model.OutageWindow{
					StartTime:    outageStart,
					EndTime:      result.Timestamp.Format(timeLayout),
					MissedProbes: currentFailure,
				})
				outageStart = ""
			}
			out.PacketsReceived++
			if result.RTTMs != nil {
				rtts = append(rtts, *result.RTTMs)
			}
			currentFailure = 0
			failureSequenceStart = ""
			continue
		}

		out.PacketsLost++
		currentFailure++
		if currentFailure == 1 {
			failureSequenceStart = result.Timestamp.Format(timeLayout)
		}
		if currentFailure > out.LongestFailureStreak {
			out.LongestFailureStreak = currentFailure
		}
		if currentFailure == outageWindowThreshold {
			outageStart = failureSequenceStart
		}
	}

	if outageStart != "" {
		out.OutageWindows = append(out.OutageWindows, model.OutageWindow{
			StartTime:    outageStart,
			EndTime:      "",
			MissedProbes: currentFailure,
		})
	}

	if out.PacketsSent > 0 {
		out.PacketLossPercent = math.Round((float64(out.PacketsLost)/float64(out.PacketsSent))*10000) / 100
	}
	if len(rtts) > 0 {
		sort.Slice(rtts, func(i, j int) bool { return rtts[i] < rtts[j] })
		minRTT := rtts[0]
		maxRTT := rtts[len(rtts)-1]
		var sum int64
		for _, rtt := range rtts {
			sum += rtt
		}
		avg := math.Round((float64(sum)/float64(len(rtts)))*100) / 100
		out.MinRTTMs = &minRTT
		out.MaxRTTMs = &maxRTT
		out.AvgRTTMs = &avg
	}
	if len(results) < minSamplesForClassification {
		out.Warnings = append(out.Warnings, "insufficient samples for confident classification")
	}

	return out
}

const timeLayout = "2006-01-02T15:04:05"

func kindForTarget(target model.Target) string {
	switch {
	case target.IsGateway:
		return "gateway"
	case target.IsInternet:
		return "internet"
	default:
		return "other"
	}
}

func classify(targets []model.TargetSummary) string {
	if len(targets) == 0 {
		return "insufficient_data"
	}

	var gateway *model.TargetSummary
	var internet []model.TargetSummary

	for i := range targets {
		target := &targets[i]
		if target.TargetKind == "gateway" {
			gateway = target
			continue
		}
		if target.TargetKind == "internet" {
			internet = append(internet, *target)
		}
	}

	primaryAvailable := 0
	if gateway != nil && gateway.PacketsSent >= minSamplesForClassification {
		primaryAvailable++
	}
	stableInternetCount := 0
	unstableInternetCount := 0
	var worstInternet *model.TargetSummary

	for i := range internet {
		target := &internet[i]
		if target.PacketsSent >= minSamplesForClassification {
			primaryAvailable++
		}
		if isStable(*target) {
			stableInternetCount++
		}
		if isUnstable(*target) {
			unstableInternetCount++
		}
		if worstInternet == nil || target.PacketLossPercent > worstInternet.PacketLossPercent {
			worstInternet = target
		}
	}

	if primaryAvailable == 0 {
		return "insufficient_data"
	}
	if gateway != nil && isUnstable(*gateway) {
		return "local_instability"
	}
	if gateway != nil && isStable(*gateway) && unstableInternetCount >= 2 {
		return "upstream_instability"
	}
	if gateway != nil && isStable(*gateway) && stableInternetCount >= 1 && worstInternet != nil && isUnstable(*worstInternet) && unstableInternetCount == 1 {
		return "destination_specific_issue"
	}
	if gateway == nil && unstableInternetCount >= 2 {
		return "upstream_instability"
	}
	if (gateway == nil || isStable(*gateway)) && unstableInternetCount == 0 {
		return "no_issue_observed"
	}
	return "insufficient_data"
}

func isStable(target model.TargetSummary) bool {
	return target.PacketsSent >= minSamplesForClassification &&
		target.PacketLossPercent <= negligibleLossPercent &&
		len(target.OutageWindows) == 0
}

func isUnstable(target model.TargetSummary) bool {
	return target.PacketsSent >= minSamplesForClassification &&
		(target.PacketLossPercent >= materialLossPercent || len(target.OutageWindows) > 0)
}
