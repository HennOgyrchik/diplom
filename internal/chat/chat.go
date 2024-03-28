package chat

import (
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"math/rand"
	"net/http"
	"path"
	"project1/internal/db"
	"project1/internal/service"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	alphabet                 = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	typeOfResponseText       = "text"
	typeOfResponseAttachment = "attachment"
	layoutDate               = "02.01.2006"
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
		if err = c.Send(tgbotapi.NewMessage(c.chatId, "тестовая кнопка")); err == nil {
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

// Send 3 попытки на отправку, иначе удалить из списка ожидания и вернуть ошибку. Возвращает AttemptsExceeded
func (c *Chat) Send(message tgbotapi.MessageConfig) error {

	for i := 0; i < 3; i++ {
		if _, err := c.Service.GetBot().Send(message); err == nil {
			return nil
		}
	}

	c.Service.DeleteFromWaitingList(c.chatId)

	return AttemptsExceeded
}

func (c *Chat) CommandSwitcher(query string) bool {
	var paymentPat = regexp.MustCompile(`^payment\d*.`)
	var rejectionPat = regexp.MustCompile(`^reject\d*.`)
	var expectationPat = regexp.MustCompile(`^wait\d*.`)
	var acceptPat = regexp.MustCompile(`^accept\d*.`)
	var deletePat = regexp.MustCompile(`^deleteMemberYes\d*.`)
	var historyPat = regexp.MustCompile(`^history\d*.`)
	var setAdminPat = regexp.MustCompile(`^setAdminYes\d*.`)

	switch cmd := query; {
	case cmd == c.Commands.Start:
		go c.startMenu()
	case cmd == c.Commands.Menu:
		go c.showMenu()
	case cmd == c.Commands.ShowTag:
		go c.showTag()
	case cmd == c.Commands.CreateFund:
		go c.createFund()
	case cmd == c.Commands.SetAdmin:
		go c.setAdmin()
	case cmd == c.Commands.Join:
		go c.join()
	case cmd == c.Commands.AwaitingPayment:
		go c.awaitingPayment()
	case cmd == c.Commands.CreateFundYes:
		go c.CreateFundYes()
	case cmd == c.Commands.ShowBalance:
		go c.showBalance()
	case cmd == c.Commands.DeleteMember:
		go c.deleteMember()
	case cmd == c.Commands.GetMembers:
		go c.getMembers()
	case cmd == c.Commands.CreateCashCollection:
		go c.createCashCollection()
	case cmd == c.Commands.CreateDebitingFunds:
		go c.createDebitingFunds()
	case cmd == c.Commands.ShowListDebtors:
		go c.showListDebtors()
	case cmd == c.Commands.Leave:
		go c.leave()
	case cmd == c.Commands.LeaveYes:
		go c.leaveYes()
	case historyPat.MatchString(cmd): //история списаний
		go func() {
			id, err := strconv.Atoi(strings.ReplaceAll(cmd, c.Commands.History, ""))
			if err != nil {
				c.writeToLog("CommandSwitcher/historyPat", err)
				c.sendAnyError()
				return
			}
			c.showHistory(id)
		}()
	case setAdminPat.MatchString(cmd): //сменить администратора
		go func() {

			id, err := strconv.ParseInt(strings.ReplaceAll(cmd, c.Commands.SetAdminYes, ""), 10, 64)
			if err != nil {
				c.writeToLog("CommandSwitcher/setAdminPat", err)
				c.sendAnyError()
				return
			}
			c.setAdminYes(id)
		}()
	case deletePat.MatchString(cmd): //удалить пользователя
		go func() {
			id, err := strconv.ParseInt(strings.ReplaceAll(cmd, c.Commands.DeleteMemberYes, ""), 10, 64)
			if err != nil {
				c.writeToLog("CommandSwitcher/deletePat", err)
				c.sendAnyError()
				return
			}
			c.deleteMemberYes(id)
		}()

	case paymentPat.MatchString(cmd): // оплата
		go func() {
			cashCollectionId, err := strconv.Atoi(strings.ReplaceAll(cmd, c.Commands.Payment, ""))
			if err != nil {
				c.writeToLog("CommandSwitcher/paymentPat", err)
				c.sendAnyError()
				return
			}
			c.payment(cashCollectionId)
		}()
	case acceptPat.MatchString(cmd): // подтверждение оплаты
		go func() {
			idTransaction, err := strconv.Atoi(strings.ReplaceAll(cmd, c.Commands.PaymentAccept, ""))
			if err != nil {
				c.writeToLog("CommandSwitcher/acceptPat", err)
				c.sendAnyError()
				return
			}
			c.changeStatusOfTransaction(idTransaction, db.StatusPaymentConfirmation)
		}()
	case expectationPat.MatchString(cmd): // ожидание оплаты
		go func() {
			idTransaction, err := strconv.Atoi(strings.ReplaceAll(cmd, c.Commands.PaymentWait, ""))
			if err != nil {
				c.writeToLog("CommandSwitcher/expectationPat", err)
				return
			}
			c.changeStatusOfTransaction(idTransaction, db.StatusPaymentExpectation)
		}()
	case rejectionPat.MatchString(cmd): // отказ оплаты
		go func() {
			idTransaction, err := strconv.Atoi(strings.ReplaceAll(cmd, c.Commands.PaymentReject, ""))
			if err != nil {
				c.writeToLog("CommandSwitcher/rejectionPat", err)
				return
			}
			c.changeStatusOfTransaction(idTransaction, db.StatusPaymentRejection)
		}()
	default:
		return false
	}

	return true
}

func (c *Chat) startMenu() {
	var startKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(c.Buttons.CreateFound.Label, c.Buttons.CreateFound.Command),
			tgbotapi.NewInlineKeyboardButtonData(c.Buttons.Join.Label, c.Buttons.Join.Command),
		),
	)

	msg := tgbotapi.NewMessage(c.chatId, "Приветствую! Выберите один из вариантов")
	msg.ReplyMarkup = &startKeyboard

	_ = c.Send(msg)
}

