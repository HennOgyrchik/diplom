package chat

import (
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"math/rand"
	"project1/cmd/service"
	"project1/db"
	"regexp"
	"strconv"
	"time"
)

type Chat struct {
	username string
	chatId   int64
	msg      *tgbotapi.Message
	*service.Service
}

func (c *Chat) test() error { // 3 попытки на отправку
	var err error

	for i := 0; i < 3; i++ {
		if err = c.send(tgbotapi.NewMessage(c.chatId, "тестовая кнопка")); err == nil {
			return err
		}
	}
	return err
}

func NewChat(username string, chatId int64, service *service.Service, message *tgbotapi.Message) *Chat {
	return &Chat{
		username: username,
		chatId:   chatId,
		msg:      message,
		Service:  service,
	}
}

func (c *Chat) GetChatId() int64 {
	return c.chatId
}

func (c *Chat) GetMessage() *tgbotapi.Message {
	return c.msg
}

// send 3 попытки на отправку, иначе удалить из списка ожидания и вернуть ошибку
func (c *Chat) send(message tgbotapi.MessageConfig) error {
	var err error

	for i := 0; i < 3; i++ {
		if _, err = c.Service.GetBot().Send(message); err == nil {
			return nil
		}
	}

	wList := c.Service.GetWaitingList()
	if _, ok := wList[c.chatId]; ok { //если ожидается ввод от пользователя, прекратить ожидание
		c.Service.DeleteFromWaitingList(c.chatId)
	}

	return AttemptsExceeded
}

func (c *Chat) CommandSwitcher(query string) {
	var paymentPat = regexp.MustCompile(`^оплатить\s\d*.`)
	var rejectionPat = regexp.MustCompile(`^отказ\s\d*.`)
	var waitingPat = regexp.MustCompile(`^ожидание\s\d*.`)
	var acceptPat = regexp.MustCompile(`^подтвердить\s\d*.`)

	switch cmd := query; {
	case cmd == "start":
		c.startMenu()
	case cmd == "menu":
		c.showMenu()
	case cmd == "создать":
		c.confirmationCreationNewFund()
	case cmd == "присоединиться":
		//c.join()
	case cmd == "создать новый фонд":
		c.creatingNewFund()
	case cmd == "баланс":
		c.showBalance()
	case cmd == "test":
		_ = c.test()
	case cmd == "участники":
		//c.getMembers()
	case cmd == "новый сбор":
		//c.createCashCollection()
	case cmd == "новое списание":
		//c.createDebitingFunds()
	case paymentPat.MatchString(cmd): // оплата
		//cashCollectionId, err := strconv.Atoi(strings.Split(cmd, " ")[1])
		//if err != nil {
		//	c.notificationAboutError("Произошла ошибка. Попробуйте еще раз.")
		//	return
		//}
		//c.payment(cashCollectionId)
	case acceptPat.MatchString(cmd): // подтверждение оплаты
		//idTransaction, err := strconv.Atoi(strings.Split(cmd, " ")[1])
		//if err != nil {
		//	c.notificationAboutError("Произошла ошибка. Попробуйте еще раз.")
		//	return
		//}
		//c.changeStatusOfTransaction(idTransaction, "подтвержден")
	case waitingPat.MatchString(cmd): // ожидание оплаты
		//idTransaction, err := strconv.Atoi(strings.Split(cmd, " ")[1])
		//if err != nil {
		//	c.notificationAboutError("Произошла ошибка. Попробуйте еще раз.")
		//	return
		//}
		//c.changeStatusOfTransaction(idTransaction, "ожидание")
	case rejectionPat.MatchString(cmd): // отказ оплаты
		//idTransaction, err := strconv.Atoi(strings.Split(cmd, " ")[1])
		//if err != nil {
		//	c.notificationAboutError("Произошла ошибка. Попробуйте еще раз.")
		//	return
		//}
		//c.changeStatusOfTransaction(idTransaction, "отказ")
	default:
		_, _ = c.GetBot().Send(tgbotapi.NewMessage(c.chatId, "Я не знаю такую команду"))
	}

}

func (c *Chat) startMenu() {
	var startKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Создать фонд", "создать"),
			tgbotapi.NewInlineKeyboardButtonData("Присоединиться", "присоединиться"),
			tgbotapi.NewInlineKeyboardButtonData("Тест", "test"),
		),
	)

	msg := tgbotapi.NewMessage(c.chatId, "Приветствую! Выберите один из вариантов")
	msg.ReplyMarkup = &startKeyboard

	_ = c.send(msg)
}

