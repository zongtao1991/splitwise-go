package main

import (
	"fmt"
	"log"
	"net/http"
	"splitwise-go/db"
	"splitwise-go/handlers"
)

func main() {
	db.Init("./splitwise.db")

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("GET /api/members", handlers.ListMembers)
	mux.HandleFunc("POST /api/members", handlers.CreateMember)

	mux.HandleFunc("GET /api/groups", handlers.ListGroups)
	mux.HandleFunc("POST /api/groups", handlers.CreateGroup)
	mux.HandleFunc("GET /api/groups/{id}", handlers.GetGroup)
	mux.HandleFunc("POST /api/groups/{id}/members", handlers.AddGroupMember)
	mux.HandleFunc("DELETE /api/groups/{id}/members/{memberId}", handlers.RemoveGroupMember)

	mux.HandleFunc("GET /api/groups/{id}/expenses", handlers.ListExpenses)
	mux.HandleFunc("POST /api/groups/{id}/expenses", handlers.CreateExpense)
	mux.HandleFunc("DELETE /api/expenses/{id}", handlers.DeleteExpense)

	mux.HandleFunc("GET /api/groups/{id}/balances", handlers.GetBalances)
	mux.HandleFunc("GET /api/groups/{id}/settlements/suggest", handlers.SuggestSettlements)
	mux.HandleFunc("POST /api/groups/{id}/settlements", handlers.RecordSettlement)
	mux.HandleFunc("GET /api/groups/{id}/settlements", handlers.GetSettlements)

	mux.HandleFunc("GET /api/groups/{id}/stats", handlers.GetStats)

	// Static files
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Page routes - serve SPA
	mux.HandleFunc("GET /", serveIndex)
	mux.HandleFunc("GET /group/{id}", serveIndex)

	port := ":7912"
	fmt.Printf("SplitEase running on http://localhost%s\n", port)
	log.Fatal(http.ListenAndServe(port, mux))
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/index.html")
}
