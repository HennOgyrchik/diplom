package main

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"os"
	"os/signal"
	"project1/cmd/chat"
	"project1/cmd/service"
)

func main() {
	file, err := os.OpenFile("log.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Failed to open log file:", err)
	}
	log.SetOutput(file)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt)
	go func() {
		for {
			select {
			case <-done:
				fmt.Println("exit")
				_ = file.Close()
				os.Exit(0)
			default:

			}
		}
	}()

	token, err := getToken("token.txt") // проверить работает ли с несуществующим файлом
	if err != nil {
		fmt.Println(err)
		log.Println("main/getToken: ", err)
		return
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		fmt.Println(err)
		log.Println("main/newBotAPI: ", err)
		return
	}
	bot.Debug = false

	srv := service.NewService(bot)

	fmt.Printf("Authorized on account %s", bot.Self.UserName)
	log.Println("Authorized on account ", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		go handlerUpdate(update, srv)
	}
}

func getToken(filename string) (string, error) {
	token, err := os.ReadFile(filename)
	return string(token[:]), err
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

	chat := chat.NewChat(update.FromChat().UserName, message.Chat.ID, srv, message)

	if userChan, ok := srv.GetUserChan(chat.GetChatId()); ok { //есть ли функции ожидающие ответа от пользователя?
		userChan <- chat.GetMessage() //если есть, отправь полученное сообщение в канал
	} else { // если нет функций ожидающих ответа, запусти новую рутину
		go chat.CommandSwitcher(command)
	}
}
