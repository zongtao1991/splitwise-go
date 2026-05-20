package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"splitwise-go/db"
	"splitwise-go/models"
)

func ListMembers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query("SELECT id, nickname, created_at FROM members ORDER BY nickname")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var members []models.Member
	for rows.Next() {
		var m models.Member
		rows.Scan(&m.ID, &m.Nickname, &m.CreatedAt)
		members = append(members, m)
	}
	json.NewEncoder(w).Encode(members)
}

func CreateMember(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Nickname string `json:"nickname"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Nickname == "" {
		http.Error(w, "nickname required", 400)
		return
	}

	result, err := db.DB.Exec("INSERT INTO members (nickname) VALUES (?)", input.Nickname)
	if err != nil {
		http.Error(w, "nickname already exists", 400)
		return
	}

	id, _ := result.LastInsertId()
	json.NewEncoder(w).Encode(models.Member{ID: id, Nickname: input.Nickname})
}

func GetMemberID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}
