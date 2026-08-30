package scheduler

import (
	"context"
	"sync"
	"time"

	"monika-go/internal/alert"
	"monika-go/internal/config"
	"monika-go/internal/logger"
	"monika-go/internal/prober"
)

// Scheduler manages the concurrent lifecycle of multiple probe execution loops.
type Scheduler struct {
	cfg          *config.Config
	log          logger.Logger
	tracker      alert.Tracker
	alertManager alert.AlertManager
	wg           sync.WaitGroup
	cancel       context.CancelFunc
}

// New constructs a new Scheduler configured with the given Config and Logger.
func New(cfg *config.Config, log logger.Logger) *Scheduler {
	return &Scheduler{
		cfg:          cfg,
		log:          log.With(logger.Component("scheduler")),
		tracker:      alert.NewTracker(),
		alertManager: alert.NewAlertManager(log),
	}
}

// Tracker returns the scheduler's State Tracker.
func (s *Scheduler) Tracker() alert.Tracker {
	return s.tracker
}

// AlertManager returns the scheduler's AlertManager.
func (s *Scheduler) AlertManager() alert.AlertManager {
	return s.alertManager
}

// Start spawns background goroutines running individual probe loops.
func (s *Scheduler) Start(ctx context.Context) error {
	runCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	s.log.Info("starting scheduler", logger.F("probe_count", len(s.cfg.Probes)))

	for _, p := range s.cfg.Probes {
		p := p
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.runProbeLoop(runCtx, p)
		}()
	}

	return nil
}

// Stop signals all active probe loops to stop and blocks until they gracefully exit.
func (s *Scheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
	s.log.Info("scheduler stopped")
}

// runProbeLoop runs an individual probe periodically based on its configured interval.
func (s *Scheduler) runProbeLoop(ctx context.Context, p config.Probe) {
	interval := p.Interval
	if interval <= 0 {
		interval = 10 // default 10 seconds per Monika specification
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	s.log.Info("starting probe loop", logger.F("probe_id", p.ID), logger.F("interval_seconds", interval))

	// Run the first probe immediately instead of waiting a full interval.
	s.executeProbe(ctx, p)

	for {
		select {
		case <-ctx.Done():
			s.log.Info("stopping probe loop", logger.F("probe_id", p.ID))
			return
		case <-ticker.C:
			s.executeProbe(ctx, p)
		}
	}
}

// executeProbe handles a single tick execution for a given probe.
func (s *Scheduler) executeProbe(ctx context.Context, p config.Probe) {
	pb := prober.NewProber(p.Spec)
	if pb == nil {
		s.log.Warn("skipping probe: unsupported or unimplemented prober type", logger.F("probe_id", p.ID), logger.F("kind", p.Spec.Kind()))
		return
	}

	s.log.Debug("executing probe", logger.F("probe_id", p.ID))
	start := time.Now()
	results, err := pb.Probe(ctx)
	duration := time.Since(start).Milliseconds()

	if err != nil {
		s.log.Error("probe failed", logger.F("probe_id", p.ID), logger.Err(err), logger.F("duration_ms", duration))

		// Record probe execution failure to evaluate incidents
		event, errTrack := s.tracker.RecordResult(p.ID, false, p.IncidentThreshold, p.RecoveryThreshold)
		if errTrack != nil {
			s.log.Error("failed to record state result", logger.F("probe_id", p.ID), logger.Err(errTrack))
		} else if event != nil {
			s.alertManager.PublishTransition(*event)
		}
		return
	}

	s.log.Info("probe completed", logger.F("probe_id", p.ID), logger.F("duration_ms", duration), logger.F("results_count", len(results)))

	runSuccess := true
	for i, res := range results {
		if !res.AlertPassed {
			runSuccess = false
			for _, fa := range res.FailedAlerts {
				s.log.Warn("probe alert failed",
					logger.F("probe_id", p.ID),
					logger.F("request_index", i),
					logger.F("assertion", fa.Assertion),
					logger.F("message", fa.Message),
				)
			}
		}

		// Probe-level alerts apply to every request result.
		for _, pa := range p.Alerts {
			if pa.Assertion == nil {
				continue
			}
			if !pa.Assertion.Evaluate(res.Result) {
				runSuccess = false
				s.log.Warn("probe alert failed",
					logger.F("probe_id", p.ID),
					logger.F("level", "probe"),
					logger.F("request_index", i),
					logger.F("assertion", pa.Assertion.String()),
					logger.F("message", pa.Message),
				)
			}
		}
	}

	// Record execution outcome to evaluate incident/recovery transitions
	event, errTrack := s.tracker.RecordResult(p.ID, runSuccess, p.IncidentThreshold, p.RecoveryThreshold)
	if errTrack != nil {
		s.log.Error("failed to record state result", logger.F("probe_id", p.ID), logger.Err(errTrack))
	} else if event != nil {
		s.alertManager.PublishTransition(*event)
	}
}
