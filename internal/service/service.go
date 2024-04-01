package service

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"project1/internal/button"
	"project1/internal/chat"
	"project1/internal/db"
	"project1/internal/env"
	"project1/internal/fileStorage"
	"sync"
)

type Service struct {
	bot         *tgbotapi.BotAPI
	wg          *sync.RWMutex
	waitingList map[int64]chan *tgbotapi.Message
	Buttons     button.List
	db          *db.Repository
	ftp         *fileStorage.FileStorage
	ctx         context.Context
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

	return &Service{
		bot:         bot,
		wg:          &sync.RWMutex{},
		waitingList: make(map[int64]chan *tgbotapi.Message),
		db:          e.DB,
		ftp:         e.FTP,
		Buttons:     button.NewButtonList(),
		ctx:         ctx,
	}, nil
}

func (s *Service) Run() {
	fmt.Printf("Authorized on account %s\n", s.bot.Self.UserName)
	log.Println("Authorized on account ", s.bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := s.bot.GetUpdatesChan(u)

	for update := range updates {
		go s.handlerUpdate(update)
	}
}

func (s *Service) handlerUpdate(update tgbotapi.Update) {
	var command string
	var message *tgbotapi.Message

	switch {
	case update.Message != nil:
		message = update.Message
		command = update.Message.Command()
	case update.CallbackQuery != nil:
		message = update.CallbackQuery.Message
		command = update.CallbackQuery.Data
	default:
		return
	}

	ch := chat.NewChat(s.ctx, update.FromChat().UserName, message.Chat.ID, s.bot, s.db, s.ftp, s.Buttons, s.waitingList, s.wg)

	if userChan, ok := s.waitingList[message.Chat.ID]; ok { //есть ли функции ожидающие ответа от пользователя?

		if !ch.CommandRouter(command) { //функция ждет ответ, проверь ответ это команда? Если это так, то она запустится
			userChan <- message //ответ не команда, отправь полученное сообщение в канал
			return
		}
		userChan <- nil
		return
	}

	if !ch.CommandRouter(command) {
		_ = ch.Send(tgbotapi.NewMessage(message.Chat.ID, "Я не знаю такую команду"))
	}

}

func (s *Service) Stop() error {
	err := s.ftp.Close()
	s.db.Close()

	return err
}
