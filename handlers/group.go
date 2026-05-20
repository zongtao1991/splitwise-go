package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"splitwise-go/db"
	"splitwise-go/models"
)

func ListGroups(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query("SELECT id, name, description, currency, created_at FROM groups ORDER BY created_at DESC")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var groups []models.Group
	for rows.Next() {
		var g models.Group
		rows.Scan(&g.ID, &g.Name, &g.Description, &g.Currency, &g.CreatedAt)
		groups = append(groups, g)
	}
	json.NewEncoder(w).Encode(groups)
}

func CreateGroup(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Currency    string `json:"currency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Name == "" {
		http.Error(w, "name required", 400)
		return
	}
	if input.Currency == "" {
		input.Currency = "CNY"
	}

	result, err := db.DB.Exec("INSERT INTO groups (name, description, currency) VALUES (?, ?, ?)",
		input.Name, input.Description, input.Currency)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	id, _ := result.LastInsertId()
	json.NewEncoder(w).Encode(models.Group{ID: id, Name: input.Name, Description: input.Description, Currency: input.Currency})
}

func GetGroup(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)

	var g models.Group
	err := db.DB.QueryRow("SELECT id, name, description, currency, created_at FROM groups WHERE id = ?", id).
		Scan(&g.ID, &g.Name, &g.Description, &g.Currency, &g.CreatedAt)
	if err != nil {
		http.Error(w, "group not found", 404)
		return
	}

	rows, _ := db.DB.Query(`
		SELECT m.id, m.nickname FROM members m
		JOIN group_members gm ON m.id = gm.member_id
		WHERE gm.group_id = ? ORDER BY m.nickname
	`, id)
	defer rows.Close()

	for rows.Next() {
		var m models.Member
		rows.Scan(&m.ID, &m.Nickname)
		g.Members = append(g.Members, m)
	}

	json.NewEncoder(w).Encode(g)
}

func AddGroupMember(w http.ResponseWriter, r *http.Request) {
	groupID, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)

	var input struct {
		MemberID int64 `json:"member_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "member_id required", 400)
		return
	}

	var groupExists bool
	err := db.DB.QueryRow("SELECT 1 FROM groups WHERE id = ?", groupID).Scan(&groupExists)
	if err != nil {
		http.Error(w, "group not found", 404)
		return
	}

	var memberExists bool
	err = db.DB.QueryRow("SELECT 1 FROM members WHERE id = ?", input.MemberID).Scan(&memberExists)
	if err != nil {
		http.Error(w, "member not found", 404)
		return
	}

	_, err = db.DB.Exec("INSERT OR IGNORE INTO group_members (group_id, member_id) VALUES (?, ?)",
		groupID, input.MemberID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(201)
}

func RemoveGroupMember(w http.ResponseWriter, r *http.Request) {
	groupID, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	memberID, _ := strconv.ParseInt(r.PathValue("memberId"), 10, 64)

	var hasExpenses bool
	err := db.DB.QueryRow(`
		SELECT 1 FROM expenses 
		WHERE group_id = ? AND payer_id = ? 
		LIMIT 1
	`, groupID, memberID).Scan(&hasExpenses)
	if err == nil && hasExpenses {
		http.Error(w, "cannot remove member with expenses in this group", 400)
		return
	}

	var hasSplits bool
	err = db.DB.QueryRow(`
		SELECT 1 FROM expense_splits es
		JOIN expenses e ON es.expense_id = e.id
		WHERE e.group_id = ? AND es.member_id = ?
		LIMIT 1
	`, groupID, memberID).Scan(&hasSplits)
	if err == nil && hasSplits {
		http.Error(w, "cannot remove member who is part of expense splits in this group", 400)
		return
	}

	result, err := db.DB.Exec("DELETE FROM group_members WHERE group_id = ? AND member_id = ?", groupID, memberID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "member not found in group", 404)
		return
	}

	w.WriteHeader(204)
}
