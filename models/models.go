package models

import "time"

type Member struct {
	ID        int64     `json:"id"`
	Nickname  string    `json:"nickname"`
	CreatedAt time.Time `json:"created_at"`
}

type Group struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Currency    string    `json:"currency"`
	CreatedAt   time.Time `json:"created_at"`
	Members     []Member  `json:"members,omitempty"`
}

type GroupMember struct {
	GroupID  int64     `json:"group_id"`
	MemberID int64     `json:"member_id"`
	JoinedAt time.Time `json:"joined_at"`
}

type Expense struct {
	ID              int64     `json:"id"`
	GroupID         int64     `json:"group_id"`
	PayerID         int64     `json:"payer_id"`
	PayerName       string    `json:"payer_name,omitempty"`
	Amount          float64   `json:"amount"`
	Currency        string    `json:"currency"`
	ExchangeRate    float64   `json:"exchange_rate"`
	AmountInDefault float64   `json:"amount_in_default"`
	Description     string    `json:"description"`
	SplitType       string    `json:"split_type"`
	ExpenseDate     string    `json:"expense_date"`
	CreatedAt       time.Time `json:"created_at"`
	Splits          []Split   `json:"splits,omitempty"`
}

type Split struct {
	ID         int64   `json:"id"`
	ExpenseID  int64   `json:"expense_id"`
	MemberID   int64   `json:"member_id"`
	MemberName string  `json:"member_name,omitempty"`
	Amount     float64 `json:"amount"`
	Percentage float64 `json:"percentage,omitempty"`
}

type Settlement struct {
	ID        int64     `json:"id"`
	GroupID   int64     `json:"group_id"`
	PayerID   int64     `json:"payer_id"`
	PayerName string    `json:"payer_name,omitempty"`
	PayeeID   int64     `json:"payee_id"`
	PayeeName string    `json:"payee_name,omitempty"`
	Amount    float64   `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}

type Balance struct {
	MemberID   int64   `json:"member_id"`
	MemberName string  `json:"member_name"`
	Amount     float64 `json:"amount"`
}

type SettlementSuggestion struct {
	From   Member `json:"from"`
	To     Member `json:"to"`
	Amount float64 `json:"amount"`
}

type Stats struct {
	TotalExpenses float64 `json:"total_expenses"`
	ExpenseCount  int     `json:"expense_count"`
	MemberStats   []MemberStat `json:"member_stats"`
}

type MemberStat struct {
	MemberID     int64   `json:"member_id"`
	MemberName   string  `json:"member_name"`
	TotalPaid    float64 `json:"total_paid"`
	TotalOwed    float64 `json:"total_owed"`
	NetBalance   float64 `json:"net_balance"`
}