func (c *Chat) showMenu() {
	ok, err := c.DB.IsMember(c.chatId)
	if err != nil {
		c.writeToLog("showMenu/isMember", err)
		c.sendAnyError()
		return
	}
	if !ok {
		if err = c.Send(tgbotapi.NewMessage(c.chatId, "Вы не являетесь участником фонда. Создайте новый фонд или присоединитесь к существующему")); err != nil {
			return
		}
		c.startMenu()
		return
	}

	var menuKeyboard = tgbotapi.NewInlineKeyboardMarkup( //меню для обычного пользователя
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(c.Buttons.ShowBalance.Label, c.Buttons.ShowBalance.Command),
			tgbotapi.NewInlineKeyboardButtonData(c.Buttons.AwaitingPayment.Label, c.Buttons.AwaitingPayment.Command),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(c.Buttons.History.Label, c.Buttons.History.Command+strconv.Itoa(0)),
			tgbotapi.NewInlineKeyboardButtonData(c.Buttons.Leave.Label, c.Buttons.Leave.Command),
		),
	)

	msg := tgbotapi.NewMessage(c.chatId, "Приветствую! Выберите один из вариантов")

	ok, err = c.DB.IsAdmin(c.chatId)
	if err != nil {
		c.writeToLog("showMenu/isAdmin", err)
		c.sendAnyError()
		return
	}

	if ok { // если админ, то дополнить меню
		menuKeyboard.InlineKeyboard = append(menuKeyboard.InlineKeyboard,
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(c.Buttons.CreateCashCollection.Label, c.Buttons.CreateCashCollection.Command),
				tgbotapi.NewInlineKeyboardButtonData(c.Buttons.CreateDebitingFunds.Label, c.Buttons.CreateDebitingFunds.Command)),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(c.Buttons.Members.Label, c.Buttons.Members.Command),
				tgbotapi.NewInlineKeyboardButtonData(c.Buttons.DebtorList.Label, c.Buttons.DebtorList.Command)),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(c.Buttons.ShowTag.Label, c.Buttons.ShowTag.Command),
				tgbotapi.NewInlineKeyboardButtonData(c.Buttons.SetAdmin.Label, c.Buttons.SetAdmin.Command)),
		)
	}

	msg.ReplyMarkup = &menuKeyboard
	_ = c.Send(msg)
}

// createFund проверяет состоит ли пользователь в другом фонде, если не состоит, то запрашивает подтверждение операции
func (c *Chat) createFund() {
	ok, err := c.DB.IsMember(c.chatId)
	if err != nil {
		c.writeToLog("createFund/isMember", err)
		c.sendAnyError()
		return
	}
	if ok {
		_ = c.Send(tgbotapi.NewMessage(c.chatId, "Вы уже являетесь участником фонда"))
		return
	}

	msg := tgbotapi.NewMessage(c.chatId, "Вы уверены, что хотите создать новый фонд?")

	var numericKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(c.Buttons.CreateFoundYes.Label, c.Buttons.CreateFoundYes.Command),
			tgbotapi.NewInlineKeyboardButtonData(c.Buttons.CreateFoundNo.Label, c.Buttons.CreateFoundNo.Command),
		),
	)
	msg.ReplyMarkup = &numericKeyboard

	_ = c.Send(msg)
}

