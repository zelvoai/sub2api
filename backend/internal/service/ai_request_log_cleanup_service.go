package service

import (
	"context"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

type AIRequestLogCleanupService struct {
	logService *AIRequestLogService
	stopCh     chan struct{}
	startOnce  sync.Once
	stopOnce   sync.Once
	wg         sync.WaitGroup
}

func NewAIRequestLogCleanupService(logService *AIRequestLogService) *AIRequestLogCleanupService {
	return &AIRequestLogCleanupService{
		logService: logService,
		stopCh:     make(chan struct{}),
	}
}

func (s *AIRequestLogCleanupService) Start() {
	if s == nil || s.logService == nil {
		return
	}
	s.startOnce.Do(func() {
		s.wg.Add(1)
		go s.loop()
	})
}

func (s *AIRequestLogCleanupService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
		s.wg.Wait()
	})
}

func (s *AIRequestLogCleanupService) loop() {
	defer s.wg.Done()
	for {
		settings, err := s.logService.GetRetentionSettings(context.Background())
		if err != nil {
			logger.LegacyPrintf("service.ai_request_log_cleanup", "[AIRequestLogCleanup] load settings failed: %v", err)
			settings = defaultAIRequestLogRetentionSettings()
		}
		wait := time.Duration(settings.CleanupIntervalMinute) * time.Minute
		if wait <= 0 {
			wait = 30 * time.Minute
		}
		timer := time.NewTimer(wait)
		select {
		case <-s.stopCh:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		case <-timer.C:
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		deleted, err := s.logService.Cleanup(ctx, settings.DeleteBatchSize)
		cancel()
		if err != nil {
			logger.LegacyPrintf("service.ai_request_log_cleanup", "[AIRequestLogCleanup] cleanup failed: %v", err)
			continue
		}
		if deleted > 0 {
			logger.LegacyPrintf("service.ai_request_log_cleanup", "[AIRequestLogCleanup] deleted=%d", deleted)
		}
	}
}
