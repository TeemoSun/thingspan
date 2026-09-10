package services

import (
	"context"
	"log"
	"time"

	"thingspan/internal/config"
)

type Scheduler struct {
	scanner *ReminderScanner
	cfg     *config.Config
	cancel  context.CancelFunc
}

func NewScheduler(scanner *ReminderScanner, cfg *config.Config) *Scheduler {
	return &Scheduler{
		scanner: scanner,
		cfg:     cfg,
	}
}

func (s *Scheduler) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	log.Printf("调度器已启动，每日 %d:00 检查提醒", s.cfg.ReminderCheckHour)

	// Run once immediately on startup to catch up on missed reminders and overdue items
	go func() {
		log.Println("调度器：启动即时补跑提醒扫描...")
		if err := s.scanner.RunScan(); err != nil {
			log.Printf("调度器初始扫描失败: %v", err)
		}
	}()

	// Loop for daily scheduled check
	go func() {
		for {
			now := s.cfg.LocalNow()
			next := time.Date(now.Year(), now.Month(), now.Day(), s.cfg.ReminderCheckHour, 0, 0, 0, s.cfg.Location)
			if !next.After(now) {
				next = next.AddDate(0, 0, 1)
			}
			waitDuration := next.Sub(now)

			select {
			case <-ctx.Done():
				return
			case <-time.After(waitDuration):
				log.Println("调度器：触发每日定时提醒扫描...")
				if err := s.scanner.RunScan(); err != nil {
					log.Printf("调度器定时扫描失败: %v", err)
				}
			}
		}
	}()
}

func (s *Scheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