// CreateFundYes создает новый фонд
func (c *Chat) CreateFundYes() {
	sum, err := c.getFloatFromUser("Введите начальную сумму фонда")
	if err != nil {
		if !errors.Is(err, Close) {
			c.sendAttemptsExceededError()
		}
		return
	}

	tag, err := c.newTag()
	if err != nil {
		c.writeToLog("CreateFundYes/newTag", err)
		c.sendAnyError()
	}

	name, err := c.getName()
	if err != nil {
		if !errors.Is(err, Close) {
			c.sendAttemptsExceededError()
		}
		return
	}

	if err = c.DB.CreateFund(tag, sum); err != nil {
		c.writeToLog("CreateFundYes", err)
		c.sendAnyError()
		return
	}

	if err = c.DB.AddMember(db.Member{
		ID:      c.chatId,
		Tag:     tag,
		IsAdmin: true,
		Login:   c.username,
		Name:    name,
	}); err != nil {
		c.writeToLog("CreateFundYes/AddMember", err)
		err = c.DB.DeleteFund(tag)
		c.writeToLog("CreateFundYes/DeleteFund", err)
		c.sendAnyError()
		return
	}

	if err = c.Send(tgbotapi.NewMessage(c.chatId, fmt.Sprintf("Новый фонд создан успешно! Присоединиться к фонду можно, используя тег: %s \nВнимание! Не показывайте этот тег посторонним людям.", tag))); err != nil {
		if err = c.DB.DeleteFund(tag); err != nil {
			c.writeToLog("CreateFundYes/DeleteFund", err)
		}
		return
	}

}

func (c *Chat) showBalance() {
	tag, err := c.DB.GetTag(c.chatId)
	if err != nil {
		c.writeToLog("showBalance/getTag", err)
		c.sendAnyError()
		return
	}
	balance, err := c.DB.ShowBalance(tag)
	if err != nil {
		c.writeToLog("showBalance", err)
		c.sendAnyError()
		return
	}

	_ = c.Send(tgbotapi.NewMessage(c.chatId, fmt.Sprintf("Текущий баланс: %.2f руб", balance)))
}

func (c *Chat) join() {
	ok, err := c.DB.IsMember(c.chatId)
	if err != nil {
		c.writeToLog("join/isMember", err)
		c.sendAnyError()
		return
	}
	if ok {
		_ = c.Send(tgbotapi.NewMessage(c.chatId, "Вы уже являетесь участником фонда"))
		return
	}

	if err = c.Send(tgbotapi.NewMessage(c.chatId, "Введите тег фонда. Если у вас нет тега, запросите его у администратора фонда")); err != nil {
		return
	}

	response, err := c.getResponse(typeOfResponseText)
	if err != nil {
		if !errors.Is(err, Close) {
			c.sendAttemptsExceededError()
		}
		return
	}

	tag := response.Text

	ok, err = c.DB.DoesTagExist(tag)
	if err != nil {
		c.writeToLog("join/doesTagExists", err)
		c.sendAnyError()
		return
	}
	if !ok {
		_ = c.Send(tgbotapi.NewMessage(c.chatId, "Фонд с таким тегом не найден"))
		return
	}

	name, err := c.getName()
	if err != nil {
		if !errors.Is(err, Close) {
			c.sendAttemptsExceededError()
		}
		return
	}

	if err = c.DB.AddMember(db.Member{
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

	_ = c.Send(tgbotapi.NewMessage(c.chatId, "Вы успешно присоединились к фонду"))
}

func (c *Chat) formatListMembers(members []db.Member) tgbotapi.MessageConfig {
	var strBuilder strings.Builder

	strBuilder.WriteString("Список участников:\n")

	for i, member := range members {
		admin := ""
		if member.IsAdmin {
			admin = "Администратор"
		}
		strBuilder.WriteString(fmt.Sprintf("%d. %s (@%s) %s\n", i+1, member.Name, member.Login, admin))

	}

	return tgbotapi.NewMessage(c.chatId, strBuilder.String())
}

func (c *Chat) getListMembers() ([]db.Member, error) {
	tag, err := c.DB.GetTag(c.chatId)
	if err != nil {
		return []db.Member{}, err
	}

	return c.DB.GetMembers(tag)
}

func (c *Chat) createCashCollection() {
	sum, err := c.getFloatFromUser("Введите сумму сбора с одного участника")
	if err != nil {
		if !errors.Is(err, Close) {
			c.sendAttemptsExceededError()
		}
		return
	}

	if err = c.Send(tgbotapi.NewMessage(c.chatId, "Укажите назначение сбора")); err != nil {
		return
	}

	answer, err := c.getResponse(typeOfResponseText)
	if err != nil {
		if !errors.Is(err, Close) {
			c.sendAttemptsExceededError()
		}
		return
	}

	tag, err := c.DB.GetTag(c.chatId)
	if err != nil {
		c.writeToLog("createCashCollection/GetTag", err)
		c.sendAnyError()
		return
	}

	id, err := c.DB.CreateCashCollection(db.CashCollection{
		Tag:        tag,
		Sum:        sum,
		Status:     db.StatusCashCollectionOpen,
		Comment:    fmt.Sprintf("Инициатор: %s", c.username),
		Purpose:    answer.Text,
		CreateDate: time.Now(),
	})
	if err != nil {
		c.writeToLog("createCashCollection/CreateCashCollection", err)
		c.sendAnyError()
		return
	}

	_ = c.Send(tgbotapi.NewMessage(c.chatId, "Сбор создан. Сообщение о сборе будет отправлено всем участникам"))

	c.collectionNotification(id, tag)
}

func (c *Chat) collectionNotification(idCollection int, tagFund string) {
	members, err := c.DB.GetMembers(tagFund)
	if err != nil {
		c.writeToLog("collectionNotification/GetMembers", err)
		c.sendAnyError()
		return
	}
	cc, err := c.DB.InfoAboutCashCollection(idCollection)
	if err != nil {
		c.writeToLog("collectionNotification/InfoAboutCashCollection", err)
		c.sendAnyError()
		return
	}

	var paymentKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(c.Buttons.Payment.Label, c.Buttons.Payment.Command+strconv.Itoa(idCollection)),
		),
	)

	for _, member := range members {
		msg := tgbotapi.NewMessage(member.ID, fmt.Sprintf("Иницирован новый сбор.\nСумма к оплате: %.2f\nНазначение: %s", cc.Sum, cc.Purpose))
		msg.ReplyMarkup = &paymentKeyboard
		_ = c.Send(msg)
	}
}

