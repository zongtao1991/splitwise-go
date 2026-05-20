package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"splitwise-go/db"
	"splitwise-go/models"
	"splitwise-go/utils"
)

func ListExpenses(w http.ResponseWriter, r *http.Request) {
	groupID, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit := 20
	offset := (page - 1) * limit

	rows, err := db.DB.Query(`
		SELECT e.id, e.group_id, e.payer_id, m.nickname, e.amount, e.currency,
		       e.exchange_rate, e.amount_in_default, e.description, e.split_type,
		       e.expense_date, e.created_at
		FROM expenses e
		JOIN members m ON e.payer_id = m.id
		WHERE e.group_id = ?
		ORDER BY e.expense_date DESC, e.id DESC
		LIMIT ? OFFSET ?
	`, groupID, limit, offset)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var expenses []models.Expense
	for rows.Next() {
		var e models.Expense
		rows.Scan(&e.ID, &e.GroupID, &e.PayerID, &e.PayerName, &e.Amount,
			&e.Currency, &e.ExchangeRate, &e.AmountInDefault, &e.Description,
			&e.SplitType, &e.ExpenseDate, &e.CreatedAt)

		splitRows, _ := db.DB.Query(`
			SELECT es.id, es.member_id, m.nickname, es.amount, es.percentage
			FROM expense_splits es
			JOIN members m ON es.member_id = m.id
			WHERE es.expense_id = ?
		`, e.ID)
		for splitRows.Next() {
			var s models.Split
			splitRows.Scan(&s.ID, &s.MemberID, &s.MemberName, &s.Amount, &s.Percentage)
			e.Splits = append(e.Splits, s)
		}
		splitRows.Close()

		expenses = append(expenses, e)
	}
	json.NewEncoder(w).Encode(expenses)
}

func CreateExpense(w http.ResponseWriter, r *http.Request) {
	groupID, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)

	var input struct {
		PayerID      int64   `json:"payer_id"`
		Amount       float64 `json:"amount"`
		Currency     string  `json:"currency"`
		ExchangeRate float64 `json:"exchange_rate"`
		Description  string  `json:"description"`
		SplitType    string  `json:"split_type"`
		ExpenseDate  string  `json:"expense_date"`
		Splits       []struct {
			MemberID   int64   `json:"member_id"`
			Amount     float64 `json:"amount"`
			Percentage float64 `json:"percentage"`
		} `json:"splits"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid input", 400)
		return
	}

	if utils.MoneyLessOrEqual(input.Amount, 0) || input.PayerID == 0 || len(input.Splits) == 0 {
		http.Error(w, "amount, payer_id, and splits required", 400)
		return
	}

	var groupCurrency string
	err := db.DB.QueryRow("SELECT currency FROM groups WHERE id = ?", groupID).Scan(&groupCurrency)
	if err != nil {
		http.Error(w, "group not found", 404)
		return
	}

	if input.Currency == "" {
		input.Currency = groupCurrency
	}
	if utils.MoneyLessOrEqual(input.ExchangeRate, 0) {
		input.ExchangeRate = 1.0
	}
	amountInDefault := utils.RoundToMoney(input.Amount * input.ExchangeRate)

	if input.SplitType == "percentage" {
		var totalPercent float64
		for _, s := range input.Splits {
			if utils.MoneyLessThan(s.Percentage, 0) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(400)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":          "百分比不能为负数",
					"split_type":     "percentage",
					"total_percent":  totalPercent,
					"expected_total": 100.0,
				})
				return
			}
			totalPercent = utils.RoundToMoney(totalPercent + s.Percentage)
		}
		if !utils.MoneyEqual(totalPercent, 100.0) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":          "百分比总和必须等于 100%",
				"split_type":     "percentage",
				"total_percent":  totalPercent,
				"expected_total": 100.0,
				"diff":           utils.RoundToMoney(100.0 - totalPercent),
			})
			return
		}
	} else if input.SplitType == "exact" {
		var totalAmount float64
		for _, s := range input.Splits {
			if utils.MoneyLessThan(s.Amount, 0) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(400)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":         "分摊金额不能为负数",
					"split_type":    "exact",
					"total_amount":  totalAmount,
					"expected_total": input.Amount,
				})
				return
			}
			totalAmount = utils.RoundToMoney(totalAmount + s.Amount)
		}
		if !utils.MoneyEqual(totalAmount, input.Amount) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":          "分摊金额总和必须等于支出总额",
				"split_type":     "exact",
				"total_amount":   totalAmount,
				"expected_total": input.Amount,
				"diff":           utils.RoundToMoney(input.Amount - totalAmount),
			})
			return
		}
	}

	tx, err := db.DB.Begin()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer tx.Rollback()

	result, err := tx.Exec(`
		INSERT INTO expenses (group_id, payer_id, amount, currency, exchange_rate,
		                      amount_in_default, description, split_type, expense_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, groupID, input.PayerID, input.Amount, input.Currency, input.ExchangeRate,
		amountInDefault, input.Description, input.SplitType, input.ExpenseDate)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	expenseID, _ := result.LastInsertId()

	var splitAmounts []float64
	if input.SplitType == "equal" {
		splitAmounts = utils.SplitAmountEqually(amountInDefault, len(input.Splits))
	} else if input.SplitType == "percentage" {
		percentages := make([]float64, len(input.Splits))
		for i, s := range input.Splits {
			percentages[i] = s.Percentage
		}
		splitAmounts = utils.SplitAmountByPercentages(amountInDefault, percentages)
	} else {
		splitAmounts = make([]float64, len(input.Splits))
		for i, s := range input.Splits {
			splitAmounts[i] = utils.RoundToMoney(s.Amount * input.ExchangeRate)
		}
	}

	for i, s := range input.Splits {
		splitAmount := splitAmounts[i]
		_, err = tx.Exec(`
			INSERT INTO expense_splits (expense_id, member_id, amount, percentage)
			VALUES (?, ?, ?, ?)
		`, expenseID, s.MemberID, splitAmount, s.Percentage)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	json.NewEncoder(w).Encode(map[string]int64{"id": expenseID})
}

func DeleteExpense(w http.ResponseWriter, r *http.Request) {
	expenseID, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)

	var exists bool
	err := db.DB.QueryRow("SELECT 1 FROM expenses WHERE id = ?", expenseID).Scan(&exists)
	if err != nil {
		http.Error(w, "expense not found", 404)
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM expense_splits WHERE expense_id = ?", expenseID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	_, err = tx.Exec("DELETE FROM expenses WHERE id = ?", expenseID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(204)
}
