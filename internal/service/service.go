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

	return &Service{
		bot:         bot,
		wg:          &sync.RWMutex{},
		waitingList: make(map[int64]chan *tgbotapi.Message),
		DB:          conf.DB,
		FTP:         conf.FTP,
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
