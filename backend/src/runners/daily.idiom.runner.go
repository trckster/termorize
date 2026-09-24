package runners

import (
	"context"
	"termorize/src/integrations/telegram"
	"termorize/src/logger"
	"termorize/src/services"
	"time"
)

func StartDailyIdiomRunner(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			if err := services.DeliverDailyIdioms(ctx, time.Now(), telegram.SendDailyIdiom); err != nil {
				logger.L().Errorw("daily idiom delivery failed", "error", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}