func (c *Chat) payment(cashCollectionId int) {
	cc, err := c.DB.InfoAboutCashCollection(cashCollectionId)
	if err != nil {
		c.writeToLog("payment/InfoAboutCashCollection", err)
		c.sendAnyError()
		return
	}

	sum, err := c.getFloatFromUser("Введите сумму пополнения")
	if err != nil {
		if !errors.Is(err, Close) {
			c.sendAttemptsExceededError()
		}
		return
	}

	if sum < cc.Sum {
		_ = c.Send(tgbotapi.NewMessage(c.chatId, "Вы не можете оплатить сумму меньше необходимой."))
		return
	}

	idTransaction, err := c.DB.InsertInTransactions(db.Transaction{
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

	_ = c.Send(tgbotapi.NewMessage(c.chatId, "Ваша оплата добавлена в очередь на подтверждение"))
	c.paymentNotification(idTransaction, sum)
}

// paymentNotification отправить запрос на подтверждение оплаты администратору
func (c *Chat) paymentNotification(idTransaction int, sum float64) { //доделать
	tag, err := c.DB.GetTag(c.chatId)
	if err != nil {
		c.writeToLog("paymentNotification/GetTag", err)
		c.sendAnyError()
		return
	}
	adminId, err := c.DB.GetAdminFund(tag)
	if err != nil {
		c.writeToLog("paymentNotification/GetAdminFund", err)
		c.sendAnyError()
		return
	}

	member, err := c.DB.GetInfoAboutMember(c.chatId)
	if err != nil {
		c.writeToLog("paymentNotification/GetInfoAboutMember", err)
		c.sendAnyError()
		return
	}

	var okKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(c.Buttons.PaymentConfirmation.Label, c.Buttons.PaymentConfirmation.Command+strconv.Itoa(idTransaction)),
			tgbotapi.NewInlineKeyboardButtonData(c.Buttons.PaymentRefusal.Label, c.Buttons.PaymentRefusal.Command+strconv.Itoa(idTransaction)),
			tgbotapi.NewInlineKeyboardButtonData(c.Buttons.PaymentExpected.Label, c.Buttons.PaymentExpected.Command+strconv.Itoa(idTransaction)),
		),
	)

	msg := tgbotapi.NewMessage(adminId, fmt.Sprintf("Подтвердите зачисление средств на счет фонда.\nСумма: %.2f\nОтправитель: %s", sum, member.Name))
	msg.ReplyMarkup = &okKeyboard
	_ = c.Send(msg)

}

