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
)

type Button struct {
	Label   string
	Command string
}

type List struct {
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

func NewButtonList() List {
	return List{
		CreateFound: Button{
			Label:   "Создать фонд",
			Command: CreateFund,
		},
		CreateFoundYes: Button{
			Label:   "Да",
			Command: CreateFundYes,
		},
		CreateFoundNo: Button{
			Label:   "Нет",
			Command: Start,
		},
		Join: Button{
			Label:   "Присоединиться",
			Command: Join,
		},
		ShowBalance: Button{
			Label:   "Баланс",
			Command: ShowBalance,
		},
		ShowTag: Button{
			Label:   "Тег",
			Command: ShowTag,
		},
		SetAdmin: Button{
			Label:   "Сменить администратора",
			Command: SetAdmin,
		},
		SetAdminYes: Button{
			Label:   "Да",
			Command: SetAdminYes,
		},
		History: Button{
			Label:   "История списаний",
			Command: History,
		},
		NextPageHistory: Button{
			Label:   "Далее",
			Command: History,
		},
		AwaitingPayment: Button{
			Label:   "Ожидает оплаты",
			Command: AwaitingPayment,
		},
		Leave: Button{
			Label:   "Покинуть фонд",
			Command: Leave,
		},
		LeaveYes: Button{
			Label:   "Да",
			Command: LeaveYes,
		},
		CreateCashCollection: Button{
			Label:   "Новый сбор",
			Command: CreateCashCollection,
		},
		CreateDebitingFunds: Button{
			Label:   "Новое списание",
			Command: CreateDebitingFunds,
		},
		Members: Button{
			Label:   "Участники",
			Command: Members,
		},
		DebtorList: Button{
			Label:   "Должники",
			Command: ShowListDebtors,
		},
		Payment: Button{
			Label:   "Оплатить",
			Command: Payment,
		},
		PaymentConfirmation: Button{
			Label:   "Подтвердить",
			Command: PaymentAccept,
		},
		PaymentRefusal: Button{
			Label:   "Отказ",
			Command: PaymentReject,
		},
		PaymentExpected: Button{
			Label:   "Ожидание",
			Command: PaymentWait,
		},
		DeleteMember: Button{
			Label:   "Удалить участника",
			Command: DeleteMember,
		},
		DeleteMemberYes: Button{
			Label:   "Да",
			Command: DeleteMemberYes,
		},
		No: Button{
			Label:   "Нет",
			Command: Menu,
		}}

}
