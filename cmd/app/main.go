package main

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	db "project1"
	"regexp"
	"strconv"
	"strings"
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
				go commandSwitcher(bot, msg, update.Message.Command(), update)
			}

		} else if update.CallbackQuery != nil {
			if usrChan, ok := memory[update.CallbackQuery.Message.Chat.ID]; ok {
				usrChan <- update.Message.Text
			} else {
				msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.Text)
				go commandSwitcher(bot, msg, update.CallbackQuery.Data, update)
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

func commandSwitcher(bot *tgbotapi.BotAPI, msg tgbotapi.MessageConfig, query string, update tgbotapi.Update) {
	var commandPat = regexp.MustCompile(`^оплатить\s\d*.`)
	switch cmd := query; {
	case cmd == "start":
		startMenu(bot, msg.ChatID)
	case cmd == "menu":
		showMenu(bot, msg.ChatID)
	case cmd == "создать":
		confirmationCreationNewFund(bot, msg.ChatID)
	case cmd == "присоединиться":
		join(bot, msg.ChatID, update)
	case cmd == "создать новый фонд":
		creatingNewFund(bot, msg.ChatID, update)
	case cmd == "баланс":
		showBalance(bot, msg.ChatID)
	case cmd == "test":
		test(bot, msg, update)
	case cmd == "новый сбор":
		createCashCollection(bot, msg.ChatID, update)
	case commandPat.MatchString(cmd): // оплата
		cashCollectionId, err := strconv.Atoi(strings.Split(cmd, " ")[1])
		if err != nil {
			return
		}
		payment(bot, msg.ChatID, update, cashCollectionId)
	default:
		msg.Text = "Я не знаю такую команду"
		if _, err := bot.Send(msg); err != nil {
			panic(err)
		}
	}

}

func payment(bot *tgbotapi.BotAPI, chatId int64, update tgbotapi.Update, cashCollectionId int) {
	sum, err := getFloatFromUser(bot, chatId, "Введите сумму пополнения без указания валюты. В качестве разделителя используйте точку.")
	if err != nil {
		return
	}

	idTransaction, err := db.InsertInTransactions(cashCollectionId, sum, "пополнение", "ожидание", "", chatId)
	if err != nil {
		return
	}

	msg := tgbotapi.NewMessage(chatId, "Ваша оплата добавлена в очередь на подтверждение")
	if _, err = bot.Send(msg); err != nil {
		return
	}
	paymentNotification(bot, chatId, idTransaction)

}

func paymentNotification(bot *tgbotapi.BotAPI, chatId int64, idTransaction int) {
	tag, err := db.GetTag(chatId)
	if err != nil {
		return
	}
	memberId, err := db.GetAdminFund(tag)
	if err != nil {
		return
	}

	var okKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Подтвердить", fmt.Sprintf("подтвердить %d", idTransaction)),
			tgbotapi.NewInlineKeyboardButtonData("Отказ", fmt.Sprintf("отказ %d", idTransaction)),
			tgbotapi.NewInlineKeyboardButtonData("Ожидание", fmt.Sprintf("ожидание %d", idTransaction)),
		),
	)
	msg := tgbotapi.NewMessage(memberId, fmt.Sprintf("Подтвердите зачисление средств на счет фонда.\nСумма: %.2f\nОтправитель: %s", sum, purpose))
	msg.ReplyMarkup = &okKeyboard
	_, _ = bot.Send(msg)

}

func getFloatFromUser(bot *tgbotapi.BotAPI, chatId int64, message string) (sum float64, err error) {
	msg := tgbotapi.NewMessage(chatId, message)
	if _, err := bot.Send(msg); err != nil {
		return 0.0, err
	}

	sum = -0.0
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
	return sum, nil
}

func createCashCollection(bot *tgbotapi.BotAPI, chatId int64, update tgbotapi.Update) {
	var err error

	msg := tgbotapi.NewMessage(chatId, "Введите сумму сбора с одного участника без указания валюты. В качестве разделителя используйте точку.")
	if _, err = bot.Send(msg); err != nil {
		return
	}

	var sum float64
	for {
		sum, err = strconv.ParseFloat(waitingResponce(chatId), 64)
		if err != nil {
			msg.Text = "Попробуйте еще раз"
			if _, err = bot.Send(msg); err != nil {
				return
			}
			continue
		}
		break
	}
	msg.Text = "Укажите назначение сбора"
	if _, err = bot.Send(msg); err != nil {
		return
	}

	purpose := waitingResponce(chatId)

	tag, err := db.GetTag(chatId)
	if err != nil {
		return
	}

	id, err := db.CreateCashCollection(tag, sum, fmt.Sprintf("Инициатор: %s", update.FromChat().UserName), purpose)
	if err != nil {
		msg.Text = "Произошла ошибка"
		_, _ = bot.Send(msg)
		return
	}
	msg.Text = "Сбор создан. Сообщение о сборе будет отправлено всем участникам."
	_, _ = bot.Send(msg)

	collectionNotification(bot, id, tag)
}