// changeStatusOfTransaction изменение статуса транзакции
func (c *Chat) changeStatusOfTransaction(idTransaction int, status string) {
	err := c.DB.ChangeStatusTransaction(idTransaction, status)
	if err != nil {
		c.writeToLog("changeStatusOfTransaction", err)
		c.sendAnyError()
		return
	}

	_ = c.Send(tgbotapi.NewMessage(c.chatId, fmt.Sprintf("Статус оплаты: %s", status)))

	t, err := c.DB.InfoAboutTransaction(idTransaction)
	if err != nil {
		c.writeToLog("changeStatusOfTransaction/InfoAboutTransaction", err)
	}

	if err = c.DB.UpdateStatusCashCollection(t.CashCollectionID); err != nil {
		c.writeToLog("changeStatusOfTransaction/CheckDebtors", err)
	}

	c.paymentChangeStatusNotification(idTransaction)
}

func (c *Chat) paymentChangeStatusNotification(idTransaction int) {
	t, err := c.DB.InfoAboutTransaction(idTransaction)
	if err != nil {
		c.writeToLog("paymentChangeStatusNotification", err)
		c.sendAnyError()
		return
	}

	_ = c.Send(tgbotapi.NewMessage(t.MemberID, fmt.Sprintf("Статус оплаты изменен на: %s", t.Status)))
}

func (c *Chat) createDebitingFunds() {
	sum, err := c.getFloatFromUser("Введите сумму списания")
	if err != nil {
		if !errors.Is(err, Close) {
			c.sendAttemptsExceededError()
		}
		return
	}

	if err = c.Send(tgbotapi.NewMessage(c.chatId, "Укажите причину списания")); err != nil {
		return
	}

	purpose, err := c.getResponse(typeOfResponseText)
	if err != nil {
		if !errors.Is(err, Close) {
			c.sendAttemptsExceededError()
		}
		return
	}

	tag, err := c.DB.GetTag(c.chatId)
	if err != nil {
		c.writeToLog("createDebitingFunds/GetTag", err)
		return
	}

	if err = c.Send(tgbotapi.NewMessage(c.chatId, "Прикрепите чек")); err != nil {
		return
	}

	attachment, err := c.getResponse(typeOfResponseAttachment)
	if err != nil {
		if !errors.Is(err, Close) {
			c.sendAttemptsExceededError()
		}
		return
	}

	var idFile string
	if attachment.Photo != nil {
		idFile = attachment.Photo[len(attachment.Photo)-1].FileID
	} else {
		idFile = attachment.Document.FileID
	}

	fileName, err := c.downloadAttachment(idFile)

	if err != nil {
		c.writeToLog("createDebitingFunds/downloadAttachment", err)
		c.sendAnyError()
		return
	}

	if ok, err := c.DB.CreateDebitingFunds(db.CashCollection{
		Tag:        tag,
		Sum:        sum,
		Comment:    fmt.Sprintf("Инициатор: %s", c.username),
		CreateDate: time.Now(),
		Purpose:    purpose.Text,
	}, c.chatId, fileName); err != nil || !ok {
		c.writeToLog("CreateDebitingFunds", err)
		c.sendAnyError()
		return
	}

	_ = c.Send(tgbotapi.NewMessage(c.chatId, "Списание проведено успешно"))

	if err = c.DebitingNotification(tag, sum, purpose.Text, fileName); err != nil {
		c.writeToLog("DebitingNotification/GetMembers", err)
		_ = c.Send(tgbotapi.NewMessage(c.chatId, "Не удалось оповестить участников о списании"))
	}
}

func (c *Chat) downloadAttachment(fileId string) (fileName string, err error) {
	bot := c.Service.GetBot()

	_, err = bot.GetFile(tgbotapi.FileConfig{FileID: fileId})
	if err != nil {
		return
	}

	pathFile, err := bot.GetFileDirectURL(fileId)
	if err != nil {
		return
	}

	resp, err := http.Get(pathFile)
	defer resp.Body.Close()
	if err != nil {
		return
	}

	fileName, err = c.FTP.StoreFile(path.Ext(pathFile), resp.Body)
	if err != nil {
		return "", err
	}

	return
}

