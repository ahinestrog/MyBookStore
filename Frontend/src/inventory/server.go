package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	inventorypb "github.com/ahinestrog/mybookstore/proto/gen/inventory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type StockItem struct {
	BookID       int64 `json:"book_id"`
	AvailableQty int32 `json:"available_qty"`
}

func main() {
	mux := http.NewServeMux()

	// estáticos con y sin prefijo /inventory
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))
	mux.Handle("/inventory/static/", http.StripPrefix("/inventory/static/", fs))

	// API endpoint (server-side calls inventory gRPC)
	mux.HandleFunc("/api/inventory", handleAPIInventory)

	// Frontend
	mux.HandleFunc("/", handleInventoryPage)
	mux.HandleFunc("/inventory", handleInventoryPage)
	mux.HandleFunc("/inventory/", handleInventoryPage)

	addr := getenv("FRONTEND_INVENTORY_ADDR", ":8082")
	fmt.Printf("🌐 Inventory Frontend escuchando en %s\n", addr)
	http.ListenAndServe(addr, mux)
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func invClient(ctx context.Context) (inventorypb.InventoryClient, *grpc.ClientConn, error) {
	// Prefer INVENTORY_SERVICE_ADDR over INVENTORY_GRPC_ADDR
	target := os.Getenv("INVENTORY_SERVICE_ADDR")
	if target == "" {
		target = getenv("INVENTORY_GRPC_ADDR", "inventory:50052")
	}
	cc, err := grpc.DialContext(ctx, target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	return inventorypb.NewInventoryClient(cc), cc, nil
}

func handleInventoryPage(w http.ResponseWriter, r *http.Request) {
	tpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := map[string]any{
		"LoggedIn": cookieUID(r) != 0,
		"UserName": cookieUName(r),
		"Prefill":  r.URL.Query().Get("ids"),
	}
	_ = tpl.Execute(w, data)
}

func handleAPIInventory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	raw := strings.TrimSpace(r.URL.Query().Get("ids"))
	if raw == "" {
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []StockItem{}})
		return
	}
	parts := strings.Split(raw, ",")
	ids := make([]int64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if id, err := strconv.ParseInt(p, 10, 64); err == nil {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []StockItem{}})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	cli, cc, err := invClient(ctx)
	if err != nil {
		http.Error(w, "inventory grpc: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer cc.Close()
	resp, err := cli.GetAvailability(ctx, &inventorypb.GetAvailabilityRequest{BookIds: ids})
	if err != nil {
		http.Error(w, "inventory grpc: "+err.Error(), http.StatusBadGateway)
		return
	}
	items := make([]StockItem, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		items = append(items, StockItem{BookID: it.GetBookId(), AvailableQty: it.GetAvailableQty()})
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
}

// --- login helpers ---
func cookieUID(r *http.Request) int64 {
	if c, err := r.Cookie("uid"); err == nil {
		if id, err2 := strconv.ParseInt(c.Value, 10, 64); err2 == nil {
			return id
		}
	}
	return 0
}
func cookieUName(r *http.Request) string {
	if c, err := r.Cookie("uname"); err == nil {
		return c.Value
	}
	return ""
}
