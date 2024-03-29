package chat

import (
	"context"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"math/rand"
	"net/http"
	"path"
	"project1/internal/button"
	"project1/internal/db"
	"project1/internal/fileStorage"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	alphabet                 = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	typeOfResponseText       = "text"
	typeOfResponseAttachment = "attachment"
	layoutDate               = "02.01.2006"
	currency                 = "руб"
)

type Chat struct {
	ctx         context.Context
	username    string
	chatId      int64
	bot         *tgbotapi.BotAPI
	db          *db.Repository
	ftp         *fileStorage.FileStorage
	buttons     button.List
	waitingList map[int64]chan *tgbotapi.Message
	wg          *sync.RWMutex
	router      map[string]func(...string)
}

func NewChat(ctx context.Context, username string, chatId int64, bot *tgbotapi.BotAPI, db *db.Repository, ftp *fileStorage.FileStorage, buttons button.List, waitingList map[int64]chan *tgbotapi.Message, wg *sync.RWMutex) *Chat {
	ch := Chat{
		ctx:         ctx,
		username:    username,
		chatId:      chatId,
		bot:         bot,
		db:          db,
		ftp:         ftp,
		buttons:     buttons,
		waitingList: waitingList,
		wg:          wg,
	}
	router := make(map[string]func(...string))
	router[button.Start] = ch.startMenu
	router[button.CreateFund] = ch.createFund
	router[button.CreateFundYes] = ch.createFundYes
	router[button.Join] = ch.join
	router[button.ShowBalance] = ch.showBalance
	router[button.CreateCashCollection] = ch.createCashCollection
	router[button.CreateDebitingFunds] = ch.createDebitingFunds
	router[button.Members] = ch.getMembers
	router[button.Payment] = ch.payment
	router[button.PaymentAccept] = ch.changeStatusOfTransaction
	router[button.PaymentReject] = ch.changeStatusOfTransaction
	router[button.PaymentWait] = ch.changeStatusOfTransaction
	router[button.Menu] = ch.showMenu
	router[button.ShowListDebtors] = ch.showListDebtors
	router[button.DeleteMember] = ch.deleteMember
	router[button.DeleteMemberYes] = ch.deleteMemberYes
	router[button.Leave] = ch.leave
	router[button.LeaveYes] = ch.leaveYes
	router[button.ShowTag] = ch.showTag
	router[button.History] = ch.showHistory
	router[button.AwaitingPayment] = ch.awaitingPayment
	router[button.SetAdmin] = ch.setAdmin
	router[button.SetAdminYes] = ch.setAdminYes

	ch.router = router
	return &ch
}

func (c *Chat) getUserChan(id int64) (chan *tgbotapi.Message, bool) {
	c.wg.RLock()
	userChan, ok := c.waitingList[id]
	c.wg.RUnlock()
	return userChan, ok
}

func (c *Chat) waitingResponse() {
	c.wg.Lock()
	c.waitingList[c.chatId] = make(chan *tgbotapi.Message)
	c.wg.Unlock()
}

func (c *Chat) stopWaiting() {
	c.wg.Lock()
	if ch, ok := c.waitingList[c.chatId]; ok {
		close(ch)
		delete(c.waitingList, c.chatId)
	}
	c.wg.Unlock()
}

// Send 3 попытки на отправку, иначе удалить из списка ожидания и вернуть ошибку. Возвращает AttemptsExceeded
func (c *Chat) Send(data tgbotapi.Chattable) error {

	for i := 0; i < 3; i++ {
		_, err := c.bot.Send(data)
		if err == nil {
			return nil
		}
	}

	c.stopWaiting()
	return AttemptsExceeded
}

func (c *Chat) CommandRouter(query string) bool {
	cmd := strings.Split(query, "/")

	if len(cmd) == 0 {
		return false
	}

	f, ok := c.router[cmd[0]]
	if !ok {
		return false
	}

	f(cmd...)
	return true
}

func (c *Chat) startMenu(...string) {
	var startKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(c.buttons.CreateFound.Label, c.buttons.CreateFound.Command),
			tgbotapi.NewInlineKeyboardButtonData(c.buttons.Join.Label, c.buttons.Join.Command),
		),
	)

	msg := tgbotapi.NewMessage(c.chatId, "Приветствую! Выберите один из вариантов")
	msg.ReplyMarkup = &startKeyboard

	_ = c.Send(msg)
}