// getFloatFromUser получить вещественное число от пользователя. Возвращает AttemptsExceeded
func (c *Chat) getFloatFromUser(message string) (float64, error) {
	var sum float64
	if err := c.Send(tgbotapi.NewMessage(c.chatId, message)); err != nil {
		return sum, err
	}

	for i := 0; i < 3; i++ {
		answer, err := c.getResponse(typeOfResponseText)
		if err != nil {
			return sum, err
		}

		sum, err = strconv.ParseFloat(answer.Text, 64)
		if err != nil {
			msg := tgbotapi.NewMessage(c.chatId, "Неверный ввод. Повторите попытку")
			if i == 2 {
				msg.Text = ""
			}
			if err = c.Send(msg); err != nil {
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
	err := c.Send(tgbotapi.NewMessage(c.chatId, "Представьтесь, пожалуйста. Введите ФИО"))
	if err != nil {
		return "", err
	}

	answer, err := c.getResponse(typeOfResponseText)
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

		if answer = <-userChan; answer == nil {
			return answer, Close
		}

		if answer.Photo != nil || answer.Document != nil {
			typeOfMessage = typeOfResponseAttachment
		} else {
			typeOfMessage = typeOfResponseText
		}

		if typeOfResponse != typeOfMessage {
			if i < 2 {
				if err := c.Send(tgbotapi.NewMessage(c.chatId, fmt.Sprintf("Вы ввели что-то не то. Количество доступных попыток: %d", 2-i))); err != nil {
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
	if err := c.Send(tgbotapi.NewMessage(c.chatId, "Произошла ошибка. Повторите попытку позже")); err != nil {
		c.writeToLog("sendError", err)
	}
}

func (c *Chat) sendAttemptsExceededError() {
	if err := c.Send(tgbotapi.NewMessage(c.chatId, "Превышено число попыток ввода")); err != nil {
		c.writeToLog("sendAttemptsExceededError", err)
	}
}

// newTag формирует новый тег. Выполняет проверку на существование. Если Тег уже существует формирует новый рекурсивно
func (c *Chat) newTag() (string, error) {
	symbols := []byte(alphabet)
	result := make([]byte, rand.Intn(5)+5)
	for i := range result {
		result[i] = symbols[rand.Intn(len(symbols))]
	}

	tag := string(result)

	ok, err := c.DB.DoesTagExist(tag)
	if err != nil || !ok {
		return tag, err
	} else {
		return c.newTag()
	}
}

func (c *Chat) writeToLog(location string, err error) {
	log.Println(c.chatId, location, err)
}

// showListDebtors список должников
func (c *Chat) showListDebtors() {
	tag, err := c.DB.GetTag(c.chatId)
	if err != nil {
		c.writeToLog("showListDebtors/GetTag", err)
		c.sendAnyError()
	}

	openCollections, err := c.DB.FindCashCollectionByStatus(tag, db.StatusCashCollectionOpen)
	if err != nil {
		c.writeToLog("showListDebtors/FindCashCollectionByStatus", err)
		c.sendAnyError()
	}

	var strBuilder strings.Builder

	if len(openCollections) == 0 {
		strBuilder.WriteString("Должников нет")
		_ = c.Send(tgbotapi.NewMessage(c.chatId, strBuilder.String()))
		return
	}

	for i, collection := range openCollections {

		strBuilder.WriteString(openCollections[i].Purpose + ":\n")

		debtorsID, err := c.DB.GetDebtorsByCollection(collection.ID)
		if err != nil {
			c.writeToLog("showListDebtors/GetDebtorsByCollection", err)
			c.sendAnyError()
		}

		for j, debtor := range debtorsID {
			member, err := c.DB.GetInfoAboutMember(debtor)
			if err != nil {
				c.writeToLog("showListDebtors/GetInfoAboutMember", err)
				c.sendAnyError()
			}

			strBuilder.WriteString(fmt.Sprintf("%d) %s (@%s)\n", j+1, member.Name, member.Login))
		}

		strBuilder.WriteString("\n")

	}

	_ = c.Send(tgbotapi.NewMessage(c.chatId, strBuilder.String()))
}

func (c *Chat) DebitingNotification(tag string, sum float64, purpose string, receipt string) error {

	members, err := c.DB.GetMembers(tag)
	if err != nil {
		return err
	}

	bot := c.GetBot()

	fb, err := c.FTP.ReadFile(receipt)
	if err != nil {
		return err
	}

	doc := tgbotapi.FileBytes{
		Name:  receipt,
		Bytes: fb,
	}

	for _, member := range members {
		if member.ID != c.chatId {
			_ = c.Send(tgbotapi.NewMessage(member.ID, fmt.Sprintf("Списаны средства\nНазначение: %s\nСумма: %.2f", purpose, sum)))
			_, _ = bot.Send(tgbotapi.NewDocument(member.ID, doc))
		}
	}

	return nil
}

func (c *Chat) deleteMember() {
	members, err := c.getListMembers()
	if err != nil {
		c.writeToLog("deleteMember/getListMembers", err)
	}

	msg := tgbotapi.NewMessage(c.chatId, "Введите номер пользователя, которого необходимо удалить")
	if err := c.Send(msg); err != nil {
		c.writeToLog("deleteMember/send", err)
		return
	}

	response, err := c.getResponse(typeOfResponseText)
	if err != nil {
		if !errors.Is(err, Close) {
			c.sendAttemptsExceededError()
		}
		return
	}

	var number int

	for i := 0; i < 3; i++ {
		number, err = strconv.Atoi(response.Text)
		if err != nil {
			if err = c.Send(tgbotapi.NewMessage(c.chatId, "Введите число")); err != nil {
				c.writeToLog("deleteMember/send", err)
				return
			}
			continue
		}
		break
	}

	msg.Text = fmt.Sprintf("Вы действительно хотите удалить %s (@%s)?", members[number-1].Name, members[number-1].Login)

	var yesNoKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(c.Buttons.DeleteMemberYes.Label, c.Buttons.DeleteMemberYes.Command+strconv.FormatInt(members[number-1].ID, 10)),
			tgbotapi.NewInlineKeyboardButtonData(c.Buttons.No.Label, c.Buttons.No.Command),
		),
	)

	msg.ReplyMarkup = &yesNoKeyboard
	_ = c.Send(msg)

}

func (c *Chat) getMembers() {
	members, err := c.getListMembers()
	if err != nil {
		c.writeToLog("getMembers/getListMembers", err)
		c.sendAnyError()
		return
	}

	msg := c.formatListMembers(members)

	var numericKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(c.Buttons.DeleteMember.Label, c.Buttons.DeleteMember.Command)))
	msg.ReplyMarkup = &numericKeyboard

	_ = c.Send(msg)
}

func (c *Chat) deleteMemberYes(id int64) {
	tag, err := c.DB.GetTag(c.chatId)
	if err != nil {
		c.writeToLog("deleteMemberYes/GetTag", err)
		c.sendAnyError()
		return
	}

	if err = c.DB.DeleteMember(tag, id); err != nil {
		c.writeToLog("deleteMemberYes/DeleteMember", err)
		c.sendAnyError()
		return
	}

	_ = c.Send(tgbotapi.NewMessage(c.chatId, "Пользователь удален"))
}

func (c *Chat) leave() {
	member, err := c.DB.GetInfoAboutMember(c.chatId)
	if err != nil {
		c.writeToLog("leave/GetInfoAboutMember", err)
		c.sendAnyError()
		return
	}

	if member.IsAdmin {
		msg := tgbotapi.NewMessage(c.chatId, "Вы являетесь администратором и не можете покинуть фонд")
		var setAdminKeyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(c.Buttons.SetAdmin.Label, c.Buttons.SetAdmin.Command),
			),
		)

		msg.ReplyMarkup = &setAdminKeyboard
		_ = c.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(c.chatId, "Вы действительно хотите покинуть фонд?")

	var yesNoKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(c.Buttons.LeaveYes.Label, c.Buttons.LeaveYes.Command),
			tgbotapi.NewInlineKeyboardButtonData(c.Buttons.No.Label, c.Buttons.No.Command),
		),
	)

	msg.ReplyMarkup = &yesNoKeyboard
	_ = c.Send(msg)
}

