package chat

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"math/rand"
	"project1/cmd/service"
	"project1/db"
	"regexp"
	"strconv"
	"strings"
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

// send 3 попытки на отправку, иначе удалить из списка ожидания и вернуть ошибку. Возвращает AttemptsExceeded
func (c *Chat) send(message tgbotapi.MessageConfig) error {

	for i := 0; i < 3; i++ {
		if _, err := c.Service.GetBot().Send(message); err == nil {
			return nil
		}
	}

	c.Service.DeleteFromWaitingList(c.chatId)

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
	case cmd == "confirmationCreateNewFund":
		c.confirmationCreateNewFund()
	case cmd == "join":
		c.join()
	case cmd == "createNewFund":
		c.createNewFund()
	case cmd == "showBalance":
		c.showBalance()
	case cmd == "test":
		_ = c.test()
	case cmd == "участники":
		c.getMembers()
	case cmd == "createCashCollection":
		c.createCashCollection()
	case cmd == "createDebitingFunds":
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
		_ = c.send(tgbotapi.NewMessage(c.chatId, "Я не знаю такую команду"))
	}

}

func (c *Chat) startMenu() {
	var startKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Создать фонд", "confirmationCreateNewFund"),
			tgbotapi.NewInlineKeyboardButtonData("Присоединиться", "join"),
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
		c.writeToLog("showMenu/isMember", err)
		c.sendAnyError()
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
			tgbotapi.NewInlineKeyboardButtonData("Баланс", "showBalance"),
			tgbotapi.NewInlineKeyboardButtonData("Оплатить", "1"), // реализовать
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Покинуть фонд", "3"), // реализовать и передвинуть
		),
	)

	msg := tgbotapi.NewMessage(c.chatId, "Приветствую! Выберите один из вариантов")

	ok, err = db.IsAdmin(c.chatId)
	if err != nil {
		c.writeToLog("showMenu/isAdmin", err)
		c.sendAnyError()
		return
	}

	if ok { // если админ, то дополнить меню
		menuKeyboard.InlineKeyboard = append(menuKeyboard.InlineKeyboard,
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Новый сбор", "createCashCollection"),
				tgbotapi.NewInlineKeyboardButtonData("Новое списание", "createDebitingFunds")),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Участники", "getMembers"),
				tgbotapi.NewInlineKeyboardButtonData("Статистика", "2"))) // реализовать
	}

	msg.ReplyMarkup = &menuKeyboard
	_ = c.send(msg)
}

// confirmationCreationNewFund проверяет состоит ли пользователь в другом фонде, если не состоит, то запрашивает подтверждение операции
func (c *Chat) confirmationCreateNewFund() {
	ok, err := db.IsMember(c.chatId)
	if err != nil {
		c.writeToLog("confirmationCreateNewFund/isMember", err)
		c.sendAnyError()
		return
	}
	if ok {
		_ = c.send(tgbotapi.NewMessage(c.chatId, "Вы уже являетесь участником фонда"))
		return
	}

	var msg tgbotapi.MessageConfig

	msg.Text = "Вы уверены, что хотите создать новый фонд?"
	var numericKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Да", "createNewFund"),
			tgbotapi.NewInlineKeyboardButtonData("Нет", "start"),
		),
	)
	msg.ReplyMarkup = &numericKeyboard

	_ = c.send(msg)
}

// creatingNewFund создает новый фонд
func (c *Chat) createNewFund() {
	sum, err := c.getFloatFromUser("Введите начальную сумму фонда")
	if err != nil {
		c.sendAttemptsExceededError()
		return
	}

	tag, err := newTag()
	if err != nil {
		c.writeToLog("createNewFund/newTag", err)
		c.sendAnyError()
	}

	name, err := c.getName()
	if err != nil {
		c.sendAttemptsExceededError()
		return
	}

	if err = db.CreateFund(tag, sum); err != nil {
		c.writeToLog("createNewFund", err)
		c.sendAnyError()
		return
	}

	if err = db.AddMember(tag, c.chatId, true, c.username, name); err != nil {
		err = db.DeleteFund(tag)
		c.writeToLog("createNewFund/addMember", err)
		c.sendAnyError()
		return
	}

	if err = c.send(tgbotapi.NewMessage(c.chatId, fmt.Sprintf("Новый фонд создан успешно! Присоединиться к фонду можно, используя тег: %s \nВнимание! Не показывайте этот тег посторонним людям.", tag))); err != nil {
		if err = db.DeleteFund(tag); err != nil {
			c.writeToLog("createNewFund/deleteFund", err)
		}
		return
	}

}

func (c *Chat) showBalance() {
	tag, err := db.GetTag(c.chatId)
	if err != nil {
		c.writeToLog("showBalance/getTag", err)
		c.sendAnyError()
		return
	}
	balance, err := db.ShowBalance(tag)
	if err != nil {
		c.writeToLog("showBalance", err)
		c.sendAnyError()
		return
	}

	_ = c.send(tgbotapi.NewMessage(c.chatId, fmt.Sprintf("Текущий баланс: %.2f руб", balance)))
}

