package jobs

import "testing"

func TestFirecrawlTriggerNowRejectsWhenAlreadyExecuting(t *testing.T) {
	scheduler := NewFirecrawlScheduler()
	scheduler.executionMutex.Lock()
	defer scheduler.executionMutex.Unlock()

	result := scheduler.TriggerNow()
	if result["accepted"] != false {
		t.Fatalf("accepted = %v, want false", result["accepted"])
	}
	if result["reason"] != "already_running" {
		t.Fatalf("reason = %v, want already_running", result["reason"])
	}
}