func (c *Chat) leaveYes() {
	tag, err := c.DB.GetTag(c.chatId)
	if err != nil {
		c.writeToLog("leaveYes/GetTag", err)
		c.sendAnyError()
		return
	}

	if err = c.DB.DeleteMember(tag, c.chatId); err != nil {
		c.writeToLog("leaveYes/DeleteMember", err)
		c.sendAnyError()
		return
	}

	_ = c.Send(tgbotapi.NewMessage(c.chatId, "Вы покинули фонд"))
	c.startMenu()
}

func (c *Chat) showTag() {
	tag, err := c.DB.GetTag(c.chatId)
	if err != nil {
		c.writeToLog("showTag/GetTag", err)
		c.sendAnyError()
		return
	}

	_ = c.Send(tgbotapi.NewMessage(c.chatId, fmt.Sprintf("Тег фонда: %s", tag)))

}

func (c *Chat) showHistory(page int) {
	tag, err := c.DB.GetTag(c.chatId)
	if err != nil {
		c.writeToLog("showHistory/GetTag", err)
		c.sendAnyError()
		return
	}
	list, err := c.DB.History(tag, page)
	if err != nil {
		c.writeToLog("showHistory", err)
		c.sendAnyError()
		return
	}

	bot := c.GetBot()
	for _, data := range list {
		fb, err := c.FTP.ReadFile(data.Receipt)
		if err != nil {
			c.writeToLog("showHistory/ReadFile", err)
			c.sendAnyError()
			return
		}
		doc := tgbotapi.FileBytes{
			Name:  data.Receipt,
			Bytes: fb,
		}

		_ = c.Send(tgbotapi.NewMessage(c.chatId, fmt.Sprintf("Назначение: %s\nСумма: %.2f\nДата: %s", data.Purpose, data.Sum, data.Date.Format(layoutDate))))

		_, _ = bot.Send(tgbotapi.NewDocument(c.chatId, doc))
	}

	switch count := len(list); count {
	case db.NumberEntriesPerPage:
		msg := tgbotapi.NewMessage(c.chatId, "Показать предыдущие?")

		var nextKeyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(c.Buttons.NextPageHistory.Label, c.Buttons.NextPageHistory.Command+strconv.Itoa(page+1))),
		)

		msg.ReplyMarkup = &nextKeyboard
		_ = c.Send(msg)
	default:
		_ = c.Send(tgbotapi.NewMessage(c.chatId, "Больше списаний нет"))
	}

}

