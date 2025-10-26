package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
)

type StockItem struct {
	BookID       int64 `json:"book_id"`
	AvailableQty int32 `json:"available_qty"`
}

// --- Mock temporal (luego conectas con gRPC Gateway real)
var mockInventory = map[int64]int32{
	1: 10,
	2: 5,
	3: 0,
	4: 12,
	5: 1,
}

func main() {
	mux := http.NewServeMux()

	// Archivos estáticos
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// API endpoint
	mux.HandleFunc("/api/inventory", handleAPIInventory)

	// Frontend - sirve la página tanto en / como en /inventory para mantener compatibilidad
	mux.HandleFunc("/", handleInventoryPage)
	mux.HandleFunc("/inventory", handleInventoryPage)

	fmt.Println("🌐 Inventory Frontend corriendo en http://localhost:8082/inventory")
	http.ListenAndServe(":8082", mux)
}

func handleInventoryPage(w http.ResponseWriter, r *http.Request) {
	tpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tpl.Execute(w, nil)
}

func handleAPIInventory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ids := strings.Split(r.URL.Query().Get("ids"), ",")
	var items []StockItem

	for _, idStr := range ids {
		idStr = strings.TrimSpace(idStr)
		if idStr == "" {
			continue
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			continue
		}
		qty, ok := mockInventory[id]
		if !ok {
			qty = 0
		}
		items = append(items, StockItem{BookID: id, AvailableQty: qty})
	}

	json.NewEncoder(w).Encode(map[string]any{"items": items})
}
