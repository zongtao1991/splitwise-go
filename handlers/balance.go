package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"splitwise-go/db"
	"splitwise-go/services"
	"splitwise-go/utils"
)

func GetBalances(w http.ResponseWriter, r *http.Request) {
	groupID, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)

	balances, err := services.GetGroupBalances(groupID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(balances)
}

func SuggestSettlements(w http.ResponseWriter, r *http.Request) {
	groupID, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)

	suggestions, err := services.SuggestSettlements(groupID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(suggestions)
}

func RecordSettlement(w http.ResponseWriter, r *http.Request) {
	groupID, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)

	var input struct {
		PayerID int64   `json:"payer_id"`
		PayeeID int64   `json:"payee_id"`
		Amount  float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid input", 400)
		return
	}

	if input.PayerID == 0 || input.PayeeID == 0 || utils.MoneyLessOrEqual(input.Amount, 0) {
		http.Error(w, "payer_id, payee_id, and valid amount required", 400)
		return
	}

	err := services.RecordSettlement(groupID, input.PayerID, input.PayeeID, input.Amount)
	if err != nil {
		var ve *services.SettlementValidationError
		if errors.As(err, &ve) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":         ve.Message,
				"payer_balance": ve.PayerBalance,
				"payee_balance": ve.PayeeBalance,
				"max_allowed":   ve.MaxAllowed,
			})
			return
		}
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(201)
}

func GetSettlements(w http.ResponseWriter, r *http.Request) {
	groupID, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)

	rows, err := db.DB.Query(`
		SELECT s.id, s.group_id, s.payer_id, mp.nickname, s.payee_id, mee.nickname,
		       s.amount, s.created_at
		FROM settlements s
		JOIN members mp ON s.payer_id = mp.id
		JOIN members mee ON s.payee_id = mee.id
		WHERE s.group_id = ?
		ORDER BY s.created_at DESC
	`, groupID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	type SettlementView struct {
		ID        int64   `json:"id"`
		GroupID   int64   `json:"group_id"`
		PayerID   int64   `json:"payer_id"`
		PayerName string  `json:"payer_name"`
		PayeeID   int64   `json:"payee_id"`
		PayeeName string  `json:"payee_name"`
		Amount    float64 `json:"amount"`
		CreatedAt string  `json:"created_at"`
	}

	var settlements []SettlementView
	for rows.Next() {
		var s SettlementView
		rows.Scan(&s.ID, &s.GroupID, &s.PayerID, &s.PayerName, &s.PayeeID, &s.PayeeName, &s.Amount, &s.CreatedAt)
		settlements = append(settlements, s)
	}
	json.NewEncoder(w).Encode(settlements)
}
