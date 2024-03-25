package service

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"project1/internal/config"
	"project1/internal/db"
	"project1/internal/ftp"
	"sync"
)

type Service struct {
	bot         *tgbotapi.BotAPI
	wg          *sync.RWMutex
	waitingList map[int64]chan *tgbotapi.Message
	DB          db.ConnString
	FTP         ftp.FTP
	Buttons     ButtonList
	Commands    CommandList
}

type Button struct {
	Label   string
	Command string
}

type ButtonList struct {
	CreateFound,
	Join,
	ShowBalance,
	AwaitingPayment,
	Leave,
	CreateCashCollection,
	CreateDebitingFunds,
	Members,
	Statistics,
	ConfirmationCreateFoundYes,
	ConfirmationCreateFoundNo,
	Payment,
	PaymentConfirmation,
	PaymentRefusal,
	PaymentExpected Button
}

type CommandList struct {
	ConfirmationCreateNewFund,
	Join,
	ShowBalance,
	CreateCashCollection,
	CreateDebitingFunds,
	GetMembers,
	CreateNewFund,
	Start,
	Payment,
	PaymentAccept,
	PaymentReject,
	PaymentWait,
	Menu,
	ShowListDebtors string
}

func NewService() (*Service, error) {
	var serv Service
	conf, err := config.NewConfig()
	if err != nil {
		return &serv, err
	}

	bot, err := tgbotapi.NewBotAPI(conf.Token)
	if err != nil {
		return &serv, err
	}
	bot.Debug = false

	cmds := CommandList{
		ConfirmationCreateNewFund: "confirmationCreateNewFund",
		Join:                      "join",
		ShowBalance:               "showBalance",
		CreateCashCollection:      "createCashCollection",
		CreateDebitingFunds:       "createDebitingFunds",
		GetMembers:                "getMembers",
		CreateNewFund:             "createNewFund",
		Start:                     "start",
		Payment:                   "payment",
		PaymentAccept:             "accept",
		PaymentReject:             "reject",
		PaymentWait:               "wait",
		Menu:                      "menu",
		ShowListDebtors:           "showListDebtors",
	}

	return &Service{
		bot:         bot,
		wg:          &sync.RWMutex{},
		waitingList: make(map[int64]chan *tgbotapi.Message),
		DB:          conf.DB,
		FTP:         conf.FTP,
		Buttons: ButtonList{
			CreateFound: Button{
				Label:   "Создать фонд",
				Command: cmds.ConfirmationCreateNewFund,
			},
			Join: Button{
				Label:   "Присоединиться",
				Command: cmds.Join,
			},
			ShowBalance: Button{
				Label:   "Баланс",
				Command: cmds.ShowBalance,
			},
			AwaitingPayment: Button{
				Label:   "Оплатить",
				Command: "1",
			}, // TODO реализовать
			Leave: Button{
				Label:   "Покинуть фонд",
				Command: "2",
			}, // TODO реализовать
			CreateCashCollection: Button{
				Label:   "Новый сбор",
				Command: cmds.CreateCashCollection,
			},
			CreateDebitingFunds: Button{
				Label:   "Новое списание",
				Command: cmds.CreateDebitingFunds,
			},
			Members: Button{
				Label:   "Участники",
				Command: cmds.GetMembers,
			},
			Statistics: Button{
				Label:   "Должники",
				Command: cmds.ShowListDebtors,
			},
			ConfirmationCreateFoundYes: Button{
				Label:   "Да",
				Command: cmds.CreateNewFund,
			},
			ConfirmationCreateFoundNo: Button{
				Label:   "Нет",
				Command: cmds.Start,
			},
			Payment: Button{
				Label:   "Оплатить",
				Command: cmds.Payment,
			},
			PaymentConfirmation: Button{
				Label:   "Подтвердить",
				Command: cmds.PaymentAccept,
			},
			PaymentRefusal: Button{
				Label:   "Отказ",
				Command: cmds.PaymentReject,
			},
			PaymentExpected: Button{
				Label:   "Ожидание",
				Command: cmds.PaymentWait,
			}},
		Commands: cmds,
	}, nil
}

func (s *Service) GetBot() *tgbotapi.BotAPI {
	return s.bot
}

func (s *Service) GetWaitingList() map[int64]chan *tgbotapi.Message {
	s.wg.RLock()
	tmp := s.waitingList
	s.wg.RUnlock()
	return tmp
}

func (s *Service) GetUserChan(id int64) (chan *tgbotapi.Message, bool) {
	userChan, ok := s.waitingList[id]
	return userChan, ok
}

func (s *Service) DeleteFromWaitingList(id int64) {
	s.wg.Lock()
	if ch, ok := s.waitingList[id]; ok {
		close(ch)
		delete(s.waitingList, id)
	}
	s.wg.Unlock()
}

func (s *Service) WaitingResponse(id int64) {
	s.wg.Lock()
	s.waitingList[id] = make(chan *tgbotapi.Message)
	s.wg.Unlock()
}

func (s *Service) StopWaiting(id int64) {
	s.wg.Lock()
	if _, ok := s.waitingList[id]; ok {
		close(s.waitingList[id])
		delete(s.waitingList, id)
	}
	s.wg.Unlock()
}
