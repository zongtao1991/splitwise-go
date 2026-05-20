package services

import (
	"sort"
	"splitwise-go/db"
	"splitwise-go/models"
	"splitwise-go/utils"
)

const minSettlementAmount = 0.01

func GetGroupBalances(groupID int64) ([]models.Balance, error) {
	members, err := getGroupMembers(groupID)
	if err != nil {
		return nil, err
	}

	balances := make(map[int64]float64)
	names := make(map[int64]string)
	for _, m := range members {
		balances[m.ID] = 0
		names[m.ID] = m.Nickname
	}

	rows, err := db.DB.Query(`
		SELECT payer_id, amount_in_default FROM expenses WHERE group_id = ?
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var payerID int64
		var amount float64
		if err := rows.Scan(&payerID, &amount); err != nil {
			return nil, err
		}
		balances[payerID] = utils.RoundToMoney(balances[payerID] + amount)
	}

	rows2, err := db.DB.Query(`
		SELECT es.member_id, es.amount FROM expense_splits es
		JOIN expenses e ON es.expense_id = e.id
		WHERE e.group_id = ?
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()

	for rows2.Next() {
		var memberID int64
		var amount float64
		if err := rows2.Scan(&memberID, &amount); err != nil {
			return nil, err
		}
		balances[memberID] = utils.RoundToMoney(balances[memberID] - amount)
	}

	rows3, err := db.DB.Query(`
		SELECT payer_id, payee_id, amount FROM settlements WHERE group_id = ?
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows3.Close()

	for rows3.Next() {
		var payerID, payeeID int64
		var amount float64
		if err := rows3.Scan(&payerID, &payeeID, &amount); err != nil {
			return nil, err
		}
		balances[payerID] = utils.RoundToMoney(balances[payerID] + amount)
		balances[payeeID] = utils.RoundToMoney(balances[payeeID] - amount)
	}

	var result []models.Balance
	for _, m := range members {
		result = append(result, models.Balance{
			MemberID:   m.ID,
			MemberName: m.Nickname,
			Amount:     utils.RoundToMoney(balances[m.ID]),
		})
	}

	return result, nil
}

func SuggestSettlements(groupID int64) ([]models.SettlementSuggestion, error) {
	balances, err := GetGroupBalances(groupID)
	if err != nil {
		return nil, err
	}

	var debtors, creditors []models.Balance
	for _, b := range balances {
		if utils.MoneyLessThan(b.Amount, -minSettlementAmount) {
			debtors = append(debtors, b)
		} else if utils.MoneyGreaterThan(b.Amount, minSettlementAmount) {
			creditors = append(creditors, b)
		}
	}

	if len(debtors) == 0 || len(creditors) == 0 {
		return []models.SettlementSuggestion{}, nil
	}

	suggestions := optimalSettlements(debtors, creditors)

	return suggestions, nil
}

func optimalSettlements(debtors, creditors []models.Balance) []models.SettlementSuggestion {
	if len(debtors) == 0 || len(creditors) == 0 {
		return []models.SettlementSuggestion{}
	}

	sort.Slice(debtors, func(i, j int) bool {
		return utils.MoneyLessThan(debtors[i].Amount, debtors[j].Amount)
	})
	sort.Slice(creditors, func(i, j int) bool {
		return utils.MoneyGreaterThan(creditors[i].Amount, creditors[j].Amount)
	})

	var suggestions []models.SettlementSuggestion
	i, j := 0, 0

	for i < len(debtors) && j < len(creditors) {
		if utils.MoneyGreaterOrEqual(debtors[i].Amount, -minSettlementAmount) {
			i++
			continue
		}
		if utils.MoneyLessOrEqual(creditors[j].Amount, minSettlementAmount) {
			j++
			continue
		}

		debt := -debtors[i].Amount
		credit := creditors[j].Amount

		var amount float64
		if utils.MoneyLessThan(debt, credit) {
			amount = debt
		} else {
			amount = credit
		}
		amount = utils.RoundToMoney(amount)

		if utils.MoneyGreaterThan(amount, 0) {
			suggestions = append(suggestions, models.SettlementSuggestion{
				From:   models.Member{ID: debtors[i].MemberID, Nickname: debtors[i].MemberName},
				To:     models.Member{ID: creditors[j].MemberID, Nickname: creditors[j].MemberName},
				Amount: amount,
			})
		}

		debtors[i].Amount = utils.RoundToMoney(debtors[i].Amount + amount)
		creditors[j].Amount = utils.RoundToMoney(creditors[j].Amount - amount)
	}

	return suggestions
}

func GetMemberBalance(groupID, memberID int64) (float64, error) {
	balances, err := GetGroupBalances(groupID)
	if err != nil {
		return 0, err
	}

	for _, b := range balances {
		if b.MemberID == memberID {
			return b.Amount, nil
		}
	}
	return 0, nil
}

type SettlementValidationError struct {
	Message       string
	PayerBalance  float64
	PayeeBalance  float64
	MaxAllowed    float64
}

func (e *SettlementValidationError) Error() string {
	return e.Message
}

func RecordSettlement(groupID, payerID, payeeID int64, amount float64) error {
	amount = utils.RoundToMoney(amount)
	if utils.MoneyLessOrEqual(amount, 0) {
		return &SettlementValidationError{
			Message: "结算金额必须大于0",
		}
	}

	if payerID == payeeID {
		return &SettlementValidationError{
			Message: "付款方和收款方不能是同一人",
		}
	}

	balances, err := GetGroupBalances(groupID)
	if err != nil {
		return err
	}

	var payerBalance, payeeBalance float64
	payerFound := false
	payeeFound := false

	for _, b := range balances {
		if b.MemberID == payerID {
			payerBalance = b.Amount
			payerFound = true
		}
		if b.MemberID == payeeID {
			payeeBalance = b.Amount
			payeeFound = true
		}
	}

	if !payerFound || !payeeFound {
		return &SettlementValidationError{
			Message: "付款方或收款方不在该分组中",
		}
	}

	if !utils.MoneyLessThan(payerBalance, 0) {
		return &SettlementValidationError{
			Message:      "付款方当前没有欠款，无需付款",
			PayerBalance: payerBalance,
			PayeeBalance: payeeBalance,
			MaxAllowed:   0,
		}
	}

	if !utils.MoneyGreaterThan(payeeBalance, 0) {
		return &SettlementValidationError{
			Message:      "收款方当前没有应收款",
			PayerBalance: payerBalance,
			PayeeBalance: payeeBalance,
			MaxAllowed:   0,
		}
	}

	maxPayerCanPay := -payerBalance
	maxPayeeCanReceive := payeeBalance
	maxAllowed := maxPayerCanPay
	if utils.MoneyLessThan(maxPayeeCanReceive, maxAllowed) {
		maxAllowed = maxPayeeCanReceive
	}

	if utils.MoneyGreaterThan(amount, maxAllowed) {
		return &SettlementValidationError{
			Message:      "结算金额超过最大可结算金额",
			PayerBalance: payerBalance,
			PayeeBalance: payeeBalance,
			MaxAllowed:   maxAllowed,
		}
	}

	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO settlements (group_id, payer_id, payee_id, amount) VALUES (?, ?, ?, ?)
	`, groupID, payerID, payeeID, amount)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func getGroupMembers(groupID int64) ([]models.Member, error) {
	rows, err := db.DB.Query(`
		SELECT m.id, m.nickname FROM members m
		JOIN group_members gm ON m.id = gm.member_id
		WHERE gm.group_id = ?
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []models.Member
	for rows.Next() {
		var m models.Member
		if err := rows.Scan(&m.ID, &m.Nickname); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, nil
}
