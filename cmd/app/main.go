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
	switch query {
	case "start":
		startMenu(bot, msg.ChatID)
	case "menu":
		showMenu(bot, msg.ChatID)
	case "создать":
		confirmationCreationNewFund(bot, msg.ChatID)
	case "присоединиться":
		join(bot, msg.ChatID)
	case "создать новый фонд":
		creatingNewFund(bot, msg.ChatID)
	case "баланс":
		showBalance(bot, msg.ChatID)
	default:
		msg.Text = "Я не знаю такую команду"
		if _, err := bot.Send(msg); err != nil {
			panic(err)
		}
	}

}

func showBalance(bot *tgbotapi.BotAPI, chatId int64) {
	tag, err := db.GetTag(chatId)
	if err != nil {
		return
	}
	balance, err := db.ShowBalance(tag)
	if err != nil {
		return
	}
	msg := tgbotapi.NewMessage(chatId, fmt.Sprintf("Текущий баланс: %.2f руб", balance))
	if _, err := bot.Send(msg); err != nil {
		return
	}

}

func startMenu(bot *tgbotapi.BotAPI, chatId int64) {
	var startKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Создать", "создать"),
			tgbotapi.NewInlineKeyboardButtonData("Присоединиться", "присоединиться"),
		),
	)

	msg := tgbotapi.NewMessage(chatId, "Приветствую! Выберите один из вариантов")
	msg.ReplyMarkup = &startKeyboard
	if _, err := bot.Send(msg); err != nil {
		return
	}
}

func showMenu(bot *tgbotapi.BotAPI, chatId int64) {
	var menuKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Баланс", "баланс"),
			tgbotapi.NewInlineKeyboardButtonData("Оплатить", "1"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Статистика", "2"),
			tgbotapi.NewInlineKeyboardButtonData("Покинуть фонд", "3"),
		),
	)
	//adminMenuKeyboard := append(menuKeyboard.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("админские функции", ""),) )

	msg := tgbotapi.NewMessage(chatId, "Приветствую! Выберите один из вариантов")
	msg.ReplyMarkup = &menuKeyboard
	if _, err := bot.Send(msg); err != nil {
		return
	}
}

func join(bot *tgbotapi.BotAPI, chatId int64) {
	msg := tgbotapi.NewMessage(chatId, "")
	ok, err := db.IsMember(chatId)
	if err != nil {
		return
	}
	if ok {
		msg.Text = "Вы уже являетесь участником фонда"
		if _, err = bot.Send(msg); err != nil {
			fmt.Println(err)
		}
		return
	}
	msg.Text = "Введите тег фонда. Если у вас нет тега, запросите его у администратора фонда."
	if _, err = bot.Send(msg); err != nil {
		fmt.Println(err)
		return
	}
	tag := waitingResponce(chatId)

	ok, err = db.ExistsFund(tag)
	if err != nil {
		return
	}
	if !ok {
		msg.Text = "Фонд с таким тегом не найден."
	} else {
		err = db.AddMember(tag, chatId, false)
		if err != nil {
			return
		}
		msg.Text = "Вы успешно присоединились к фонду."
	}

	if _, err = bot.Send(msg); err != nil {
		fmt.Println(err)
		return
	}

}

func confirmationCreationNewFund(bot *tgbotapi.BotAPI, chatId int64) {
	msg := tgbotapi.NewMessage(chatId, "")
	ok, err := db.IsMember(chatId)
	if err != nil {
		return
	}
	if ok {
		msg.Text = "Вы уже являетесь участником фонда"
	} else {
		msg.Text = "Вы уверены, что хотите создать новый чат?"
		var numericKeyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Да", "создать новый фонд"),
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