func (c *Chat) awaitingPayment() {
	tag, err := c.DB.GetTag(c.chatId)
	if err != nil {
		c.writeToLog("awaitingPayment/GetTag", err)
		c.sendAnyError()
	}

	openCollections, err := c.DB.FindCashCollectionByStatus(tag, db.StatusCashCollectionOpen)
	if err != nil {
		c.writeToLog("awaitingPayment/FindCashCollectionByStatus", err)
		c.sendAnyError()
	}

	count := 0
	for _, collection := range openCollections {
		debtorsID, err := c.DB.GetDebtorsByCollection(collection.ID)
		if err != nil {
			c.writeToLog("showListDebtors/GetDebtorsByCollection", err)
			c.sendAnyError()
			return
		}

		for _, debtor := range debtorsID {
			if debtor == c.chatId {
				msg := tgbotapi.NewMessage(c.chatId, fmt.Sprintf("Назначение: %s\nСумма: %.2f", collection.Purpose, collection.Sum))

				var paymentKeyboard = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData(c.Buttons.Payment.Label, c.Buttons.Payment.Command+strconv.Itoa(collection.ID)),
					),
				)
				msg.ReplyMarkup = &paymentKeyboard
				_ = c.Send(msg)
				count++
				continue
			}
		}

	}

	if count == 0 {
		_ = c.Send(tgbotapi.NewMessage(c.chatId, "Задолженностей нет"))
	}

}

func (c *Chat) setAdmin() {
	members, err := c.getListMembers()
	if err != nil {
		c.writeToLog("setAdmin/getListMembers", err)
		c.sendAnyError()
		return
	}

	msg := c.formatListMembers(members)

	if err = c.Send(msg); err != nil {
		c.writeToLog("setAdmin/Send", err)
		return
	}

	if err = c.Send(tgbotapi.NewMessage(c.chatId, "Введите номер участника, которого вы хотите назначить администратором")); err != nil {
		c.writeToLog("setAdmin/Send", err)
		return
	}

	response, err := c.getResponse(typeOfResponseText)
	if err != nil {
		if !errors.Is(err, Close) {
			c.sendAttemptsExceededError()
		}
		return
	}

	var number int

	for i := 0; i < 3; i++ {
		number, err = strconv.Atoi(response.Text)
		if err != nil {
			if err = c.Send(tgbotapi.NewMessage(c.chatId, "Введите число")); err != nil {
				c.writeToLog("setAdmin/send", err)
				return
			}
			continue
		}
		break
	}

	if members[number-1].ID == c.chatId {
		_ = c.Send(tgbotapi.NewMessage(c.chatId, "Вы уже являетесь администратором"))
		return
	}
	msg.Text = fmt.Sprintf("Вы действительно хотите назначить администратором %s (@%s)?", members[number-1].Name, members[number-1].Login)

	var yesNoKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(c.Buttons.SetAdminYes.Label, c.Buttons.SetAdminYes.Command+strconv.FormatInt(members[number-1].ID, 10)),
			tgbotapi.NewInlineKeyboardButtonData(c.Buttons.No.Label, c.Buttons.No.Command),
		),
	)

	msg.ReplyMarkup = &yesNoKeyboard
	_ = c.Send(msg)
}

func (c *Chat) setAdminYes(id int64) {
	tag, err := c.DB.GetTag(c.chatId)
	if err != nil {
		c.writeToLog("setAdminYes/GetTag", err)
		c.sendAnyError()
		return
	}

	if ok, err := c.DB.SetAdmin(tag, c.chatId, id); err != nil || !ok {
		c.writeToLog("setAdminYes", err)
		c.sendAnyError()
		return
	}

	_ = c.Send(tgbotapi.NewMessage(c.chatId, "Администратор сменен"))

	c.setAdminNotification(id)
}

func (c *Chat) setAdminNotification(id int64) {
	_ = c.Send(tgbotapi.NewMessage(id, "Вас назначили администратором"))
}
