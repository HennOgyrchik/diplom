package main

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"os"
	"project1/cmd/chat"
	"project1/cmd/service"
)

//var waitingList = map[int64]chan *tgbotapi.Message{}
//
//type response struct {
//	bot      *tgbotapi.BotAPI
//	message  *tgbotapi.Message
//	chatId   int64
//	username string
//}

func main() {
	token, err := getToken("token.txt") // проверить работает ли с несуществующим файлом
	if err != nil {
		fmt.Println(err)
		return
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		fmt.Println(err)
		return
	}

	bot.Debug = false

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	srv := service.NewService(bot)

	updates := bot.GetUpdatesChan(u)

	for update := range updates {

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
			continue
		}

		chat := chat.NewChat(update.FromChat().UserName, message.Chat.ID, srv)

		if usrMap, ok := srv.GetWaitingList()[chat.GetChatId()]; ok { //есть ли функции ожидающие ответа от пользователя?
			usrMap <- chat.GetMessage() //если есть, отправь полученное сообщение в канал
		} else { // если нет функций ожидающих ответа, запусти новую рутину
			go chat.CommandSwitcher(command)
		}
	}
}

func getToken(filename string) (string, error) {
	token, err := os.ReadFile(filename)
	return string(token[:]), err
}
