package main

import (
	"flag"
	"fmt"
	chatdata "lit-night-bot/chat-data"
	chatio "lit-night-bot/io"
	"strconv"
	"time"
)

type migrationCandidate struct {
	chatID   int64
	result   chatdata.MigrationResult
	migrated *chatdata.ChatData
}

func runMigrationCommand(args []string) int {
	flags := flag.NewFlagSet("migrate", flag.ContinueOnError)
	apply := flags.Bool("apply", false, "write migrated data; without this flag only a dry-run is performed")
	chatID := flags.Int64("chat-id", 0, "migrate only one chat ID")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	logger := getLogger(true).WithField("entry", "migration")
	storage := chatio.NewIOChatData(logger, GetDataPath())
	if err := migrateStoredChats(storage, *apply, *chatID, time.Now()); err != nil {
		fmt.Printf("❌ Миграция остановлена: %v\n", err)
		return 1
	}
	return 0
}

func migrateStoredChats(storage *chatio.IoChatData, apply bool, onlyChatID int64, now time.Time) error {
	files, err := storage.GetDatasList()
	if err != nil {
		return err
	}
	candidates := make([]migrationCandidate, 0)
	for _, file := range files {
		chatID, err := strconv.ParseInt(file, 10, 64)
		if err != nil || (onlyChatID != 0 && chatID != onlyChatID) {
			continue
		}
		data := storage.GetChatData(chatID)
		if data == nil {
			return fmt.Errorf("чат %d: JSON не читается", chatID)
		}
		if !data.IsLegacy() {
			fmt.Printf("⏭ Чат %d: уже v%d, пропущен\n", chatID, data.SchemaVersion)
			continue
		}
		migrated, result, err := chatdata.MigrateV1(data, now)
		if err != nil {
			return fmt.Errorf("чат %d: %w", chatID, err)
		}
		migrated.MigrationComplete = true
		if err := migrated.ValidateV2(); err != nil {
			return fmt.Errorf("чат %d: %w", chatID, err)
		}
		candidates = append(candidates, migrationCandidate{chatID: chatID, result: result, migrated: migrated})
		fmt.Printf("🔎 Чат %d: книг %d, история %d, вишлист %d, проверить карточек %d\n", chatID, result.TotalBookCount, result.HistoryCount, result.WishlistCount, result.NeedsReview)
	}
	if len(candidates) == 0 {
		if onlyChatID != 0 {
			fmt.Printf("ℹ️ Для чата %d нет JSON v1, требующего миграции.\n", onlyChatID)
		} else {
			fmt.Println("ℹ️ Нет JSON v1, требующих миграции.")
		}
		return nil
	}
	if !apply {
		fmt.Printf("\n✅ Dry-run успешен. Файлов к миграции: %d.\n", len(candidates))
		fmt.Println("Для записи запустите ту же команду с --apply.")
		return nil
	}
	for _, candidate := range candidates {
		backupPath, err := storage.BackupChatData(candidate.chatID)
		if err != nil {
			return fmt.Errorf("чат %d: резервная копия: %w", candidate.chatID, err)
		}
		if err := storage.SaveChatData(candidate.chatID, candidate.migrated); err != nil {
			return fmt.Errorf("чат %d: запись v2: %w", candidate.chatID, err)
		}
		saved := storage.GetChatData(candidate.chatID)
		if saved == nil || saved.ValidateV2() != nil || len(saved.Books) != candidate.result.TotalBookCount {
			if restoreErr := storage.RestoreChatData(candidate.chatID, backupPath); restoreErr != nil {
				return fmt.Errorf("чат %d: проверка v2 и откат не удались: %v", candidate.chatID, restoreErr)
			}
			return fmt.Errorf("чат %d: v2 не прошёл проверку, выполнен откат", candidate.chatID)
		}
		fmt.Printf("✅ Чат %d: мигрирован, копия %s\n", candidate.chatID, backupPath)
	}
	fmt.Printf("\n✅ Миграция завершена. Обновлено JSON: %d.\n", len(candidates))
	return nil
}
