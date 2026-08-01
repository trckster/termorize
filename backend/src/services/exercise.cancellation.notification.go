package services

import "sync"

var cancelledTelegramExercisesNotifier struct {
	sync.RWMutex
	fn func([]CancelledTelegramExercise)
}

func RegisterCancelledTelegramExercisesNotifier(fn func([]CancelledTelegramExercise)) {
	cancelledTelegramExercisesNotifier.Lock()
	defer cancelledTelegramExercisesNotifier.Unlock()
	cancelledTelegramExercisesNotifier.fn = fn
}

func notifyCancelledTelegramExercises(exercises []CancelledTelegramExercise) {
	if len(exercises) == 0 {
		return
	}
	cancelledTelegramExercisesNotifier.RLock()
	fn := cancelledTelegramExercisesNotifier.fn
	cancelledTelegramExercisesNotifier.RUnlock()
	if fn != nil {
		fn(exercises)
	}
}
