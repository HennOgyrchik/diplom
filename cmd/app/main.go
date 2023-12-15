package main

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	db "project1"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//var waitingList = map[int64]chan *tgbotapi.Message{}

// var waitingList = map[int64]chan [2]interface{}{}
var waitingList = map[int64]map[string]chan *tgbotapi.Message{}

type responce struct {
	bot      *tgbotapi.BotAPI
	message  *tgbotapi.Message
	chatId   int64
	username string
}

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
		resp := responce{bot: bot}
		resp.username = update.FromChat().UserName

		var command string

		switch {
		case update.Message != nil:
			resp.message = update.Message
			command = update.Message.Command()
		case update.CallbackQuery != nil:
			resp.message = update.CallbackQuery.Message
			command = update.CallbackQuery.Data
		default:
			continue

		}

		resp.chatId = resp.message.Chat.ID

		if usrMap, ok := waitingList[resp.chatId]; ok { //есть ли функции ожидающие ответа от пользователя?
			if resp.message.Document != nil || resp.message.Photo != nil { // если ответ какой-то файл, сделай пометку о типе файла и отправь ответ в функцию
				usrMap["attachment"] <- resp.message
			} else {
				usrMap["text"] <- resp.message
			}
		} else { // если нет функций ожидающих ответа, запусти новую рутину
			//msg := tgbotapi.NewMessage(message.Chat.ID, message.Text)
			go commandSwitcher(resp, command)
		}
	}
}