func (c *Chat) showMenu() {
	ok, err := db.IsMember(c.chatId)
	if err != nil {
		log.Println(time.Now(), c.chatId, err)
		c.sendError()
		return
	}
	if !ok {
		if err = c.send(tgbotapi.NewMessage(c.chatId, "Вы не являетесь участником фонда. Создайте новый фонд или присоединитесь к существующему.")); err != nil {
			return
		}
		c.startMenu()
		return
	}

	var menuKeyboard = tgbotapi.NewInlineKeyboardMarkup( //меню для обычного пользователя
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Баланс", "баланс"),
			tgbotapi.NewInlineKeyboardButtonData("Оплатить", "1"), // реализовать
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Покинуть фонд", "3"), // реализовать
		),
	)

	msg := tgbotapi.NewMessage(c.chatId, "Приветствую! Выберите один из вариантов")

	ok, err = db.IsAdmin(c.chatId)
	if err != nil {
		log.Println(time.Now(), c.chatId, err)
		c.sendError()
		return
	}

	if ok { // если админ, то дополнить меню
		menuKeyboard.InlineKeyboard = append(menuKeyboard.InlineKeyboard,
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Новый сбор", "новый сбор"),
				tgbotapi.NewInlineKeyboardButtonData("Новое списание", "новое списание")),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Участники", "участники"),
				tgbotapi.NewInlineKeyboardButtonData("Статистика", "2"))) // реализовать
	}

	msg.ReplyMarkup = &menuKeyboard
	_ = c.send(msg)
}