func (c *Chat) join() {
	ok, err := db.IsMember(c.chatId)
	if err != nil {
		c.writeToLog("join/isMember", err)
		c.sendAnyError()
		return
	}
	if ok {
		_ = c.send(tgbotapi.NewMessage(c.chatId, "Вы уже являетесь участником фонда"))
		return
	}

	if err = c.send(tgbotapi.NewMessage(c.chatId, "Введите тег фонда. Если у вас нет тега, запросите его у администратора фонда")); err != nil {
		return
	}

	response, err := c.getResponse("text")
	if err != nil {
		c.sendAttemptsExceededError()
		return
	}

	tag := response.Text

	ok, err = db.DoesTagExist(tag)
	if err != nil {
		c.writeToLog("join/doesTagExists", err)
		c.sendAnyError()
		return
	}
	if !ok {
		_ = c.send(tgbotapi.NewMessage(c.chatId, "Фонд с таким тегом не найден"))
		return
	}

	name, err := c.getName()
	if err != nil {
		c.sendAttemptsExceededError()
		return
	}

	err = db.AddMember(tag, c.chatId, false, c.username, name)
	if err != nil {
		c.writeToLog("join/addMember", err)
		c.sendAnyError()
		return
	}

	_ = c.send(tgbotapi.NewMessage(c.chatId, "Вы успешно присоединились к фонду"))
}

func (c *Chat) getMembers() {
	tag, err := db.GetTag(c.chatId)
	if err != nil {
		c.writeToLog("getMembers/getTag", err)
		c.sendAnyError()
		return
	}

	idMembers, err := db.GetMembers(tag)
	if err != nil {
		c.writeToLog("getMembers", err)
		c.sendAnyError()
		return
	}

	var strBuilder strings.Builder

	strBuilder.WriteString("Список участников:\n")

	for i, id := range idMembers {
		isAdmin, login, name, err := db.GetInfoAboutMember(id)
		if err != nil {
			c.writeToLog("getMembers/getInfoAboutMember", err)
			c.sendAnyError()
			return
		}
		admin := ""
		if isAdmin {
			admin = "Администратор"
		}
		strBuilder.WriteString(fmt.Sprintf("%d. %s (@%s) %s\n", i+1, name, login, admin))

	}

	_ = c.send(tgbotapi.NewMessage(c.chatId, strBuilder.String()))

}

func (c *Chat) createCashCollection() {
	sum, err := c.getFloatFromUser("Введите сумму сбора с одного участника.")
	if err != nil {
		c.sendAttemptsExceededError()
		return
	}

	if err = c.send(tgbotapi.NewMessage(c.chatId, "Укажите назначение сбора")); err != nil {
		return
	}

	answer, err := c.getResponse("text")
	if err != nil {
		c.sendAttemptsExceededError()
		return
	}

	tag, err := db.GetTag(c.chatId)
	if err != nil {
		c.sendAnyError()
		return
	}

	id, err := db.CreateCashCollection(tag, sum, "открыт", fmt.Sprintf("Инициатор: %s", c.username), answer.Text, "")
	if err != nil {
		c.sendAnyError()
		return
	}

	_ = c.send(tgbotapi.NewMessage(c.chatId, "Сбор создан. Сообщение о сборе будет отправлено всем участникам"))

	c.collectionNotification(id, tag)
}

func (c *Chat) collectionNotification(idCollection int, tagFund string) {
	members, err := db.GetMembers(tagFund)
	if err != nil {
		c.sendAnyError()
		return
	}
	sum, purpose, err := db.InfoAboutCashCollection(idCollection)
	if err != nil {
		c.sendAnyError()
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
		_ = c.send(msg)
	}
}

//
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

// getFloatFromUser получить вещественное число от пользователя. Возвращает AttemptsExceeded
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

// getName получить имя пользователя. Возвращает AttemptsExceeded
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

// getResponse получить ответ от пользователя. typeOfResponse может быть attachment или text. Возвращает AttemptsExceeded
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

func (c *Chat) sendAnyError() {
	if err := c.send(tgbotapi.NewMessage(c.chatId, "Произошла ошибка. Повторите попытку позже")); err != nil {
		c.writeToLog("sendError", err)
	}
}

func (c *Chat) sendAttemptsExceededError() {
	if err := c.send(tgbotapi.NewMessage(c.chatId, "Превышено число попыток ввода")); err != nil {
		c.writeToLog("sendAttemptsExceededError", err)
	}
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

func (c *Chat) writeToLog(location string, err error) {
	log.Println(time.Now(), c.chatId, fmt.Sprintf("%s: ", location), err)
}