func (c *Chat) showMenu(...string) {
	ok, err := c.db.IsMember(c.ctx, c.chatId)
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
	msg := tgbotapi.NewMessage(c.chatId, "Приветствую! Выберите один из вариантов")

	member, err := c.db.GetInfoAboutMember(c.ctx, c.chatId)
	if err != nil {
		c.writeToLog("showMenu/GetInfoAboutMember", err)
		c.sendAnyError()
		return
	}

	var menuKeyboard = tgbotapi.NewInlineKeyboardMarkup( //меню для обычного пользователя
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(c.buttons.ShowBalance.Label, c.buttons.ShowBalance.Command),
			tgbotapi.NewInlineKeyboardButtonData(c.buttons.AwaitingPayment.Label, c.buttons.AwaitingPayment.Command),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(c.buttons.History.Label, c.buttons.History.Command+"/0"),
			tgbotapi.NewInlineKeyboardButtonData(c.buttons.Leave.Label, c.buttons.Leave.Command),
		),
	)

	if member.IsAdmin { // если админ, то дополнить меню
		menuKeyboard.InlineKeyboard = append(menuKeyboard.InlineKeyboard,
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(c.buttons.CreateCashCollection.Label, c.buttons.CreateCashCollection.Command),
				tgbotapi.NewInlineKeyboardButtonData(c.buttons.CreateDebitingFunds.Label, c.buttons.CreateDebitingFunds.Command),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(c.buttons.Members.Label, c.buttons.Members.Command),
				tgbotapi.NewInlineKeyboardButtonData(c.buttons.DebtorList.Label, c.buttons.DebtorList.Command),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(c.buttons.ShowTag.Label, c.buttons.ShowTag.Command),
				tgbotapi.NewInlineKeyboardButtonData(c.buttons.SetAdmin.Label, c.buttons.SetAdmin.Command),
			),
		)
	}

	fmt.Println(c.buttons.ShowTag.Label, c.buttons.ShowTag.Command)
	msg.ReplyMarkup = &menuKeyboard

	_ = c.Send(msg)

}

// createFund проверяет состоит ли пользователь в другом фонде, если не состоит, то запрашивает подтверждение операции
func (c *Chat) createFund(...string) {
	ok, err := c.db.IsMember(c.ctx, c.chatId)
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
			tgbotapi.NewInlineKeyboardButtonData(c.buttons.CreateFoundYes.Label, c.buttons.CreateFoundYes.Command),
			tgbotapi.NewInlineKeyboardButtonData(c.buttons.CreateFoundNo.Label, c.buttons.CreateFoundNo.Command),
		),
	)
	msg.ReplyMarkup = &numericKeyboard

	_ = c.Send(msg)
}

// createFundYes создает новый фонд
func (c *Chat) createFundYes(...string) {
	sum, err := c.getFloatFromUser("Введите начальную сумму фонда")
	if err != nil {
		if !errors.Is(err, Close) {
			c.sendAttemptsExceededError()
		}
		return
	}

	tag, err := c.newTag()
	if err != nil {
		c.writeToLog("createFundYes/newTag", err)
		c.sendAnyError()
	}

	name, err := c.getName()
	if err != nil {
		if !errors.Is(err, Close) {
			c.sendAttemptsExceededError()
		}
		return
	}

	if err = c.db.CreateFund(c.ctx, tag, sum); err != nil {
		c.writeToLog("createFundYes", err)
		c.sendAnyError()
		return
	}

	if err = c.db.AddMember(c.ctx, db.Member{
		ID:      c.chatId,
		Tag:     tag,
		IsAdmin: true,
		Login:   c.username,
		Name:    name,
	}); err != nil {
		c.writeToLog("createFundYes/AddMember", err)
		err = c.db.DeleteFund(c.ctx, tag)
		c.writeToLog("createFundYes/DeleteFund", err)
		c.sendAnyError()
		return
	}

	if err = c.Send(tgbotapi.NewMessage(c.chatId, fmt.Sprintf("Новый фонд создан успешно! Присоединиться к фонду можно, используя тег: %s \nВнимание! Не показывайте этот тег посторонним людям.", tag))); err != nil {
		if err = c.db.DeleteFund(c.ctx, tag); err != nil {
			c.writeToLog("createFundYes/DeleteFund", err)
		}
		return
	}

}

func (c *Chat) showBalance(...string) {
	tag, err := c.db.GetTag(c.ctx, c.chatId)
	if err != nil {
		c.writeToLog("showBalance/getTag", err)
		c.sendAnyError()
		return
	}
	balance, err := c.db.ShowBalance(c.ctx, tag)
	if err != nil {
		c.writeToLog("showBalance", err)
		c.sendAnyError()
		return
	}

	_ = c.Send(tgbotapi.NewMessage(c.chatId, fmt.Sprintf("Текущий баланс фонда: %.2f %s", balance, currency)))
}

