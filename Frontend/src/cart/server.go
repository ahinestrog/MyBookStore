package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	neturl "net/url"
	"os"
	"strconv"
	"time"

	cartpb "github.com/ahinestrog/mybookstore/proto/gen/cart"
	commonpb "github.com/ahinestrog/mybookstore/proto/gen/common"
	inventorypb "github.com/ahinestrog/mybookstore/proto/gen/inventory"
	orderpb "github.com/ahinestrog/mybookstore/proto/gen/order"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Server struct {
	tpl         *template.Template
	client      cartpb.CartClient
	orderClient orderpb.OrderClient
	orderAddr   string
	invClient   inventorypb.InventoryClient
	invAddr     string
}

func mustEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	httpAddr := mustEnv("GATEWAY_HTTP_ADDR", ":8080")
	grpcTarget := mustEnv("CART_GRPC_TARGET", "localhost:50051")
	orderAddr := mustEnv("ORDER_GRPC_TARGET", "localhost:50054")

	// Prefer INVENTORY_SERVICE_ADDR, fallback to INVENTORY_GRPC_TARGET
	invAddr := os.Getenv("INVENTORY_SERVICE_ADDR")
	if invAddr == "" {
		invAddr = mustEnv("INVENTORY_GRPC_TARGET", "inventory:50052")
	}

	cc, err := grpc.Dial(grpcTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial grpc %s: %v", grpcTarget, err)
	}
	defer cc.Close()

	orderCC, err := grpc.Dial(orderAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial order grpc %s: %v", orderAddr, err)
	}
	defer orderCC.Close()

	invCC, err := grpc.Dial(invAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial inventory grpc %s: %v", invAddr, err)
	}
	defer invCC.Close()

	tpl := template.Must(template.ParseGlob("./templates/*.html"))
	s := &Server{
		tpl:         tpl,
		client:      cartpb.NewCartClient(cc),
		orderClient: orderpb.NewOrderClient(orderCC),
		orderAddr:   orderAddr,
		invClient:   inventorypb.NewInventoryClient(invCC),
		invAddr:     invAddr,
	}

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/cart", s.handleCart)
	mux.HandleFunc("/add", s.handleAdd)
	mux.HandleFunc("/remove", s.handleRemove)
	mux.HandleFunc("/remove_line", s.handleRemoveLine)
	mux.HandleFunc("/clear", s.handleClear)
	mux.HandleFunc("/checkout", s.handleCheckout)

	log.Printf("[gateway] HTTP %s -> gRPC %s, Order %s, Inventory %s", httpAddr, grpcTarget, orderAddr, invAddr)
	if err := http.ListenAndServe(httpAddr, mux); err != nil {
		log.Fatal(err)
	}
}

func (s *Server) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

// Para demo, fijamos el usuario. En producción, vendría de sesión/autenticación.
func (s *Server) userID(r *http.Request) int64 {
	if c, err := r.Cookie("uid"); err == nil {
		if id, err2 := strconv.ParseInt(c.Value, 10, 64); err2 == nil {
			return id
		}
	}
	return 0
}

func (s *Server) userName(r *http.Request) string {
	if c, err := r.Cookie("uname"); err == nil {
		return c.Value
	}
	return ""
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Serve the cart view at the root so that accessing /cart/ (rewritten to /)
	// shows the current cart. This is necessary when the Ingress rewrites
	// prefixed paths (e.g. /cart/) to / on the backend.
	s.handleCart(w, r)
}

type MoneyView struct{ Cents int64 }

func (m MoneyView) String() string {
	// Muestra centavos como entero simple para mantener todo en cents en el demo
	return strconv.FormatInt(m.Cents, 10)
}

type ItemVM struct {
	BookID int64
	Title  string
	Qty    int32
	Unit   MoneyView
	Line   MoneyView
}

type CartVM struct {
	Items []ItemVM
	Total MoneyView
	Msg   string
}

