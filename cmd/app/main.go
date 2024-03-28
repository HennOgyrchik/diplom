package main

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"os"
	"os/signal"
	"project1/internal/chat"
	"project1/internal/service"
	"syscall"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	defer cancel()

	file, err := os.OpenFile("log.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Failed to open log file:", err)
	}
	log.SetOutput(file)

	srv, err := service.NewService(ctx)

	if err != nil {
		log.Println("main/NewService: ", err)
		return
	}

	go func() {
		<-ctx.Done()

		if err := srv.FTP.Close(); err != nil {
			log.Println("FTP Close: ", err)
		}
		srv.DB.Close()
		if err = file.Close(); err != nil {
			log.Println("LogFile Close: ", err)
		}
		os.Exit(1)

	}()

	bot := srv.GetBot()

	fmt.Printf("Authorized on account %s", bot.Self.UserName)
	log.Println("Authorized on account ", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		go handlerUpdate(update, srv)
	}
}

func handlerUpdate(update tgbotapi.Update, srv *service.Service) {
	var command string
	var message *tgbotapi.Message

	switch {
	case update.Message != nil:
		message = update.Message
		command = update.Message.Command()
	case update.CallbackQuery != nil:
		message = update.CallbackQuery.Message
		command = update.CallbackQuery.Data
	default:
		return
	}

	ch := chat.NewChat(update.FromChat().UserName, message.Chat.ID, srv)

	if userChan, ok := srv.GetUserChan(ch.GetChatId()); ok { //есть ли функции ожидающие ответа от пользователя?
		if !ch.CommandSwitcher(command) { //функция ждет ответ, проверь ответ это команда? Если это так, то она запустится
			userChan <- message //ответ не команда, отправь полученное сообщение в канал
			return
		}
		userChan <- nil
		return
	}

	if !ch.CommandSwitcher(command) {
		_ = ch.Send(tgbotapi.NewMessage(ch.GetChatId(), "Я не знаю такую команду"))
	}

}