func (c *Chat) join(...string) {
	ok, err := c.db.IsMember(c.ctx, c.chatId)
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

	ok, err = c.db.DoesTagExist(c.ctx, tag)
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

	if err = c.db.AddMember(c.ctx, db.Member{
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
	tag, err := c.db.GetTag(c.ctx, c.chatId)
	if err != nil {
		return []db.Member{}, err
	}

	return c.db.GetMembers(c.ctx, tag)
}

func (c *Chat) createCashCollection(...string) {
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

	tag, err := c.db.GetTag(c.ctx, c.chatId)
	if err != nil {
		c.writeToLog("createCashCollection/GetTag", err)
		c.sendAnyError()
		return
	}

	id, err := c.db.CreateCashCollection(c.ctx, db.CashCollection{
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
	members, err := c.db.GetMembers(c.ctx, tagFund)
	if err != nil {
		c.writeToLog("collectionNotification/GetMembers", err)
		c.sendAnyError()
		return
	}
	cc, err := c.db.InfoAboutCashCollection(c.ctx, idCollection)
	if err != nil {
		c.writeToLog("collectionNotification/InfoAboutCashCollection", err)
		c.sendAnyError()
		return
	}

	var paymentKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(c.buttons.Payment.Label, c.buttons.Payment.Command+"/"+strconv.Itoa(idCollection)),
		),
	)

	for _, member := range members {
		msg := tgbotapi.NewMessage(member.ID, fmt.Sprintf("Иницирован новый сбор.\nСумма к оплате: %.2f %s\nНазначение: %s", cc.Sum, currency, cc.Purpose))
		msg.ReplyMarkup = &paymentKeyboard
		_ = c.Send(msg)
	}
}

func (c *Chat) payment(args ...string) {
	cashCollectionId, err := strconv.Atoi(args[1])
	if err != nil {
		c.writeToLog("payment/ParseInt", err)
		c.sendAnyError()
		return
	}

	cc, err := c.db.InfoAboutCashCollection(c.ctx, cashCollectionId)
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

	idTransaction, err := c.db.InsertInTransactions(c.ctx, db.Transaction{
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
	tag, err := c.db.GetTag(c.ctx, c.chatId)
	if err != nil {
		c.writeToLog("paymentNotification/GetTag", err)
		c.sendAnyError()
		return
	}
	adminId, err := c.db.GetAdminFund(c.ctx, tag)
	if err != nil {
		c.writeToLog("paymentNotification/GetAdminFund", err)
		c.sendAnyError()
		return
	}

	member, err := c.db.GetInfoAboutMember(c.ctx, c.chatId)
	if err != nil {
		c.writeToLog("paymentNotification/GetInfoAboutMember", err)
		c.sendAnyError()
		return
	}

	var okKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(c.buttons.PaymentConfirmation.Label, fmt.Sprintf("%s/%s/%s", c.buttons.PaymentConfirmation.Command, strconv.Itoa(idTransaction), db.StatusPaymentConfirmation)),
			tgbotapi.NewInlineKeyboardButtonData(c.buttons.PaymentRefusal.Label, fmt.Sprintf("%s/%s/%s", c.buttons.PaymentRefusal.Command, strconv.Itoa(idTransaction), db.StatusPaymentRejection)),
			tgbotapi.NewInlineKeyboardButtonData(c.buttons.PaymentExpected.Label, fmt.Sprintf("%s/%s/%s", c.buttons.PaymentExpected.Command, strconv.Itoa(idTransaction), db.StatusPaymentExpectation)),
		),
	)

	msg := tgbotapi.NewMessage(adminId, fmt.Sprintf("Подтвердите зачисление средств на счет фонда.\nСумма: %.2f %s\nОтправитель: %s", sum, currency, member.Name))
	msg.ReplyMarkup = &okKeyboard
	_ = c.Send(msg)

}

// changeStatusOfTransaction изменение статуса транзакции
func (c *Chat) changeStatusOfTransaction(args ...string) {
	idTransaction, err := strconv.Atoi(args[1])
	if err != nil {
		c.writeToLog("changeStatusOfTransaction/ParseInt", err)
		c.sendAnyError()
		return
	}

	if err = c.db.ChangeStatusTransaction(c.ctx, idTransaction, args[2]); err != nil {
		c.writeToLog("changeStatusOfTransaction", err)
		c.sendAnyError()
		return
	}

	_ = c.Send(tgbotapi.NewMessage(c.chatId, fmt.Sprintf("Статус оплаты: %s", args[2])))

	t, err := c.db.InfoAboutTransaction(c.ctx, idTransaction)
	if err != nil {
		c.writeToLog("changeStatusOfTransaction/InfoAboutTransaction", err)
	}

	if err = c.db.UpdateStatusCashCollection(c.ctx, t.CashCollectionID); err != nil {
		c.writeToLog("changeStatusOfTransaction/CheckDebtors", err)
	}

	c.paymentChangeStatusNotification(idTransaction)
}

func (c *Chat) paymentChangeStatusNotification(idTransaction int) {
	t, err := c.db.InfoAboutTransaction(c.ctx, idTransaction)
	if err != nil {
		c.writeToLog("paymentChangeStatusNotification", err)
		c.sendAnyError()
		return
	}

	_ = c.Send(tgbotapi.NewMessage(t.MemberID, fmt.Sprintf("Статус оплаты изменен на: %s", t.Status)))
}

func (c *Chat) createDebitingFunds(...string) {
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

	tag, err := c.db.GetTag(c.ctx, c.chatId)
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

	if ok, err := c.db.CreateDebitingFunds(c.ctx, db.CashCollection{
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
	_, err = c.bot.GetFile(tgbotapi.FileConfig{FileID: fileId})
	if err != nil {
		return
	}

	pathFile, err := c.bot.GetFileDirectURL(fileId)
	if err != nil {
		return
	}

	resp, err := http.Get(pathFile)
	defer resp.Body.Close()
	if err != nil {
		return
	}

	fileName, err = c.ftp.StoreFile(path.Ext(pathFile), resp.Body)
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
	c.waitingResponse()
	defer c.stopWaiting()

	var typeOfMessage string
	var answer *tgbotapi.Message

	for i := 0; i < 3; i++ {
		userChan, _ := c.getUserChan(c.chatId)

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

	ok, err := c.db.DoesTagExist(c.ctx, tag)
	if err != nil || !ok {
		return tag, err
	} else {
		return c.newTag()
	}
}

func (c *Chat) writeToLog(location string, err error) {
	log.Println(c.chatId, location, err)
}

// showListDebtors отправляет список должников
func (c *Chat) showListDebtors(...string) {
	debtors, err := c.getListDebtors(db.StatusCashCollectionOpen)
	if err != nil {
		c.writeToLog("showListDebtors/getListDebtors", err)
		c.sendAnyError()
		return
	}

	var strBuilder strings.Builder

	if len(debtors) == 0 {
		strBuilder.WriteString("Должников нет")
		_ = c.Send(tgbotapi.NewMessage(c.chatId, strBuilder.String()))
		return
	}

	for cc, debtorList := range debtors {
		strBuilder.WriteString(fmt.Sprintf("%s:\n", cc.Purpose))

		for i, debtor := range debtorList {
			strBuilder.WriteString(fmt.Sprintf("%d) %s (@%s)\n", i+1, debtor.Name, debtor.Login))
		}

		strBuilder.WriteString("\n")

	}

	_ = c.Send(tgbotapi.NewMessage(c.chatId, strBuilder.String()))
}

// getListDebtors возвращает список должников по статусу CashCollection
func (c *Chat) getListDebtors(status string) (debtors map[db.CashCollection][]db.Member, err error) {
	tag, err := c.db.GetTag(c.ctx, c.chatId)
	if err != nil {
		return debtors, fmt.Errorf("GetTag:%w", err)
	}

	collections, err := c.db.FindCashCollectionByStatus(c.ctx, tag, status)
	if err != nil {
		return debtors, fmt.Errorf("FindCashCollectionByStatus:%w", err)
	}

	debtors = make(map[db.CashCollection][]db.Member)

	for _, collection := range collections {

		debtorsByCC, err := c.db.GetDebtorsByCollection(c.ctx, collection.ID)
		if err != nil {
			return debtors, fmt.Errorf("GetDebtorsByCollection:%w", err)
		}

		debtors[collection] = debtorsByCC

	}
	return

}

func (c *Chat) DebitingNotification(tag string, sum float64, purpose string, receipt string) error {

	members, err := c.db.GetMembers(c.ctx, tag)
	if err != nil {
		return err
	}

	fb, err := c.ftp.ReadFile(receipt)
	if err != nil {
		return err
	}

	doc := tgbotapi.FileBytes{
		Name:  receipt,
		Bytes: fb,
	}

	for _, member := range members {
		if member.ID != c.chatId {
			document := tgbotapi.NewDocument(member.ID, doc)
			document.Caption = fmt.Sprintf("Списаны средства\nНазначение: %s\nСумма: %.2f %s", purpose, sum, currency)
			_ = c.Send(document)

		}
	}

	return nil
}

func (c *Chat) deleteMember(...string) {
	members, err := c.getListMembers()
	if err != nil {
		c.writeToLog("deleteMember/getListMembers", err)
	}

	msg := tgbotapi.NewMessage(c.chatId, "Введите номер пользователя, которого необходимо удалить")
	if err := c.Send(msg); err != nil {
		c.writeToLog("deleteMember/send", err)
		return
	}

	var number int

	for i := 0; i < 5; i++ {
		response, err := c.getResponse(typeOfResponseText)
		if err != nil || i == 4 {
			if !errors.Is(err, Close) {
				c.sendAttemptsExceededError()
			}
			return
		}

		number, err = strconv.Atoi(response.Text)
		if err != nil {
			if err = c.Send(tgbotapi.NewMessage(c.chatId, "Введите число")); err != nil {
				c.writeToLog("deleteMember/send", err)
				return
			}
			continue
		}

		if number < 1 || number > len(members) {
			if err = c.Send(tgbotapi.NewMessage(c.chatId, "Введите корректное число")); err != nil {
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
			tgbotapi.NewInlineKeyboardButtonData(c.buttons.DeleteMemberYes.Label, c.buttons.DeleteMemberYes.Command+"/"+strconv.FormatInt(members[number-1].ID, 10)),
			tgbotapi.NewInlineKeyboardButtonData(c.buttons.No.Label, c.buttons.No.Command),
		),
	)

	msg.ReplyMarkup = &yesNoKeyboard
	_ = c.Send(msg)

}

func (c *Chat) getMembers(...string) {
	members, err := c.getListMembers()
	if err != nil {
		c.writeToLog("getMembers/getListMembers", err)
		c.sendAnyError()
		return
	}

	msg := c.formatListMembers(members)

	var numericKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(c.buttons.DeleteMember.Label, c.buttons.DeleteMember.Command)))
	msg.ReplyMarkup = &numericKeyboard

	_ = c.Send(msg)
}

func (c *Chat) deleteMemberYes(args ...string) {
	id, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		c.writeToLog("deleteMemberYes/ParseInt", err)
		c.sendAnyError()
		return
	}

	tag, err := c.db.GetTag(c.ctx, c.chatId)
	if err != nil {
		c.writeToLog("deleteMemberYes/GetTag", err)
		c.sendAnyError()
		return
	}

	if err = c.db.DeleteMember(c.ctx, tag, id); err != nil {
		c.writeToLog("deleteMemberYes/DeleteMember", err)
		c.sendAnyError()
		return
	}

	_ = c.Send(tgbotapi.NewMessage(c.chatId, "Пользователь удален"))
}

func (c *Chat) leave(...string) {
	member, err := c.db.GetInfoAboutMember(c.ctx, c.chatId)
	if err != nil {
		c.writeToLog("leave/GetInfoAboutMember", err)
		c.sendAnyError()
		return
	}

	if member.IsAdmin {
		msg := tgbotapi.NewMessage(c.chatId, "Вы являетесь администратором и не можете покинуть фонд")
		var setAdminKeyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(c.buttons.SetAdmin.Label, c.buttons.SetAdmin.Command),
			),
		)

		msg.ReplyMarkup = &setAdminKeyboard
		_ = c.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(c.chatId, "Вы действительно хотите покинуть фонд?")

	var yesNoKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(c.buttons.LeaveYes.Label, c.buttons.LeaveYes.Command),
			tgbotapi.NewInlineKeyboardButtonData(c.buttons.No.Label, c.buttons.No.Command),
		),
	)

	msg.ReplyMarkup = &yesNoKeyboard
	_ = c.Send(msg)
}