func (r *responce) downloadAttachment(fileId string) {
	_, err := r.bot.GetFile(tgbotapi.FileConfig{FileID: fileId})
	if err != nil {
		return
	}

	pathFile, _ := r.bot.GetFileDirectURL(fileId)

	resp, err := http.Get(pathFile)
	defer resp.Body.Close()
	if err != nil {
		return
	}

	f, err := os.Create(path.Base(pathFile))
	defer f.Close()
	if err != nil {
		return
	}

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return
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

func commandSwitcher(resp responce, query string) {
	var paymentPat = regexp.MustCompile(`^оплатить\s\d*.`)
	var rejectionPat = regexp.MustCompile(`^отказ\s\d*.`)
	var waitingPat = regexp.MustCompile(`^ожидание\s\d*.`)
	var acceptPat = regexp.MustCompile(`^подтвердить\s\d*.`)

	switch cmd := query; {
	case cmd == "start":
		resp.startMenu()
	case cmd == "menu":
		resp.showMenu()
	case cmd == "создать":
		resp.confirmationCreationNewFund()
	case cmd == "присоединиться":
		resp.join()
	case cmd == "создать новый фонд":
		resp.creatingNewFund()
	case cmd == "баланс":
		resp.showBalance()
	case cmd == "test":
		//test(bot, msg, update)
	case cmd == "новый сбор":
		resp.createCashCollection()
	case cmd == "новое списание":
		resp.createDebitingFunds()
	case paymentPat.MatchString(cmd): // оплата
		cashCollectionId, err := strconv.Atoi(strings.Split(cmd, " ")[1])
		if err != nil {
			return
		}
		resp.payment(cashCollectionId)
	case acceptPat.MatchString(cmd): // подтверждение оплаты
		idTransaction, err := strconv.Atoi(strings.Split(cmd, " ")[1])
		if err != nil {
			return
		}
		resp.changeStatusOfTransaction(idTransaction, "подтвержден")
	case waitingPat.MatchString(cmd): // ожидание оплаты
		idTransaction, err := strconv.Atoi(strings.Split(cmd, " ")[1])
		if err != nil {
			return
		}
		resp.changeStatusOfTransaction(idTransaction, "ожидание")
	case rejectionPat.MatchString(cmd): // отказ оплаты
		idTransaction, err := strconv.Atoi(strings.Split(cmd, " ")[1])
		if err != nil {
			return
		}
		resp.changeStatusOfTransaction(idTransaction, "отказ")
	default:
		msg := tgbotapi.NewMessage(resp.chatId, "Я не знаю такую команду")
		if _, err := resp.bot.Send(msg); err != nil {
			panic(err)
		}
	}

}

func (r *responce) createDebitingFunds() {
	sum, err := r.getFloatFromUser("Введите сумму списания.")
	if err != nil {
		return
	}

	msg := tgbotapi.NewMessage(r.chatId, "Укажите причину списания")
	if _, err = r.bot.Send(msg); err != nil {
		return
	}

	purpose := r.waitingResponce("text").Text

	tag, err := db.GetTag(r.chatId)
	if err != nil {
		return
	}

	y, m, d := time.Now().Date()

	id, err := db.CreateCashCollection(tag, sum, "закрыт", fmt.Sprintf("Инициатор: %s", r.username), purpose, fmt.Sprintf("%d-%d-%d", y, m, d))
	if err != nil {
		msg.Text = "Произошла ошибка"
		_, _ = r.bot.Send(msg)
		return
	}

	msg.Text = "Прикрепите чек файлом"
	if _, err = r.bot.Send(msg); err != nil {
		return
	}
	////////////////////////////////////////////ожидание чека файлом или картинкой
	_ = r.waitingResponce("attachment")
	///////////////////////////////////////////////////////////////////////////
	msg.Text = "Списание оформлено. Все участники будут проинформированы"
	_, _ = r.bot.Send(msg)

	fmt.Println(id)
}

func (r *responce) changeStatusOfTransaction(idTransaction int, status string) {
	err := db.ChangeStatusTransaction(idTransaction, status)
	if err != nil {
		return
	}
	msg := tgbotapi.NewMessage(r.chatId, fmt.Sprintf("Статус оплаты: %s", status))
	_, _ = r.bot.Send(msg)

	r.paymentChangeStatusNotification(idTransaction)
}

func (r *responce) paymentChangeStatusNotification(idTransaction int) {
	status, _, _, memberId, _, err := db.InfoAboutTransaction(idTransaction)
	if err != nil {
		return
	}
	msg := tgbotapi.NewMessage(memberId, fmt.Sprintf("Статус оплаты изменен на: %s", status))
	_, _ = r.bot.Send(msg)
}

func (r *responce) payment(cashCollectionId int) {
	sum, err := r.getFloatFromUser("Введите сумму пополнения.")
	if err != nil {
		return
	}

	idTransaction, err := db.InsertInTransactions(cashCollectionId, sum, "пополнение", "ожидание", "", r.chatId)
	if err != nil {
		return
	}
	if err != nil {
		return
	}

	msg := tgbotapi.NewMessage(r.chatId, "Ваша оплата добавлена в очередь на подтверждение")
	if _, err = r.bot.Send(msg); err != nil {
		return
	}
	r.paymentNotification(idTransaction)

}

func (r *responce) paymentNotification(idTransaction int) { //доделать
	tag, err := db.GetTag(r.chatId)
	if err != nil {
		return
	}
	adminId, err := db.GetAdminFund(tag)
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

	_, _, _, memberId, sum, err := db.InfoAboutTransaction(idTransaction)

	_, _, name, err := db.GetInfoAboutMember(memberId)

	msg := tgbotapi.NewMessage(adminId, fmt.Sprintf("Подтвердите зачисление средств на счет фонда.\nСумма: %.2f\nОтправитель: %s", sum, name))
	msg.ReplyMarkup = &okKeyboard
	_, _ = r.bot.Send(msg)

}

func (r *responce) getFloatFromUser(message string) (sum float64, err error) {
	msg := tgbotapi.NewMessage(r.chatId, message)
	if _, err := r.bot.Send(msg); err != nil {
		return 0.0, err
	}

	sum = -0.0
	for {
		sum, err = strconv.ParseFloat(r.waitingResponce("text").Text, 64)
		if err != nil {
			msg = tgbotapi.NewMessage(r.chatId, "Попробуйте еще раз")
			if _, err = r.bot.Send(msg); err != nil {
				return
			}
			continue
		}
		break
	}
	return sum, nil
}

func (r *responce) createCashCollection() {
	sum, err := r.getFloatFromUser("Введите сумму сбора с одного участника.")
	if err != nil {
		return
	}

	msg := tgbotapi.NewMessage(r.chatId, "Укажите назначение сбора")
	if _, err = r.bot.Send(msg); err != nil {
		return
	}

	purpose := r.waitingResponce("text").Text

	tag, err := db.GetTag(r.chatId)
	if err != nil {
		return
	}

	id, err := db.CreateCashCollection(tag, sum, "открыт", fmt.Sprintf("Инициатор: %s", r.username), purpose, "")
	if err != nil {
		msg.Text = "Произошла ошибка"
		_, _ = r.bot.Send(msg)
		return
	}
	msg.Text = "Сбор создан. Сообщение о сборе будет отправлено всем участникам."
	_, _ = r.bot.Send(msg)

	r.collectionNotification(id, tag)
}

func (r *responce) collectionNotification(idCollection int, tagFund string) {
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
		_, _ = r.bot.Send(msg)
	}
}

func test() {

	t := time.Date(2009, time.February, 5, 23, 0, 0, 0, time.UTC)

	y, m, d := t.Date()
	fmt.Println(1, fmt.Sprintf("%d-%d-%d", y, m, d))
}

