package service

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"sync"
)

type Service struct {
	bot         *tgbotapi.BotAPI
	wg          *sync.RWMutex
	waitingList map[int64]chan *tgbotapi.Message
}

func NewService(bot *tgbotapi.BotAPI) *Service {
	return &Service{
		bot:         bot,
		wg:          &sync.RWMutex{},
		waitingList: make(map[int64]chan *tgbotapi.Message),
	}
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
	close(s.waitingList[id])
	delete(s.waitingList, id)
	s.wg.Unlock()
}
