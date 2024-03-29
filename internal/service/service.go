package service

import (
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"project1/internal/db"
	"project1/internal/env"
	"project1/internal/fileStorage"
	"sync"
)

type Service struct {
	bot         *tgbotapi.BotAPI
	wg          *sync.RWMutex
	waitingList map[int64]chan *tgbotapi.Message
	Buttons     ButtonList
	Commands    CommandList
	DB          *db.Repository
	FTP         *fileStorage.FileStorage
	Ctx         context.Context
}

type Button struct {
	Label   string
	Command string
}

type ButtonList struct {
	CreateFound, CreateFoundYes, CreateFoundNo,
	Join,
	ShowBalance,
	AwaitingPayment,
	CreateCashCollection,
	CreateDebitingFunds,
	Members,
	DebtorList,
	Payment, PaymentConfirmation, PaymentRefusal, PaymentExpected,
	DeleteMember, DeleteMemberYes,
	Leave, LeaveYes,
	ShowTag,
	History, NextPageHistory,
	SetAdmin, SetAdminYes,
	OpenCC, ClosedCC,
	No Button
}

type CommandList struct {
	CreateFund, CreateFundYes,
	Join,
	ShowBalance,
	CreateCashCollection,
	CreateDebitingFunds,
	GetMembers,
	CreateNewFund,
	Start,
	Payment, PaymentAccept, PaymentReject, PaymentWait,
	Menu,
	ShowListDebtors,
	DeleteMember, DeleteMemberYes,
	Leave, LeaveYes,
	ShowTag,
	History,
	AwaitingPayment,
	OpenCC, ClosedCC,
	SetAdmin, SetAdminYes string
}

func NewService(ctx context.Context) (*Service, error) {
	e, err := env.Setup(ctx)
	if err != nil {
		log.Fatal("setup.Setup: ", err)
	}

	bot, err := tgbotapi.NewBotAPI(e.Token)
	if err != nil {
		return nil, err
	}
	bot.Debug = false

	cmds := CommandList{
		CreateFund:           "createFund",
		CreateFundYes:        "createFundYes",
		Join:                 "join",
		ShowBalance:          "showBalance",
		CreateCashCollection: "createCashCollection",
		CreateDebitingFunds:  "createDebitingFunds",
		GetMembers:           "getMembers",
		Start:                "start",
		Payment:              "payment",
		PaymentAccept:        "accept",
		PaymentReject:        "reject",
		PaymentWait:          "wait",
		Menu:                 "menu",
		ShowListDebtors:      "showListDebtors",
		DeleteMember:         "deleteMember",
		DeleteMemberYes:      "deleteMemberYes",
		Leave:                "leave",
		LeaveYes:             "leaveYes",
		ShowTag:              "showTag",
		History:              "history",
		AwaitingPayment:      "awaitingPayment",
		SetAdmin:             "setAdmin",
		SetAdminYes:          "setAdminYes",
		OpenCC:               "openCC",
		ClosedCC:             "closedCC",
	}

	return &Service{
		bot:         bot,
		wg:          &sync.RWMutex{},
		waitingList: make(map[int64]chan *tgbotapi.Message),
		DB:          e.DB,
		FTP:         e.FTP,
		Buttons: ButtonList{
			CreateFound: Button{
				Label:   "Создать фонд",
				Command: cmds.CreateFund,
			},
			CreateFoundYes: Button{
				Label:   "Да",
				Command: cmds.CreateFundYes,
			},
			CreateFoundNo: Button{
				Label:   "Нет",
				Command: cmds.Start,
			},
			Join: Button{
				Label:   "Присоединиться",
				Command: cmds.Join,
			},
			ShowBalance: Button{
				Label:   "Баланс",
				Command: cmds.ShowBalance,
			},
			ShowTag: Button{
				Label:   "Тег",
				Command: cmds.ShowTag,
			},
			SetAdmin: Button{
				Label:   "Сменить администратора",
				Command: cmds.SetAdmin,
			},
			SetAdminYes: Button{
				Label:   "Да",
				Command: cmds.SetAdminYes,
			},
			History: Button{
				Label:   "История списаний",
				Command: cmds.History,
			},
			NextPageHistory: Button{
				Label:   "Далее",
				Command: cmds.History,
			},
			AwaitingPayment: Button{
				Label:   "Ожидает оплаты",
				Command: cmds.AwaitingPayment,
			},
			Leave: Button{
				Label:   "Покинуть фонд",
				Command: cmds.Leave,
			},
			LeaveYes: Button{
				Label:   "Да",
				Command: cmds.LeaveYes,
			},
			OpenCC: Button{
				Label:   "Открытые сборы",
				Command: cmds.OpenCC,
			},
			ClosedCC: Button{
				Label:   "Закрытые сборы",
				Command: cmds.ClosedCC,
			},
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
			DebtorList: Button{
				Label:   "Должники",
				Command: cmds.ShowListDebtors,
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
			},
			DeleteMember: Button{
				Label:   "Удалить участника",
				Command: cmds.DeleteMember,
			},
			DeleteMemberYes: Button{
				Label:   "Да",
				Command: cmds.DeleteMemberYes,
			},
			No: Button{
				Label:   "Нет",
				Command: cmds.Menu,
			}},
		Commands: cmds,
		Ctx:      ctx,
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
