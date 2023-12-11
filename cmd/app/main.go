package main

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	db "project1"
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
	case "menu":
		msg.Text = "Какое-то кнопочное меню"
		if _, err := bot.Send(msg); err != nil {
			panic(err)
		}
	case "create":
		confirmationCreationNewFund(bot, msg.ChatID)
	case "join":
		test(bot, msg.ChatID)
	case "Создать новый фонд":
		creatingNewFund(bot, msg.ChatID)

	default:
		msg.Text = "Я не знаю такую команду"
		if _, err := bot.Send(msg); err != nil {
			panic(err)
		}
	}

}

func test(bot *tgbotapi.BotAPI, chatId int64) {
	msg := tgbotapi.NewMessage(chatId, "Тестовая функция")
	if _, err := bot.Send(msg); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(chatId)
}

func confirmationCreationNewFund(bot *tgbotapi.BotAPI, chatId int64) {
	msg := tgbotapi.NewMessage(chatId, "")
	ok, err := db.IsMember(chatId)
	if err != nil {
		return
	}
	if ok {
		msg.Text = "Вы уже являетесь участником фонда" //подумать куда выйти/ в какое меню
	} else {
		msg.Text = "Вы уверены, что хотите создать новый чат?"
		var numericKeyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Да", "Создать новый фонд"),
				tgbotapi.NewInlineKeyboardButtonData("Нет", "start"),
			),
		)
		msg.ReplyMarkup = numericKeyboard
	}
	if _, err = bot.Send(msg); err != nil {
		panic(err)
	}
}

func creatingNewFund(bot *tgbotapi.BotAPI, chatId int64) {
	var err error

	msg := tgbotapi.NewMessage(chatId, "Введите начальную сумму фонда без указания валюты. В качестве разделителя используйте точку. Например: 50.25")
	if _, err = bot.Send(msg); err != nil {
		return
	}

	var sum float64
	for {
		sum, err = strconv.ParseFloat(waitingResponce(chatId), 64)
		if err != nil {
			msg = tgbotapi.NewMessage(chatId, "Попробуйте еще раз")
			if _, err = bot.Send(msg); err != nil {
				return
			}
			continue
		}
		break
	}

	var tag string
	for i := 0; i < 10; i++ {
		tag = newTag()

		ok, err := db.DoesTagExist(tag)
		if err != nil {
			return
		}
		if !ok {
			continue
		}
		break
	}

	err = db.CreateFund(tag, sum)
	if err != nil {
		return
	}

	err = db.AddMember(tag, chatId, true)
	if err != nil {
		return
	}

	msg = tgbotapi.NewMessage(chatId, fmt.Sprintf("Новый фонд создан успешно! Присоединиться к фонду можно, используя тег: %s \nВнимание! Не показывайте этот тег посторонним людям.", tag))
	if _, err := bot.Send(msg); err != nil {
		return
	}

}

func waitingResponce(chatId int64) string { //организовать блокировку одновременного обращения к мапе
	memory[chatId] = make(chan string)
	defer delete(memory, chatId)
	defer close(memory[chatId])
	return <-memory[chatId]
}

func newTag() string {
	symbols := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	result := make([]byte, rand.Intn(5)+5)
	for i := range result {
		result[i] = symbols[rand.Intn(len(symbols))]
	}
	return string(result)
}
