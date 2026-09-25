package services

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"termorize/src/data/db"
	"termorize/src/models"
	"time"
)

// DeliverDailyIdioms is polled during the 11:00 local minute. A per-user database
// lock and a persistent local-date receipt prevent parallel workers and restarts
// from sending the same day's idiom again. Failed sends may retry in that minute.
func DeliverDailyIdioms(ctx context.Context, now time.Time, send func(models.User, DailyIdiomResponse) error) error {
	var ids []uint
	if err := db.DB.WithContext(ctx).Model(&models.User{}).
		Where("telegram_id > 0 AND settings->'telegram'->>'bot_enabled' = 'true' AND settings->'telegram'->>'daily_idiom_enabled' = 'true'").Pluck("id", &ids).Error; err != nil {
		return err
	}
	var failures []error
	for _, id := range ids {
		err := db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var locked bool
			if err := tx.Raw("SELECT pg_try_advisory_xact_lock(hashtext(?))", fmt.Sprintf("daily-idiom-delivery:%d", id)).Scan(&locked).Error; err != nil {
				return err
			}
			if !locked {
				return nil
			}
			var user models.User
			if err := tx.First(&user, id).Error; err != nil {
				return err
			}
			if !user.Settings.Telegram.BotEnabled || !user.Settings.Telegram.DailyIdiomEnabled || user.TelegramID <= 0 {
				return nil
			}
			location, err := time.LoadLocation(user.Settings.TimeZone)
			if err != nil || user.Settings.TimeZone == "" {
				location = time.UTC
			}
			local := now.In(location)
			if local.Hour() != 11 || local.Minute() != 0 {
				return nil
			}
			date := local.Format(time.DateOnly)
			var count int64
			if err := tx.Table("daily_idiom_deliveries").Where("user_id = ? AND date = ?", id, date).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return nil
			}
			daily, err := GetDailyIdiom(ctx, id, now)
			if err != nil {
				return err
			}
			if daily.Idiom == nil {
				return nil
			}
			if err := send(user, *daily); err != nil {
				return err
			}
			return tx.Exec("INSERT INTO daily_idiom_deliveries (user_id, date, daily_idiom_id, sent_at) VALUES (?, ?, ?, ?)", id, date, daily.Idiom.ID, now).Error
		})
		if err != nil {
			failures = append(failures, fmt.Errorf("user %d: %w", id, err))
		}
	}
	return errors.Join(failures...)
}