func (c *Chat) leaveYes(...string) {
	tag, err := c.db.GetTag(c.ctx, c.chatId)
	if err != nil {
		c.writeToLog("leaveYes/GetTag", err)
		c.sendAnyError()
		return
	}

	if err = c.db.DeleteMember(c.ctx, tag, c.chatId); err != nil {
		c.writeToLog("leaveYes/DeleteMember", err)
		c.sendAnyError()
		return
	}

	_ = c.Send(tgbotapi.NewMessage(c.chatId, "Вы покинули фонд"))
	c.startMenu()
}

func (c *Chat) showTag(...string) {
	tag, err := c.db.GetTag(c.ctx, c.chatId)
	if err != nil {
		c.writeToLog("showTag/GetTag", err)
		c.sendAnyError()
		return
	}

	_ = c.Send(tgbotapi.NewMessage(c.chatId, fmt.Sprintf("Тег фонда: %s", tag)))

}

func (c *Chat) showHistory(args ...string) {
	page, err := strconv.Atoi(args[1])
	if err != nil {
		c.writeToLog("showHistory/strconvAtoi", err)
		c.sendAnyError()
		return
	}

	tag, err := c.db.GetTag(c.ctx, c.chatId)
	if err != nil {
		c.writeToLog("showHistory/GetTag", err)
		c.sendAnyError()
		return
	}
	list, err := c.db.History(c.ctx, tag, page)
	if err != nil {
		c.writeToLog("showHistory", err)
		c.sendAnyError()
		return
	}

	for _, data := range list {
		fb, err := c.ftp.ReadFile(data.Receipt)
		if err != nil {
			c.writeToLog("showHistory/ReadFile", err)
			c.sendAnyError()
			return
		}
		doc := tgbotapi.FileBytes{
			Name:  data.Receipt,
			Bytes: fb,
		}

		document := tgbotapi.NewDocument(c.chatId, doc)
		document.Caption = fmt.Sprintf("Назначение: %s\nСумма: %.2f %s\nДата: %s", data.Purpose, data.Sum, currency, data.Date.Format(layoutDate))
		_ = c.Send(document)
	}

	switch count := len(list); count {
	case db.NumberEntriesPerPage:
		msg := tgbotapi.NewMessage(c.chatId, "Показать предыдущие?")

		var nextKeyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(c.buttons.NextPageHistory.Label, fmt.Sprintf("%s/%d", c.buttons.NextPageHistory.Command, page+1))),
		)

		msg.ReplyMarkup = &nextKeyboard
		_ = c.Send(msg)
	default:
		_ = c.Send(tgbotapi.NewMessage(c.chatId, "Больше списаний нет"))
	}

}

