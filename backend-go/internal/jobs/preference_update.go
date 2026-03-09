package jobs

import (
	"log"
	"sync"
	"time"

	"my-robot-backend/internal/domain/preferences"
	"my-robot-backend/internal/platform/database"
)

type PreferenceUpdateScheduler struct {
	checkInterval int
	stopChan      chan bool
	wg            sync.WaitGroup
	mu            sync.Mutex
	running       bool
}

func NewPreferenceUpdateScheduler(checkInterval int) *PreferenceUpdateScheduler {
	return &PreferenceUpdateScheduler{
		checkInterval: checkInterval,
		stopChan:      make(chan bool),
		running:       false,
	}
}

func (s *PreferenceUpdateScheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	s.running = true
	s.wg.Add(1)

	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(time.Duration(s.checkInterval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.runUpdate()
			case <-s.stopChan:
				log.Println("Preference update scheduler stopped")
				return
			}
		}
	}()

	log.Printf("Preference update scheduler started (interval: %d seconds)", s.checkInterval)
	return nil
}

func (s *PreferenceUpdateScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	close(s.stopChan)
	s.wg.Wait()
	s.stopChan = make(chan bool)
}

func (s *PreferenceUpdateScheduler) runUpdate() {
	log.Println("Running preference update...")

	preferenceService := preferences.NewPreferenceService(database.DB)
	if err := preferenceService.UpdateAllPreferences(); err != nil {
		log.Printf("Preference update failed: %v", err)
	} else {
		log.Println("Preference update completed successfully")
	}
}

func (s *PreferenceUpdateScheduler) TriggerManualUpdate() {
	log.Println("Manual preference update triggered")
	s.runUpdate()
}