// confirmationCreationNewFund проверяет состоит ли пользователь в другом фонде, если не состоит, то запрашивает подтверждение операции
func (c *Chat) confirmationCreationNewFund() {
	msg := tgbotapi.NewMessage(c.chatId, "")

	ok, err := db.IsMember(c.chatId)
	if err != nil {
		log.Println(time.Now(), c.chatId, err)
		c.sendError()
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
	_ = c.send(msg)
}

// creatingNewFund создает новый фонд
func (c *Chat) creatingNewFund() {
	sum, err := c.getFloatFromUser("Введите начальную сумму фонда")
	switch {
	case errors.Is(err, AttemptsExceeded):
		_ = c.send(tgbotapi.NewMessage(c.chatId, "Превышено число попыток ввода"))
		return
	case err != nil:
		log.Println(time.Now(), c.chatId, err)
		c.sendError()
		return
	}

	tag, err := newTag()
	if err != nil {
		log.Println(time.Now(), c.chatId, err)
		c.sendError()
	}

	name, err := c.getName()
	if err != nil {
		log.Println(time.Now(), c.chatId, err)
		c.sendError()
		return
	}

	err = db.CreateFund(tag, sum)
	if err != nil {
		log.Println(time.Now(), c.chatId, err)
		c.sendError()
		return
	}

	err = db.AddMember(tag, c.chatId, true, c.username, name)
	if err != nil {
		err = db.DeleteFund(tag)
		log.Println(time.Now(), c.chatId, err)
		c.sendError()
		return
	}

	err = c.send(tgbotapi.NewMessage(c.chatId, fmt.Sprintf("Новый фонд создан успешно! Присоединиться к фонду можно, используя тег: %s \nВнимание! Не показывайте этот тег посторонним людям.", tag)))
	if err != nil {
		err = db.DeleteFund(tag)
		log.Println(time.Now(), c.chatId, err)
		return
	}

}

func (c *Chat) showBalance() {
	tag, err := db.GetTag(c.chatId)
	if err != nil {
		log.Println(time.Now(), c.chatId, err)
		c.sendError()
		return
	}
	balance, err := db.ShowBalance(tag)
	if err != nil {
		log.Println(time.Now(), c.chatId, err)
		c.sendError()
		return
	}

	_ = c.send(tgbotapi.NewMessage(c.chatId, fmt.Sprintf("Текущий баланс: %.2f руб", balance)))
}

//
//func (r *response) downloadAttachment(fileId string) (fileName string, err error) {
//	_, err = r.bot.GetFile(tgbotapi.FileConfig{FileID: fileId})
//	if err != nil {
//		return
//	}
//
//	pathFile, err := r.bot.GetFileDirectURL(fileId)
//	if err != nil {
//		return
//	}
//
//	resp, err := http.Get(pathFile)
//	defer resp.Body.Close()
//	if err != nil {
//		return
//	}
//
//	fileName = strconv.FormatInt(r.chatId, 10) + "_" + path.Base(pathFile)
//	ok, err := ftp.StoreFile(fileName, resp.Body)
//	if err != nil {
//		fmt.Println(err)
//	}
//	fmt.Print(ok)
//
//	return
//}
//
//func (r *response) createDebitingFunds() {
//	sum, err := r.getFloatFromUser("Введите сумму списания.")
//	if err != nil {
//		return
//	}
//
//	msg := tgbotapi.NewMessage(r.chatId, "Укажите причину списания")
//	if _, err = r.bot.Send(msg); err != nil {
//		return
//	}
//
//	answer, err := r.waitingResponse("text")
//	if err != nil {
//		return
//	}
//	purpose := answer.Text
//
//	tag, err := db.GetTag(r.chatId)
//	if err != nil {
//		return
//	}
//
//	msg.Text = "Прикрепите чек файлом"
//	if _, err = r.bot.Send(msg); err != nil {
//		return
//	}
//	////////////////////////////////////////////ожидание чека файлом или картинкой
//	answer, err = r.waitingResponse("attachment")
//	if err != nil {
//		return
//	}
//
//	var file string
//	if answer.Photo != nil {
//		file = answer.Photo[len(answer.Photo)-1].FileID
//
//	} else {
//		file = answer.Document.FileID
//	}
//	fileName, err := r.downloadAttachment(file)
//	if err != nil {
//		return
//	}
//
//	///////////////////////////Создание транзакции////////////////////////////////////////////////
//	ok, err := db.CreateDebitingFunds(r.chatId, tag, sum, fmt.Sprintf("Инициатор: %s", r.username), purpose, fileName)
//	if err != nil || !ok {
//		r.notificationAboutError("Произошла ошибка. Попробуйте еще раз.")
//		return
//	}
//	///////////////////////////////////////////////////////////////////////////
//	msg.Text = "Списание проведено успешно."
//	_, _ = r.bot.Send(msg)
//
//}
//
//func (r *response) changeStatusOfTransaction(idTransaction int, status string) {
//	err := db.ChangeStatusTransaction(idTransaction, status)
//	if err != nil {
//		r.notificationAboutError("Произошла ошибка. Попробуйте еще раз.")
//		return
//	}
//	msg := tgbotapi.NewMessage(r.chatId, fmt.Sprintf("Статус оплаты: %s", status))
//	_, _ = r.bot.Send(msg)
//
//	r.paymentChangeStatusNotification(idTransaction)
//}
//
//func (r *response) paymentChangeStatusNotification(idTransaction int) {
//	status, _, _, memberId, _, err := db.InfoAboutTransaction(idTransaction)
//	if err != nil {
//		r.notificationAboutError("Произошла ошибка. Попробуйте еще раз.")
//		return
//	}
//	msg := tgbotapi.NewMessage(memberId, fmt.Sprintf("Статус оплаты изменен на: %s", status))
//	_, _ = r.bot.Send(msg)
//}
//
//func (r *response) payment(cashCollectionId int) {
//	target, _, err := db.InfoAboutCashCollection(cashCollectionId)
//	if err != nil {
//		r.notificationAboutError("Произошла ошибка. Попробуйте еще раз.")
//		return
//	}
//
//	sum, err := r.getFloatFromUser("Введите сумму пополнения.")
//	if err != nil {
//		r.notificationAboutError("Произошла ошибка. Попробуйте еще раз.")
//		return
//	}
//
//	if sum < target {
//		_, _ = r.bot.Send(tgbotapi.NewMessage(r.chatId, "Вы не можете оплатить сумму меньше необходимой."))
//		return
//	}
//
//	idTransaction, err := db.InsertInTransactions(cashCollectionId, sum, "пополнение", "ожидание", "", r.chatId)
//	if err != nil {
//		r.notificationAboutError("Произошла ошибка. Попробуйте еще раз.")
//		return
//	}
//
//	msg := tgbotapi.NewMessage(r.chatId, "Ваша оплата добавлена в очередь на подтверждение")
//	_, _ = r.bot.Send(msg)
//	r.paymentNotification(idTransaction)
//}
//
//func (r *response) paymentNotification(idTransaction int) { //доделать
//	tag, err := db.GetTag(r.chatId)
//	if err != nil {
//		r.notificationAboutError("Произошла ошибка. Попробуйте еще раз.")
//		return
//	}
//	adminId, err := db.GetAdminFund(tag)
//	if err != nil {
//		r.notificationAboutError("Произошла ошибка. Попробуйте еще раз.")
//		return
//	}
//
//	var okKeyboard = tgbotapi.NewInlineKeyboardMarkup(
//		tgbotapi.NewInlineKeyboardRow(
//			tgbotapi.NewInlineKeyboardButtonData("Подтвердить", fmt.Sprintf("подтвердить %d", idTransaction)),
//			tgbotapi.NewInlineKeyboardButtonData("Отказ", fmt.Sprintf("отказ %d", idTransaction)),
//			tgbotapi.NewInlineKeyboardButtonData("Ожидание", fmt.Sprintf("ожидание %d", idTransaction)),
//		),
//	)
//
//	_, _, _, memberId, sum, err := db.InfoAboutTransaction(idTransaction)
//
//	_, _, name, err := db.GetInfoAboutMember(memberId)
//
//	msg := tgbotapi.NewMessage(adminId, fmt.Sprintf("Подтвердите зачисление средств на счет фонда.\nСумма: %.2f\nОтправитель: %s", sum, name))
//	msg.ReplyMarkup = &okKeyboard
//	_, _ = r.bot.Send(msg)
//
//}
//

//func (r *response) createCashCollection() {
//	sum, err := r.getFloatFromUser("Введите сумму сбора с одного участника.")
//	if err != nil {
//		r.notificationAboutError("Произошла ошибка. Попробуйте еще раз.")
//		return
//	}
//
//	msg := tgbotapi.NewMessage(r.chatId, "Укажите назначение сбора")
//	if _, err = r.bot.Send(msg); err != nil {
//		return
//	}
//
//	answer, err := r.waitingResponse("text")
//	if err != nil {
//		r.notificationAboutError("Произошла ошибка. Попробуйте еще раз.")
//		return
//	}
//	purpose := answer.Text
//
//	tag, err := db.GetTag(r.chatId)
//	if err != nil {
//		r.notificationAboutError("Произошла ошибка. Попробуйте еще раз.")
//		return
//	}
//
//	id, err := db.CreateCashCollection(tag, sum, "открыт", fmt.Sprintf("Инициатор: %s", r.username), purpose, "")
//	if err != nil {
//		r.notificationAboutError("Произошла ошибка. Попробуйте еще раз.")
//		return
//	}
//	msg.Text = "Сбор создан. Сообщение о сборе будет отправлено всем участникам."
//	_, _ = r.bot.Send(msg)
//
//	r.collectionNotification(id, tag)
//}
//
//func (r *response) collectionNotification(idCollection int, tagFund string) {
//	members, err := db.SelectMembers(tagFund)
//	if err != nil {
//		r.notificationAboutError("Произошла ошибка. Попробуйте еще раз.")
//		return
//	}
//	sum, purpose, err := db.InfoAboutCashCollection(idCollection)
//	if err != nil {
//		r.notificationAboutError("Произошла ошибка. Попробуйте еще раз.")
//		return
//	}
//
//	for _, value := range members {
//		var paymentKeyboard = tgbotapi.NewInlineKeyboardMarkup(
//			tgbotapi.NewInlineKeyboardRow(
//				tgbotapi.NewInlineKeyboardButtonData("Оплатить", fmt.Sprintf("оплатить %d", idCollection)),
//			),
//		)
//		msg := tgbotapi.NewMessage(value, fmt.Sprintf("Иницирован новый сбор.\nСумма к оплате: %.2f\nНазначение: %s", sum, purpose))
//		msg.ReplyMarkup = &paymentKeyboard
//		_, _ = r.bot.Send(msg)
//	}
//}
//
//func (r *response) join() {
//	msg := tgbotapi.NewMessage(r.chatId, "")
//	ok, err := db.IsMember(r.chatId)
//	if err != nil {
//		r.notificationAboutError("Произошла ошибка. Попробуйте еще раз.")
//		return
//	}
//	if ok {
//		msg.Text = "Вы уже являетесь участником фонда"
//		_, _ = r.bot.Send(msg)
//		return
//	}
//	msg.Text = "Введите тег фонда. Если у вас нет тега, запросите его у администратора фонда."
//	if _, err = r.bot.Send(msg); err != nil {
//		return
//	}
//	answer, err := r.waitingResponse("text")
//	if err != nil {
//		r.notificationAboutError("Произошла ошибка. Попробуйте еще раз.")
//	}
//	tag := answer.Text
//
//	ok, err = db.ExistsFund(tag)
//	if err != nil {
//		r.notificationAboutError("Произошла ошибка. Попробуйте еще раз.")
//		return
//	}
//	if !ok {
//		msg.Text = "Фонд с таким тегом не найден."
//	} else {
//
//		name, err := r.getName()
//		if err != nil {
//			r.notificationAboutError("Произошла ошибка. Попробуйте еще раз.")
//			return
//		} else {
//			err = db.AddMember(tag, r.chatId, false, r.username, name)
//			if err != nil {
//				r.notificationAboutError("Произошла ошибка. Попробуйте еще раз.")
//				return
//			}
//			msg.Text = "Вы успешно присоединились к фонду."
//		}
//	}
//
//	_, _ = r.bot.Send(msg)
//}
//
//
//
//func (r *response) notificationAboutError(message string) {
//	if message == "" {
//		message = "Произошла ошибка. Попробуйте позже"
//	}
//
//	msg := tgbotapi.NewMessage(r.chatId, message)
//	_, _ = r.bot.Send(msg)
//	return
//}
//
//func (r *response) getMembers() {
//	tag, err := db.GetTag(r.chatId)
//	if err != nil {
//		r.notificationAboutError("")
//		return
//	}
//
//	id_members, err := db.SelectMembers(tag)
//	if err != nil {
//		r.notificationAboutError("")
//		return
//	}
//
//	message := "Список участников:\n"
//
//	for i, member := range id_members {
//		is_admin, login, name, err := db.GetInfoAboutMember(member)
//		if err != nil {
//			r.notificationAboutError("")
//			return
//		}
//		admin := ""
//		if is_admin {
//			admin = "Администратор"
//		}
//		message = message + fmt.Sprintf("%d. %s (@%s) %s\n", i+1, name, login, admin)
//	}
//
//	msg := tgbotapi.NewMessage(r.chatId, message)
//	_, _ = r.bot.Send(msg)
//}

// getFloatFromUser получить вещественное число от пользователя
func (c *Chat) getFloatFromUser(message string) (float64, error) {
	var sum float64
	if err := c.send(tgbotapi.NewMessage(c.chatId, message)); err != nil {
		return sum, err
	}

	for i := 0; i < 3; i++ {
		answer, err := c.getResponse("text")
		if err != nil {
			return sum, err
		}

		sum, err = strconv.ParseFloat(answer.Text, 64)
		if err != nil {
			msg := tgbotapi.NewMessage(c.chatId, "Неверный ввод. Повторите попытку")
			if i == 2 {
				msg.Text = ""
			}
			if err = c.send(msg); err != nil {
				return sum, err
			}
			continue
		}
		return sum, nil
	}
	return sum, AttemptsExceeded
}

func (c *Chat) getName() (string, error) {
	err := c.send(tgbotapi.NewMessage(c.chatId, "Представьтесь, пожалуйста. Введите ФИО"))
	if err != nil {
		return "", err
	}

	answer, err := c.getResponse("text")
	if err != nil {
		return "", err
	}
	return answer.Text, nil
}

// getResponse получить ответ от пользователя. typeOfResponse может быть attachment или text
func (c *Chat) getResponse(typeOfResponse string) (*tgbotapi.Message, error) {
	c.Service.WaitingResponse(c.chatId)
	defer c.Service.StopWaiting(c.chatId)

	var typeOfMessage string
	var answer *tgbotapi.Message

	for i := 0; i < 3; i++ {
		userChan, _ := c.GetUserChan(c.chatId)

		answer = <-userChan

		if answer.Photo != nil || answer.Document != nil {
			typeOfMessage = "attachment"
		} else {
			typeOfMessage = "text"
		}

		if typeOfResponse != typeOfMessage {
			if i < 2 {
				if err := c.send(tgbotapi.NewMessage(c.chatId, fmt.Sprintf("Вы ввели что-то не то. Количество доступных попыток: %d", 2-i))); err != nil {
					return nil, err
				}
			}
			continue
		}
		return answer, nil
	}

	return answer, AttemptsExceeded
}

func (c *Chat) sendError() {
	_ = c.send(tgbotapi.NewMessage(c.chatId, "Произошла ошибка. Повторите попытку позже"))
}

// newTag формирует новый тег. Выполняет проверку на существование. Если Тег уже существует формирует новый рекурсивно
func newTag() (string, error) {
	symbols := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	result := make([]byte, rand.Intn(5)+5)
	for i := range result {
		result[i] = symbols[rand.Intn(len(symbols))]
	}

	tag := string(result)

	ok, err := db.DoesTagExist(tag)
	if err != nil || !ok {
		return tag, err
	} else {
		return newTag()
	}
}
