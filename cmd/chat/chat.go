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
	var paymentPat = regexp.MustCompile(`^payment\s\d*.`)
	var rejectionPat = regexp.MustCompile(`^reject\s\d*.`)
	var waitingPat = regexp.MustCompile(`^wait\s\d*.`)
	var acceptPat = regexp.MustCompile(`^accept\s\d*.`)

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
	case cmd == "getMembers":
		c.getMembers()
	case cmd == "createCashCollection":
		c.createCashCollection()
	case cmd == "createDebitingFunds":
		//c.createDebitingFunds()
	case paymentPat.MatchString(cmd): // оплата
		cashCollectionId, err := strconv.Atoi(strings.Split(cmd, " ")[1])
		if err != nil {
			c.sendAnyError()
			return
		}
		c.payment(cashCollectionId)
	case acceptPat.MatchString(cmd): // подтверждение оплаты
		idTransaction, err := strconv.Atoi(strings.Split(cmd, " ")[1])
		if err != nil {
			c.writeToLog("CommandSwitcher/acceptPat", err)
			c.sendAnyError()
			return
		}
		c.changeStatusOfTransaction(idTransaction, "подтвержден")
	case waitingPat.MatchString(cmd): // ожидание оплаты
		idTransaction, err := strconv.Atoi(strings.Split(cmd, " ")[1])
		if err != nil {
			c.writeToLog("CommandSwitcher/waitingPat", err)
			return
		}
		c.changeStatusOfTransaction(idTransaction, "ожидание")
	case rejectionPat.MatchString(cmd): // отказ оплаты
		idTransaction, err := strconv.Atoi(strings.Split(cmd, " ")[1])
		if err != nil {
			c.writeToLog("CommandSwitcher/rejectionPat", err)
			return
		}
		c.changeStatusOfTransaction(idTransaction, "отказ")
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

	msg := tgbotapi.NewMessage(c.chatId, "Вы уверены, что хотите создать новый фонд?")

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

	if err = db.AddMember(db.Member{
		ID:      c.chatId,
		Tag:     tag,
		IsAdmin: true,
		Login:   c.username,
		Name:    name,
	}); err != nil {
		c.writeToLog("createNewFund/AddMember", err)
		err = db.DeleteFund(tag)
		c.writeToLog("createNewFund/DeleteFund", err)
		c.sendAnyError()
		return
	}

	if err = c.send(tgbotapi.NewMessage(c.chatId, fmt.Sprintf("Новый фонд создан успешно! Присоединиться к фонду можно, используя тег: %s \nВнимание! Не показывайте этот тег посторонним людям.", tag))); err != nil {
		if err = db.DeleteFund(tag); err != nil {
			c.writeToLog("createNewFund/DeleteFund", err)
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

	if err = db.AddMember(db.Member{
		ID:      c.chatId,
		Tag:     tag,
		IsAdmin: false,
		Login:   c.username,
		Name:    name,
	}); err != nil {
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

	members, err := db.GetMembers(tag)
	if err != nil {
		c.writeToLog("getMembers", err)
		c.sendAnyError()
		return
	}

	var strBuilder strings.Builder

	strBuilder.WriteString("Список участников:\n")

	for i, member := range members {
		admin := ""
		if member.IsAdmin {
			admin = "Администратор"
		}
		strBuilder.WriteString(fmt.Sprintf("%d. %s (@%s) %s\n", i+1, member.Name, member.Login, admin))

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
		c.writeToLog("createCashCollection/GetTag", err)
		c.sendAnyError()
		return
	}

	id, err := db.CreateCashCollection(db.CashCollection{
		Tag:        tag,
		Sum:        sum,
		Status:     "открыт",
		Comment:    fmt.Sprintf("Инициатор: %s", c.username),
		Purpose:    answer.Text,
		CreateDate: time.Now(),
	})
	if err != nil {
		c.writeToLog("createCashCollection/CreateCashCollection", err)
		c.sendAnyError()
		return
	}

	_ = c.send(tgbotapi.NewMessage(c.chatId, "Сбор создан. Сообщение о сборе будет отправлено всем участникам"))

	c.collectionNotification(id, tag)
}

func (c *Chat) collectionNotification(idCollection int, tagFund string) {
	members, err := db.GetMembers(tagFund)
	if err != nil {
		c.writeToLog("collectionNotification/GetMembers", err)
		c.sendAnyError()
		return
	}
	cc, err := db.InfoAboutCashCollection(idCollection)
	if err != nil {
		c.writeToLog("collectionNotification/InfoAboutCashCollection", err)
		c.sendAnyError()
		return
	}

	var paymentKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Оплатить", fmt.Sprintf("payment %d", idCollection)),
		),
	)

	for _, member := range members {
		msg := tgbotapi.NewMessage(member.ID, fmt.Sprintf("Иницирован новый сбор.\nСумма к оплате: %.2f\nНазначение: %s", cc.Sum, cc.Purpose))
		msg.ReplyMarkup = &paymentKeyboard
		_ = c.send(msg)
	}
}