func collectionNotification(bot *tgbotapi.BotAPI, idCollection int, tagFund string) {
	members, err := db.SelectMembers(tagFund)
	if err != nil {
		return
	}
	sum, purpose, err := db.InfoAboutCashCollection(idCollection)
	if err != nil {
		return
	}

	for _, value := range members {
		var paymentKeyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Оплатить", fmt.Sprintf("оплатить %d", idCollection)),
			),
		)
		msg := tgbotapi.NewMessage(value, fmt.Sprintf("Иницирован новый сбор.\nСумма к оплате: %.2f\nНазначение: %s", sum, purpose))
		msg.ReplyMarkup = &paymentKeyboard
		_, _ = bot.Send(msg)
	}
}

func test(bot *tgbotapi.BotAPI, msg tgbotapi.MessageConfig, update tgbotapi.Update) {
	fmt.Println(1, update.FromChat().UserName)
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
			tgbotapi.NewInlineKeyboardButtonData("Создать фонд", "создать"),
			tgbotapi.NewInlineKeyboardButtonData("Присоединиться", "присоединиться"),
			tgbotapi.NewInlineKeyboardButtonData("Тест", "test"),
		),
	)

	msg := tgbotapi.NewMessage(chatId, "Приветствую! Выберите один из вариантов")
	msg.ReplyMarkup = &startKeyboard
	if _, err := bot.Send(msg); err != nil {
		return
	}
}

func showMenu(bot *tgbotapi.BotAPI, chatId int64) {
	ok, err := db.IsMember(chatId)
	if err != nil {
		return
	}
	if !ok {
		msg := tgbotapi.NewMessage(chatId, "Вы не являетесь участником фонда. Создайте новый фонд или присоединитесь к существующему.")
		if _, err = bot.Send(msg); err != nil {
			return
		}
		startMenu(bot, chatId)
		return
	}

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

	msg := tgbotapi.NewMessage(chatId, "Приветствую! Выберите один из вариантов")

	ok, err = db.IsAdmin(chatId)

	if err != nil {
		return
	}
	if ok {
		fmt.Println(1)
		menuKeyboard.InlineKeyboard = append(menuKeyboard.InlineKeyboard,
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("админские функции", "9"),
				tgbotapi.NewInlineKeyboardButtonData("Создать сбор", "новый сбор"),
				tgbotapi.NewInlineKeyboardButtonData("Участники", "участники")))
	}

	msg.ReplyMarkup = &menuKeyboard
	if _, err = bot.Send(msg); err != nil {
		fmt.Println(err)
		return
	}
}

func join(bot *tgbotapi.BotAPI, chatId int64, update tgbotapi.Update) {
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
		err = db.AddMember(tag, chatId, false, update.FromChat().UserName)
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
		msg.Text = "Вы уверены, что хотите создать новый фонд?"
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

func creatingNewFund(bot *tgbotapi.BotAPI, chatId int64, update tgbotapi.Update) {
	var err error
	sum, err := getFloatFromUser(bot, chatId, "Введите начальную сумму фонда без указания валюты. В качестве разделителя используйте точку.")
	if err != nil {
		return
	}
	/*msg := tgbotapi.NewMessage(chatId, "Введите начальную сумму фонда без указания валюты. В качестве разделителя используйте точку. Например: 50.25")
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
	}*/

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

	err = db.AddMember(tag, chatId, true, update.FromChat().UserName)
	if err != nil {
		return
	}

	msg := tgbotapi.NewMessage(chatId, fmt.Sprintf("Новый фонд создан успешно! Присоединиться к фонду можно, используя тег: %s \nВнимание! Не показывайте этот тег посторонним людям.", tag))
	if _, err = bot.Send(msg); err != nil {
		return
	}

}

func waitingResponce(chatId int64) string {
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