func (r *responce) showBalance() {
	tag, err := db.GetTag(r.chatId)
	if err != nil {
		return
	}
	balance, err := db.ShowBalance(tag)
	if err != nil {
		return
	}
	msg := tgbotapi.NewMessage(r.chatId, fmt.Sprintf("Текущий баланс: %.2f руб", balance))
	if _, err = r.bot.Send(msg); err != nil {
		return
	}

}

func (r *responce) startMenu() {
	var startKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Создать фонд", "создать"),
			tgbotapi.NewInlineKeyboardButtonData("Присоединиться", "присоединиться"),
			tgbotapi.NewInlineKeyboardButtonData("Тест", "test"),
		),
	)

	msg := tgbotapi.NewMessage(r.chatId, "Приветствую! Выберите один из вариантов")
	msg.ReplyMarkup = &startKeyboard
	if _, err := r.bot.Send(msg); err != nil {
		return
	}
}

func (r *responce) showMenu() {
	ok, err := db.IsMember(r.chatId)
	if err != nil {
		return
	}
	if !ok {
		msg := tgbotapi.NewMessage(r.chatId, "Вы не являетесь участником фонда. Создайте новый фонд или присоединитесь к существующему.")
		if _, err = r.bot.Send(msg); err != nil {
			return
		}
		r.startMenu()
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

	msg := tgbotapi.NewMessage(r.chatId, "Приветствую! Выберите один из вариантов")

	ok, err = db.IsAdmin(r.chatId)

	if err != nil {
		return
	}
	if ok {
		menuKeyboard.InlineKeyboard = append(menuKeyboard.InlineKeyboard,
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("админские функции", "9"),
				tgbotapi.NewInlineKeyboardButtonData("Новый сбор", "новый сбор"),
				tgbotapi.NewInlineKeyboardButtonData("Новое списание", "новое списание"),
				tgbotapi.NewInlineKeyboardButtonData("Участники", "участники")))
	}

	msg.ReplyMarkup = &menuKeyboard
	if _, err = r.bot.Send(msg); err != nil {
		fmt.Println(err)
		return
	}
}

func (r *responce) join() {
	msg := tgbotapi.NewMessage(r.chatId, "")
	ok, err := db.IsMember(r.chatId)
	if err != nil {
		return
	}
	if ok {
		msg.Text = "Вы уже являетесь участником фонда"
		if _, err = r.bot.Send(msg); err != nil {
			fmt.Println(err)
		}
		return
	}
	msg.Text = "Введите тег фонда. Если у вас нет тега, запросите его у администратора фонда."
	if _, err = r.bot.Send(msg); err != nil {
		fmt.Println(err)
		return
	}
	tag := r.waitingResponce("text").Text

	ok, err = db.ExistsFund(tag)
	if err != nil {
		return
	}
	if !ok {
		msg.Text = "Фонд с таким тегом не найден."
	} else {
		err = db.AddMember(tag, r.chatId, false, r.username, r.getName())
		if err != nil {
			return
		}
		msg.Text = "Вы успешно присоединились к фонду."
	}

	if _, err = r.bot.Send(msg); err != nil {
		fmt.Println(err)
		return
	}

}

func (r *responce) confirmationCreationNewFund() {
	msg := tgbotapi.NewMessage(r.chatId, "")
	ok, err := db.IsMember(r.chatId)
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
	if _, err = r.bot.Send(msg); err != nil {
		panic(err)
	}
}

func (r *responce) creatingNewFund() {
	var err error
	sum, err := r.getFloatFromUser("Введите начальную сумму фонда без указания валюты.")
	if err != nil {
		return
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
	name := r.getName()
	err = db.AddMember(tag, r.chatId, true, r.username, name)
	if err != nil {
		return
	}

	msg := tgbotapi.NewMessage(r.chatId, fmt.Sprintf("Новый фонд создан успешно! Присоединиться к фонду можно, используя тег: %s \nВнимание! Не показывайте этот тег посторонним людям.", tag))
	if _, err = r.bot.Send(msg); err != nil {
		return
	}

}

func (r *responce) waitingResponce(obj string) *tgbotapi.Message {
	waitingList[r.chatId] = map[string]chan *tgbotapi.Message{}
	defer delete(waitingList, r.chatId)

	waitingList[r.chatId][obj] = make(chan *tgbotapi.Message)
	defer close(waitingList[r.chatId][obj])

	return <-waitingList[r.chatId][obj]

}

func newTag() string {
	symbols := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	result := make([]byte, rand.Intn(5)+5)
	for i := range result {
		result[i] = symbols[rand.Intn(len(symbols))]
	}
	return string(result)
}

func (r *responce) getName() string {
	msg := tgbotapi.NewMessage(r.chatId, "Представьтесь, пожалуйста. Введите ФИО")
	if _, err := r.bot.Send(msg); err != nil {
		return ""
	}
	return r.waitingResponce("text").Text
}
