package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

const (
	dateFormat           = "2006-Jan-02"
	notificationInterval = 30 * time.Second
)

var (
	fixedDate         time.Time
	messages          = make(map[int64]string)
	waitingForCapsule = make(map[int64]bool)
)

func main() {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		log.Fatalf("Ошибка авторизации в боте: %v", err)
	}

	log.Printf("Авторизован в аккаунте %s", bot.Self.UserName)
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	fixedDate, err = time.Parse(dateFormat, "2025-Sep-02")
	if err != nil {
		log.Fatalf("Ошибка парсинга даты: %v", err)
	}
	log.Printf("Фиксированная дата: %s", fixedDate)

	go startNotifier(ctx, bot)

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates, err := bot.GetUpdatesChan(updateConfig)
	if err != nil {
		log.Fatalf("Ошибка получения обновлений: %v", err)
	}

	handleUpdates(bot, updates)
}

func handleUpdates(bot *tgbotapi.BotAPI, updates tgbotapi.UpdatesChannel) {
	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() {
			handleCommand(bot, update.Message)
		} else {
			handleMessage(bot, update.Message)
		}
	}
}

func handleCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	switch message.Command() {
	case "start":
		sendMessage(bot, message.Chat.ID, "Привет, дружок! 🌟 Я твой помощник по капсулам времени! Напиши свою капсулу времени, и я сохраню её для тебя!", getReplyMarkup())
	case "help":
		sendMessage(bot, message.Chat.ID, "🤗 Привет! Я здесь, чтобы помочь тебе с капсулами времени! Вот что ты можешь сделать:\n\n"+
			"1️⃣ **Написать капсулу**: Нажми на кнопку 'Написать капсулу', и я помогу тебе сохранить твои мысли и мечты на будущее!\n\n"+
			"2️⃣ **Получить капсулу**: Когда придет время, ты сможешь получить свою капсулу, нажав на кнопку 'Получить капсулу'. Я напомню тебе о ней!\n\n"+
			"Если у тебя есть вопросы, просто напиши мне, и я постараюсь помочь! 🌈", nil)
	}
}

func handleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	switch message.Text {
	case "Написать капсулу":
		waitingForCapsule[message.Chat.ID] = true
		sendMessage(bot, message.Chat.ID, "Ура! 🎉 Пожалуйста, напиши свою капсулу времени. Я с нетерпением жду твоих слов!", nil)

	case "Получить капсулу":
		handleRetrieveCapsule(bot, message.From.ID)

	default:
		if waitingForCapsule[message.Chat.ID] {
			messages[message.Chat.ID] = message.Text
			delete(waitingForCapsule, message.Chat.ID)
			sendMessage(bot, message.Chat.ID, "Капсула времени сохранена! 🎊 Теперь она будет ждать своего времени!", nil)
		} else {
			sendMessage(bot, message.Chat.ID, "Ой, я не совсем понял. Можешь попробовать еще раз? 🤔 Или можешь использовать /help для моей помощи", nil)
		}
	}
}

func handleRetrieveCapsule(bot *tgbotapi.BotAPI, userId int) {
	remainingTime := time.Until(fixedDate)

	if capsule := getCapsule(userId); capsule != "" {
		if remainingTime > 0 {
			daysRemaining := int(remainingTime.Hours() / 24)
			sendMessage(bot, int64(userId), fmt.Sprintf("Ой, подожди еще %d дней до 2 сентября 2025 года! ⏳ Но не переживай, твоя капсула скоро будет готова!", daysRemaining), nil)
		} else {
			sendCapsule(bot, userId)
		}
	} else {
		sendMessage(bot, int64(userId), "У тебя еще нет капсулы времени. Давай напишем одну вместе! ✍️", nil)
	}
}

func getReplyMarkup() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Написать капсулу"),
			tgbotapi.NewKeyboardButton("Получить капсулу"),
		),
	)
}

func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string, replyMarkup interface{}) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = replyMarkup
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}

func getCapsule(userId int) string {
	if capsule, exists := messages[int64(userId)]; exists {
		return capsule
	}
	return ""
}

func sendCapsule(bot *tgbotapi.BotAPI, userId int) {
	capsule := getCapsule(userId)
	if capsule != "" {
		sendMessage(bot, int64(userId), fmt.Sprintf("Вот твоя капсула времени🎈: %s. Надеюсь, она принесет тебе радость!", capsule), nil)
		delete(messages, int64(userId))
	}
}

func startNotifier(ctx context.Context, bot *tgbotapi.BotAPI) {
	ticker := time.NewTicker(notificationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			notifyUsers(bot)
		case <-ctx.Done():
			log.Println("[INFO] Notifier stopped")
			return
		}
	}
}

func notifyUsers(bot *tgbotapi.BotAPI) {
	for userId := range messages {
		sendCapsule(bot, int(userId))
	}
}