func (c *Chat) payment(cashCollectionId int) {
	cc, err := db.InfoAboutCashCollection(cashCollectionId)
	if err != nil {
		c.writeToLog("payment/InfoAboutCashCollection", err)
		c.sendAnyError()
		return
	}

	sum, err := c.getFloatFromUser("Введите сумму пополнения")
	if err != nil {
		c.sendAttemptsExceededError()
		return
	}

	if sum < cc.Sum {
		_ = c.send(tgbotapi.NewMessage(c.chatId, "Вы не можете оплатить сумму меньше необходимой."))
		return
	}

	idTransaction, err := db.InsertInTransactions(db.Transaction{
		CashCollectionID: cashCollectionId,
		Sum:              sum,
		Type:             "пополнение",
		Status:           "ожидание",
		Receipt:          "",
		MemberID:         c.chatId,
		Date:             time.Now(),
	})
	if err != nil {
		c.writeToLog("payment/InsertInTransactions", err)
		c.sendAnyError()
		return
	}

	_ = c.send(tgbotapi.NewMessage(c.chatId, "Ваша оплата добавлена в очередь на подтверждение"))
	c.paymentNotification(idTransaction, sum)
}

// paymentNotification отправить запрос на подтверждение оплаты администратору
func (c *Chat) paymentNotification(idTransaction int, sum float64) { //доделать
	tag, err := db.GetTag(c.chatId)
	if err != nil {
		c.writeToLog("paymentNotification/GetTag", err)
		c.sendAnyError()
		return
	}
	adminId, err := db.GetAdminFund(tag)
	if err != nil {
		c.writeToLog("paymentNotification/GetAdminFund", err)
		c.sendAnyError()
		return
	}

	member, err := db.GetInfoAboutMember(c.chatId)
	if err != nil {
		c.writeToLog("paymentNotification/GetInfoAboutMember", err)
		c.sendAnyError()
		return
	}

	var okKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Подтвердить", fmt.Sprintf("accept %d", idTransaction)),
			tgbotapi.NewInlineKeyboardButtonData("Отказ", fmt.Sprintf("reject %d", idTransaction)),
			tgbotapi.NewInlineKeyboardButtonData("Ожидание", fmt.Sprintf("wait %d", idTransaction)),
		),
	)

	msg := tgbotapi.NewMessage(adminId, fmt.Sprintf("Подтвердите зачисление средств на счет фонда.\nСумма: %.2f\nОтправитель: %s", sum, member.Name))
	msg.ReplyMarkup = &okKeyboard
	_ = c.send(msg)

}

// changeStatusOfTransaction изменение статуса транзакции
func (c *Chat) changeStatusOfTransaction(idTransaction int, status string) {
	err := db.ChangeStatusTransaction(idTransaction, status)
	if err != nil {
		c.writeToLog("changeStatusOfTransaction", err)
		c.sendAnyError()
		return
	}

	_ = c.send(tgbotapi.NewMessage(c.chatId, fmt.Sprintf("Статус оплаты: %s", status)))

	t, err := db.InfoAboutTransaction(idTransaction)
	if err != nil {
		c.writeToLog("changeStatusOfTransaction/InfoAboutTransaction", err)
	}

	if err = db.UpdateStatusCashCollection(t.CashCollectionID); err != nil {
		fmt.Println(err)
		c.writeToLog("changeStatusOfTransaction/CheckDebtors", err)
	}

	c.paymentChangeStatusNotification(idTransaction)
}

func (c *Chat) paymentChangeStatusNotification(idTransaction int) {
	t, err := db.InfoAboutTransaction(idTransaction)
	if err != nil {
		c.writeToLog("paymentChangeStatusNotification", err)
		c.sendAnyError()
		return
	}

	_ = c.send(tgbotapi.NewMessage(t.MemberID, fmt.Sprintf("Статус оплаты изменен на: %s", t.Status)))
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
	log.Println(c.chatId, location, err)
}
