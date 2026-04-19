package run

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/ben/argus/internal/analyze"
	"github.com/ben/argus/internal/collect"
	"github.com/ben/argus/internal/config"
	"github.com/ben/argus/internal/model"
	"github.com/ben/argus/internal/output"
	"github.com/ben/argus/internal/ping"
	"github.com/ben/argus/internal/targets"
)

const version = "1.0.0"

var ErrPartialFailure = errors.New("argus completed with partial failures")

type runtimeDeps struct {
	gatewayResolver targets.GatewayResolver
	deviceCollector collect.DeviceCollector
	eventCollector  collect.EventCollector
	prober          ping.Prober
	now             func() time.Time
}

func Execute(ctx context.Context, cfg config.Config) error {
	deps := runtimeDeps{
		gatewayResolver: collect.NewGatewayResolver(),
		deviceCollector: collect.NewDeviceCollector(),
		eventCollector:  collect.NewEventCollector(),
		prober:          collect.NewProber(),
		now:             time.Now,
	}
	return execute(ctx, cfg, deps)
}

func execute(ctx context.Context, cfg config.Config, deps runtimeDeps) error {
	if err := cfg.ValidatePlatform(runtime.GOOS); err != nil {
		return err
	}

	start := deps.now()
	runID := cfg.RunID
	if runID == "" {
		hostname, _ := collect.Hostname()
		runID = fmt.Sprintf("%s-%s", start.Format("20060102-150405"), hostname)
	}

	writer, err := output.New(cfg.OutDir, start)
	if err != nil {
		return err
	}

	deviceInfo, deviceWarnings := deps.deviceCollector.Collect(ctx)
	allWarnings := append([]string(nil), deviceWarnings...)

	targetList, targetWarnings := targets.Build(cfg, deps.gatewayResolver)
	allWarnings = append(allWarnings, targetWarnings...)
	if len(targetList) == 0 {
		return fmt.Errorf("no ping targets available")
	}

	resultsByTarget := make(map[string][]model.ProbeResult, len(targetList))
	pingLogNames := make([]string, 0, len(targetList))
	resultsCh := make(chan model.ProbeResult, len(targetList))
	errorsCh := make(chan error, len(targetList)*2)

	runCtx, cancel := context.WithTimeout(ctx, cfg.Duration)
	defer cancel()

	var wg sync.WaitGroup
	for _, target := range targetList {
		logger, err := ping.NewCSVLogger(writer.RunDir(), target)
		if err != nil {
			return err
		}
		pingLogNames = append(pingLogNames, target.FileName)
		worker := ping.Worker{
			Target:  target,
			RunID:   runID,
			Timeout: target.Interval,
			Prober:  deps.prober,
			Logger:  logger,
			Now:     deps.now,
		}
		wg.Add(1)
		go func(w ping.Worker) {
			defer wg.Done()
			w.Run(runCtx, resultsCh, errorsCh)
		}(worker)
	}

	go func() {
		wg.Wait()
		close(resultsCh)
		close(errorsCh)
	}()

	for result := range resultsCh {
		resultsByTarget[result.TargetLabel] = append(resultsByTarget[result.TargetLabel], result)
	}

	var partialErrors []string
	for err := range errorsCh {
		partialErrors = append(partialErrors, err.Error())
	}
	sort.Strings(partialErrors)

	end := deps.now()

	var events []model.EventRecord
	if cfg.IncludeEvents {
		records, eventWarnings := deps.eventCollector.Collect(ctx, start.Add(-5*time.Minute), end)
		events = records
		allWarnings = append(allWarnings, eventWarnings...)
	} else {
		allWarnings = append(allWarnings, "event collection disabled by configuration")
	}

	summary := analyze.BuildSummary(targetList, resultsByTarget, allWarnings)
	if err := writer.WriteJSON("device_info.json", deviceInfo); err != nil {
		return err
	}
	if err := writer.WriteJSON("events.json", events); err != nil {
		return err
	}
	if err := writer.WriteJSON("summary.json", summary); err != nil {
		return err
	}

	runMeta := model.RunMetadata{
		RunID:           runID,
		Tool:            model.ToolName,
		Version:         version,
		StartTime:       start.Format(time.RFC3339),
		EndTime:         end.Format(time.RFC3339),
		DurationMinutes: cfg.DurationMinutes(),
		Files:           output.ManifestFor(pingLogNames),
		Warnings:        allWarnings,
		Errors:          partialErrors,
	}
	if err := writer.WriteJSON("run.json", runMeta); err != nil {
		return err
	}

	if len(partialErrors) > 0 {
		return ErrPartialFailure
	}
	return nil
}
