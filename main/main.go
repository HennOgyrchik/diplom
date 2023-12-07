package main

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

var memory = map[int64]chan string{}

func main() {

	bot, err := tgbotapi.NewBotAPI(getToken())
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = false

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			if usrChan, ok := memory[update.Message.Chat.ID]; ok {
				usrChan <- update.Message.Text
			} else {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
				go commandSwitcher(bot, &msg, update.Message.Command())
			}

		} else if update.CallbackQuery != nil {
			if usrChan, ok := memory[update.CallbackQuery.Message.Chat.ID]; ok {
				usrChan <- update.Message.Text
			} else {
				msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.Text)

				go commandSwitcher(bot, &msg, update.CallbackQuery.Data)
			}
		}
	}
}

func getToken() string {
	file, err := os.Open("token.txt")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	file.Close()

	token, err := ioutil.ReadFile("token.txt")
	if err != nil {
		panic(err)
	}
	return string(token[:])
}

func commandSwitcher(bot *tgbotapi.BotAPI, msg *tgbotapi.MessageConfig, query string) {
	var numericKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Создать", "create"),
			tgbotapi.NewInlineKeyboardButtonData("Присоединиться", "join"),
		),
	)
	switch query {

	case "start":
		msg.Text = "Приветствую! Выберите один из вариантов"
		msg.ReplyMarkup = &numericKeyboard
		if _, err := bot.Send(msg); err != nil {
			panic(err)
		}
	case "create":
		confirmationCreationNewFund(bot, msg.ChatID)
	case "join":
		msg.Text = "Присоединение"
		if _, err := bot.Send(msg); err != nil {
			panic(err)
		}
	case "Создать новый фонд":
		creatingNewFund(bot, msg.ChatID)

	case "Не создавать новый фонд":
		msg.Text = "Не создаю"
		if _, err := bot.Send(msg); err != nil {
			panic(err)
		}

	default:
		msg.Text = "Я не знаю такую команду"
		if _, err := bot.Send(msg); err != nil {
			panic(err)
		}
	}

}

func confirmationCreationNewFund(bot *tgbotapi.BotAPI, chatId int64) {
	msg := tgbotapi.NewMessage(chatId, "Вы уверены, что хотите создать новый чат?")

	var numericKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Да", "Создать новый фонд"),
			tgbotapi.NewInlineKeyboardButtonData("Нет", "Не создавать новый фонд"),
		),
	)
	msg.ReplyMarkup = numericKeyboard

	if _, err := bot.Send(msg); err != nil {
		panic(err)
	}
}

func creatingNewFund(bot *tgbotapi.BotAPI, chatId int64) {
	var err error
	memory[chatId] = make(chan string)
	fmt.Println(memory)
	msg := tgbotapi.NewMessage(chatId, "Введите начальную сумму фонда без указания валюты. Например: 50.25")
	if _, err = bot.Send(msg); err != nil {
		panic(err)
	}
	var sum float64
	for {
		sum, err = strconv.ParseFloat(<-memory[chatId], 64)
		if err != nil {
			msg = tgbotapi.NewMessage(chatId, "Попробуйте еще раз")
			if _, err = bot.Send(msg); err != nil {
				panic(err)
			}
			continue
		}
		break
	}
	close(memory[chatId])
	delete(memory, chatId)
	msg = tgbotapi.NewMessage(chatId, strconv.FormatFloat(sum, 'f', -1, 64))
	if _, err := bot.Send(msg); err != nil {
		panic(err)
	}
	fmt.Println(memory)
}
