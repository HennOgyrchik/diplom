package button

const (
	CreateFund           = "createFund"
	CreateFundYes        = "createFundYes"
	Join                 = "join"
	ShowBalance          = "showBalance"
	CreateCashCollection = "createCashCollection"
	CreateDebitingFunds  = "createDebitingFunds"
	Members              = "members"
	Start                = "start"
	Payment              = "payment"
	PaymentAccept        = "accept"
	PaymentReject        = "reject"
	PaymentWait          = "wait"
	Menu                 = "menu"
	ShowListDebtors      = "showListDebtors"
	DeleteMember         = "deleteMember"
	DeleteMemberYes      = "deleteMemberYes"
	Leave                = "leave"
	LeaveYes             = "leaveYes"
	ShowTag              = "showTag"
	History              = "history"
	AwaitingPayment      = "awaitingPayment"
	SetAdmin             = "setAdmin"
	SetAdminYes          = "setAdminYes"
	AwaitingConfirmation = "awaitingConfirmation"
)

type List struct {
	CreateFund,
	Join,
	ShowBalance,
	AwaitingPayment, AwaitingConfirmation,
	CreateCashCollection,
	CreateDebitingFunds,
	Members,
	DebtorList,
	Payment, PaymentConfirmation, PaymentRefusal, PaymentExpected,
	DeleteMember,
	Leave,
	ShowTag,
	History, NextPageHistory,
	SetAdmin,
	No, Yes string
}

func NewButtonList() List {
	return List{
		Yes:                  "Да",
		No:                   "Нет",
		CreateFund:           "Создать фонд",
		Join:                 "Присоединиться",
		ShowBalance:          "Баланс",
		ShowTag:              "Тег",
		SetAdmin:             "Сменить администратора",
		History:              "История списаний",
		NextPageHistory:      "Далее",
		AwaitingPayment:      "Ожидает оплаты",
		AwaitingConfirmation: "Подтверждение оплаты",
		Leave:                "Покинуть фонд",
		CreateCashCollection: "Новый сбор",
		CreateDebitingFunds:  "Новое списание",
		Members:              "Участники",
		DebtorList:           "Должники",
		Payment:              "Оплатить",
		PaymentConfirmation:  "Подтвердить",
		PaymentRefusal:       "Отказ",
		PaymentExpected:      "Ожидание",
		DeleteMember:         "Удалить участника",
	}
}