func (c *Chat) awaitingPayment(...string) {
	tag, err := c.db.GetTag(c.ctx, c.chatId)
	if err != nil {
		c.writeToLog("awaitingPayment/GetTag", err)
		c.sendAnyError()
	}

	openCollections, err := c.db.FindCashCollectionByStatus(c.ctx, tag, db.StatusCashCollectionOpen)
	if err != nil {
		c.writeToLog("awaitingPayment/FindCashCollectionByStatus", err)
		c.sendAnyError()
	}

	count := 0
	for _, collection := range openCollections {
		debtors, err := c.db.GetDebtorsByCollection(c.ctx, collection.ID)
		if err != nil {
			c.writeToLog("showListDebtors/GetDebtorsByCollection", err)
			c.sendAnyError()
			return
		}

		for _, debtor := range debtors {
			if debtor.ID == c.chatId {
				msg := tgbotapi.NewMessage(c.chatId, fmt.Sprintf("Назначение: %s\nСумма: %.2f %s", collection.Purpose, collection.Sum, currency))

				var paymentKeyboard = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData(c.buttons.Payment.Label, fmt.Sprintf("%s/%d", c.buttons.Payment.Command, collection.ID)),
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

func (c *Chat) setAdmin(...string) {
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
			tgbotapi.NewInlineKeyboardButtonData(c.buttons.SetAdminYes.Label, c.buttons.SetAdminYes.Command+"/"+strconv.FormatInt(members[number-1].ID, 10)),
			tgbotapi.NewInlineKeyboardButtonData(c.buttons.No.Label, c.buttons.No.Command),
		),
	)

	msg.ReplyMarkup = &yesNoKeyboard
	_ = c.Send(msg)
}

func (c *Chat) setAdminYes(args ...string) {
	id, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		c.writeToLog("setAdminYes/strconvParseInt", err)
		c.sendAnyError()
		return
	}

	tag, err := c.db.GetTag(c.ctx, c.chatId)
	if err != nil {
		c.writeToLog("setAdminYes/GetTag", err)
		c.sendAnyError()
		return
	}

	if ok, err := c.db.SetAdmin(c.ctx, tag, c.chatId, id); err != nil || !ok {
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
