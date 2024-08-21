package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var (
	shortForm         = "2006-Jan-02"
	fixedDate         time.Time
	notifyTime        = 30 * time.Second
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

	fixedDate, err = time.Parse(shortForm, "2025-Sep-02")
	if err != nil {
		log.Fatal(err)
	}

	go func(ctx context.Context, bot *tgbotapi.BotAPI) {
		if err = startNotifier(ctx, bot); err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Printf("Error starting notifier: %v", err)
				return
			}
			log.Printf("[INFO] notifier stopped")
		}
	}(ctx, bot)

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates, err := bot.GetUpdatesChan(updateConfig)
	if err != nil {
		log.Fatalf("Ошибка получения обновлений: %v", err)
	}

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
		msg := tgbotapi.NewMessage(message.Chat.ID, "Привет, дружок! 🌟 Я твой помощник по капсулам времени! Напиши свою капсулу времени, и я сохраню её для тебя!")
		msg.ReplyMarkup = getReplyMarkup()
		sendMessage(bot, msg)
	case "help":
		msg := tgbotapi.NewMessage(message.Chat.ID, "🤗 Привет! Я здесь, чтобы помочь тебе с капсулами времени! Вот что ты можешь сделать:\n\n"+
			"1️⃣ **Написать капсулу**: Нажми на кнопку 'Написать капсулу', и я помогу тебе сохранить твои мысли и мечты на будущее!\n\n"+
			"2️⃣ **Получить капсулу**: Когда придет время, ты сможешь получить свою капсулу, нажав на кнопку 'Получить капсулу'. Я напомню тебе о ней!\n\n"+
			"Если у тебя есть вопросы, просто напиши мне, и я постараюсь помочь! 🌈")
		sendMessage(bot, msg)
	}

}

func handleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	switch message.Text {
	case "Написать капсулу":
		msg := tgbotapi.NewMessage(message.Chat.ID, "Ура! 🎉 Пожалуйста, напиши свою капсулу времени. Я с нетерпением жду твоих слов!")
		sendMessage(bot, msg)
		waitingForCapsule[message.Chat.ID] = true

	case "Получить капсулу":
		userId := message.From.ID
		SendMessageCapsule(bot, userId)
	default:
		if waiting, ok := waitingForCapsule[message.Chat.ID]; ok && waiting {
			messages[message.Chat.ID] = message.Text
			if _, err := bot.DeleteMessage(tgbotapi.DeleteMessageConfig{
				ChatID:    message.Chat.ID,
				MessageID: message.MessageID,
			}); err != nil {
				log.Printf("Ошибка удаления сообщения: %v", err)
			}

			msg := tgbotapi.NewMessage(message.Chat.ID, "Капсула времени сохранена! 🎊 Теперь она будет ждать своего времени!")
			sendMessage(bot, msg)
			delete(waitingForCapsule, message.Chat.ID)

		} else {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Ой, я не совсем понял. Можешь попробовать еще раз? 🤔 Или можешь использовать /help для моей помощи")
			sendMessage(bot, msg)
		}
	}
}

func getReplyMarkup() tgbotapi.ReplyKeyboardMarkup {
	btn1 := tgbotapi.NewKeyboardButton("Написать капсулу")
	btn2 := tgbotapi.NewKeyboardButton("Получить капсулу")

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(btn1, btn2),
	)
	return keyboard
}

func sendMessage(bot *tgbotapi.BotAPI, msg tgbotapi.MessageConfig) {
	if _, err := bot.Send(msg); err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}

func SendMessageCapsule(bot *tgbotapi.BotAPI, userId int) {
	if capsule, ok := messages[int64(userId)]; ok {
		remainingTime := time.Until(fixedDate)
		if remainingTime > 0 {
			daysRemaining := int(remainingTime.Hours() / 24)
			msg := tgbotapi.NewMessage(int64(userId),
				fmt.Sprintf("Ой, подожди еще %d дней до 2 сентября 2025 года! ⏳ Но не переживай, твоя капсула скоро будет готова!", daysRemaining))
			sendMessage(bot, msg)
		} else {
			msg := tgbotapi.NewMessage(int64(userId), fmt.Sprintf("Вот твоя капсула времени: %s 🎈 Надеюсь, она принесет тебе радость!", capsule))
			sendMessage(bot, msg)
		}
	} else {
		msg := tgbotapi.NewMessage(int64(userId), "У тебя еще нет капсулы времени. Давай напишем одну вместе! ✍️")
		sendMessage(bot, msg)
	}
}

func startNotifier(ctx context.Context, bot *tgbotapi.BotAPI) error {
	ticker := time.NewTicker(notifyTime)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			Notify(bot)
		case <-ctx.Done():
			return ctx.Err()
		}
	}

}

func Notify(bot *tgbotapi.BotAPI) {
	for userId, _ := range messages {
		SendMessageCapsule(bot, int(userId))
	}
}