func toVM(cv *cartpb.CartView, msg string) CartVM {
	vm := CartVM{Msg: msg}
	for _, it := range cv.GetItems() {
		vm.Items = append(vm.Items, ItemVM{
			BookID: it.GetBookId(),
			Title:  it.GetTitle(),
			Qty:    it.GetQty(),
			Unit:   MoneyView{Cents: it.GetUnitPrice().GetCents()},
			Line:   MoneyView{Cents: it.GetLineTotal().GetCents()},
		})
	}
	vm.Total = MoneyView{Cents: cv.GetTotal().GetCents()}
	return vm
}

func (s *Server) handleCart(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := s.ctx()
	defer cancel()
	log.Printf("handleCart: user=%d", s.userID(r))
	resp, err := s.client.GetCart(ctx, &commonpb.UserRef{UserId: s.userID(r)})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	log.Printf("handleCart: got %d items", len(resp.GetItems()))
	msg := r.URL.Query().Get("msg")
	logged := s.userID(r) != 0
	s.renderCart(w, resp, msg, logged, s.userName(r))
}

func (s *Server) handleAdd(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	bookID, _ := strconv.ParseInt(r.Form.Get("book_id"), 10, 64)
	qty64, _ := strconv.ParseInt(r.Form.Get("qty"), 10, 32)
	if qty64 <= 0 {
		qty64 = 1
	}

	ctx, cancel := s.ctx()
	defer cancel()
	log.Printf("handleAdd: user=%d book=%d qty=%d", s.userID(r), bookID, qty64)
	cv, err := s.client.AddItem(ctx, &cartpb.AddItemRequest{
		UserId: s.userID(r),
		BookId: bookID,
		Qty:    int32(qty64),
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	log.Printf("handleAdd: cart now %d items", len(cv.GetItems()))
	http.Redirect(w, r, "/cart/?msg=Item%20agregado", http.StatusSeeOther)
}

func (s *Server) handleRemove(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	bookID, _ := strconv.ParseInt(r.Form.Get("book_id"), 10, 64)
	qty64, _ := strconv.ParseInt(r.Form.Get("qty"), 10, 32)
	if qty64 <= 0 {
		qty64 = 1 // por defecto, decrementa 1
	}

	ctx, cancel := s.ctx()
	defer cancel()
	cv, err := s.client.RemoveItem(ctx, &cartpb.RemoveItemRequest{
		UserId: s.userID(r),
		BookId: bookID,
		Qty:    int32(qty64),
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_ = cv
	http.Redirect(w, r, "/cart/?msg=Cantidad%20reducida", http.StatusSeeOther)
}

func (s *Server) handleRemoveLine(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	bookID, _ := strconv.ParseInt(r.Form.Get("book_id"), 10, 64)

	ctx, cancel := s.ctx()
	defer cancel()
	cv, err := s.client.RemoveItem(ctx, &cartpb.RemoveItemRequest{
		UserId: s.userID(r),
		BookId: bookID,
		Qty:    0, // regla de tu proto: 0 u omitido elimina la línea
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_ = cv
	http.Redirect(w, r, "/cart/?msg=Ítem%20eliminado", http.StatusSeeOther)
}

func (s *Server) handleClear(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := s.ctx()
	defer cancel()
	cv, err := s.client.ClearCart(ctx, &commonpb.UserRef{UserId: s.userID(r)})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_ = cv
	http.Redirect(w, r, "/cart/?msg=Carrito%20vaciado", http.StatusSeeOther)
}

func (s *Server) handleCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/cart/", http.StatusSeeOther)
		return
	}

	// Require login
	uid := s.userID(r)
	if uid == 0 {
		http.Redirect(w, r, "/user/login?from=/cart/", http.StatusSeeOther)
		return
	}

	ctx, cancel := s.ctx()
	defer cancel()

	// Validate inventory before creating order
	// Fetch current cart to get items and quantities
	cv, err := s.client.GetCart(ctx, &commonpb.UserRef{UserId: uid})
	if err != nil {
		log.Printf("[checkout] GetCart failed: %v", err)
		http.Redirect(w, r, "/cart/?msg=No%20se%20pudo%20obtener%20carrito", http.StatusSeeOther)
		return
	}
	if len(cv.GetItems()) == 0 {
		http.Redirect(w, r, "/cart/?msg=Carrito%20vac%C3%ADo", http.StatusSeeOther)
		return
	}
	// Build list of IDs
	ids := make([]int64, 0, len(cv.GetItems()))
	qtyByID := make(map[int64]int32, len(cv.GetItems()))
	titleByID := make(map[int64]string, len(cv.GetItems()))
	for _, it := range cv.GetItems() {
		ids = append(ids, it.GetBookId())
		qtyByID[it.GetBookId()] = it.GetQty()
		titleByID[it.GetBookId()] = it.GetTitle()
	}
	// Query inventory availability
	invResp, err := s.invClient.GetAvailability(ctx, &inventorypb.GetAvailabilityRequest{BookIds: ids})
	if err != nil {
		log.Printf("[checkout] inventory GetAvailability failed: %v", err)
		http.Redirect(w, r, "/cart/?msg=No%20se%20pudo%20validar%20inventario", http.StatusSeeOther)
		return
	}
	avail := map[int64]int32{}
	for _, it := range invResp.GetItems() {
		avail[it.GetBookId()] = it.GetAvailableQty()
	}
	// Check each item
	for id, need := range qtyByID {
		have := avail[id]
		if have < need {
			title := titleByID[id]
			msg := fmt.Sprintf("Sin%20stock:%20%s%20(id:%d)%20req:%d%20disp:%d", neturl.QueryEscape(title), id, need, have)
			http.Redirect(w, r, "/cart/?msg="+msg, http.StatusSeeOther)
			return
		}
	}

	// 1. Create order via order service (inventory prevalidated)
	resp, err := s.orderClient.CreateOrder(ctx, &orderpb.CreateOrderRequest{UserId: uid})
	if err != nil {
		log.Printf("[checkout] CreateOrder failed: %v", err)
		http.Redirect(w, r, "/cart/?msg=Error%20al%20crear%20orden", http.StatusSeeOther)
		return
	}

	// 2. Clear cart after successful order creation
	if _, err := s.client.ClearCart(ctx, &commonpb.UserRef{UserId: uid}); err != nil {
		log.Printf("[checkout] ClearCart failed (order already created): %v", err)
		// Don't fail checkout if cart clear fails
	}

	// 3. Redirect to order status page (PRG pattern)
	orderID := resp.GetOrderId()
	log.Printf("[checkout] Order created successfully: order_id=%d, user_id=%d, total=%d",
		orderID, uid, resp.GetTotal().GetCents())
	http.Redirect(w, r, fmt.Sprintf("/order/status?id=%d", orderID), http.StatusSeeOther)
}

// renderCart ejecuta el layout principal para que los bloques definidos en cart.html se inserten
// y adapta los datos al shape que esperan las plantillas (Cart, Msg y helper FormatCOP).
func (s *Server) renderCart(w http.ResponseWriter, cv *cartpb.CartView, msg string, loggedIn bool, userName string) {
	vm := toVM(cv, msg)
	data := struct {
		Items     []ItemVM
		Total     MoneyView
		Msg       string
		FormatCOP func(int64) string
		Query     string
		Year      int
		LoggedIn  bool
		UserName  string
	}{
		Items: vm.Items,
		Total: vm.Total,
		Msg:   vm.Msg,
		FormatCOP: func(cents int64) string {
			pesos := cents / 100
			// formato sencillo con separadores de miles
			s := strconv.FormatInt(pesos, 10)
			out := ""
			for i, r := range s {
				if i != 0 && (len(s)-i)%3 == 0 {
					out += "."
				}
				out += string(r)
			}
			return fmt.Sprintf("$ %s", out)
		},
		Query:    "",
		Year:     time.Now().Year(),
		LoggedIn: loggedIn,
		UserName: userName,
	}
	// Parse only layout and cart templates to ensure the cart content block is the one used
	tpl, err := template.ParseFiles("./templates/layout.html", "./templates/cart.html")
	if err != nil {
		log.Printf("template parse error: %v", err)
		http.Error(w, "template error", 500)
		return
	}
	if err := tpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		log.Printf("template render error: %v", err)
		http.Error(w, "template error", 500)
	}
}
