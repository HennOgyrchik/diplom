package service

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type Service struct {
	bot         *tgbotapi.BotAPI
	waitingList map[int64]chan *tgbotapi.Message
}

func NewService(bot *tgbotapi.BotAPI) *Service {
	return &Service{
		bot:         bot,
		waitingList: make(map[int64]chan *tgbotapi.Message),
	}
}

func (s *Service) GetBot() *tgbotapi.BotAPI {
	return s.bot
}

func (s *Service) GetWaitingList() map[int64]chan *tgbotapi.Message {
	return s.waitingList
}
