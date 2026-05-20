package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"splitwise-go/db"
	"splitwise-go/models"
	"splitwise-go/utils"
)

func GetStats(w http.ResponseWriter, r *http.Request) {
	groupID, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)

	var stats models.Stats

	db.DB.QueryRow("SELECT COALESCE(SUM(amount_in_default), 0), COUNT(*) FROM expenses WHERE group_id = ?", groupID).
		Scan(&stats.TotalExpenses, &stats.ExpenseCount)
	stats.TotalExpenses = utils.RoundToMoney(stats.TotalExpenses)

	rows, err := db.DB.Query(`
		SELECT m.id, m.nickname,
		       COALESCE(SUM(CASE WHEN e.payer_id = m.id THEN e.amount_in_default ELSE 0 END), 0) as total_paid,
		       COALESCE((SELECT SUM(es.amount) FROM expense_splits es JOIN expenses e2 ON es.expense_id = e2.id WHERE es.member_id = m.id AND e2.group_id = ?), 0) as total_owed
		FROM members m
		JOIN group_members gm ON m.id = gm.member_id
		LEFT JOIN expenses e ON e.group_id = gm.group_id
		WHERE gm.group_id = ?
		GROUP BY m.id
	`, groupID, groupID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var ms models.MemberStat
		rows.Scan(&ms.MemberID, &ms.MemberName, &ms.TotalPaid, &ms.TotalOwed)
		ms.TotalPaid = utils.RoundToMoney(ms.TotalPaid)
		ms.TotalOwed = utils.RoundToMoney(ms.TotalOwed)
		ms.NetBalance = utils.RoundToMoney(ms.TotalPaid - ms.TotalOwed)
		stats.MemberStats = append(stats.MemberStats, ms)
	}

	json.NewEncoder(w).Encode(stats)
}
