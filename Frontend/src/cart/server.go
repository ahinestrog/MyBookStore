package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	cartpb "github.com/ahinestrog/mybookstore/proto/gen/cart"
	commonpb "github.com/ahinestrog/mybookstore/proto/gen/common"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Server struct {
	tpl    *template.Template
	client cartpb.CartClient
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

	cc, err := grpc.Dial(grpcTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial grpc %s: %v", grpcTarget, err)
	}
	defer cc.Close()

	tpl := template.Must(template.ParseGlob("./templates/*.html"))
	s := &Server{
		tpl:    tpl,
		client: cartpb.NewCartClient(cc),
	}

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/cart", s.handleCart)
	mux.HandleFunc("/add", s.handleAdd)
	mux.HandleFunc("/remove", s.handleRemove)
	mux.HandleFunc("/remove_line", s.handleRemoveLine)
	mux.HandleFunc("/clear", s.handleClear)

	log.Printf("[gateway] HTTP %s -> gRPC %s", httpAddr, grpcTarget)
	if err := http.ListenAndServe(httpAddr, mux); err != nil {
		log.Fatal(err)
	}
}

func (s *Server) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

// Para demo, fijamos el usuario. En producción, vendría de sesión/autenticación.
func (s *Server) userID(_ *http.Request) int64 { return 1 }

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	_ = s.tpl.ExecuteTemplate(w, "index.html", nil)
}

type MoneyView struct{ Cents int64 }

func (m MoneyView) String() string {
	// Muestra centavos como entero simple para mantener todo en cents en el demo
	return strconv.FormatInt(m.Cents, 10)
}

type ItemVM struct {
	BookID   int64
	Title    string
	Qty      int32
	Unit     MoneyView
	Line     MoneyView
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
	resp, err := s.client.GetCart(ctx, &commonpb.UserRef{UserId: s.userID(r)})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_ = s.tpl.ExecuteTemplate(w, "cart.html", toVM(resp, ""))
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
	cv, err := s.client.AddItem(ctx, &cartpb.AddItemRequest{
		UserId: s.userID(r),
		BookId: bookID,
		Qty:    int32(qty64),
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_ = s.tpl.ExecuteTemplate(w, "cart.html", toVM(cv, "Ítem agregado"))
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
	_ = s.tpl.ExecuteTemplate(w, "cart.html", toVM(cv, "Cantidad reducida"))
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
	_ = s.tpl.ExecuteTemplate(w, "cart.html", toVM(cv, "Ítem eliminado"))
}

func (s *Server) handleClear(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := s.ctx()
	defer cancel()
	cv, err := s.client.ClearCart(ctx, &commonpb.UserRef{UserId: s.userID(r)})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_ = s.tpl.ExecuteTemplate(w, "cart.html", toVM(cv, "Carrito vaciado"))
}
