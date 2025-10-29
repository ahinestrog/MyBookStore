package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	paymentpb "github.com/ahinestrog/mybookstore/proto/gen/payment"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config

type Config struct {
	Port            string // HTTP del frontend de payment
	PaymentGRPCAddr string // host:puerto del microservicio payment
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func loadConfig() Config {
	return Config{
		Port:            getenv("FRONTEND_PAYMENT_PORT", "8084"),
		PaymentGRPCAddr: getenv("PAYMENT_GRPC_ADDR", "payment:50053"),
	}
}

// Templates

func parseTemplates() (*template.Template, error) {
	t := template.New("base.html").Funcs(template.FuncMap{
		"since": func(unix int64) string {
			if unix == 0 {
				return "-"
			}
			return time.Unix(unix, 0).Format("2006-01-02 15:04:05")
		},
		"badgeClass": func(state string) string {
			switch state {
			case "PENDING":
				return "badge badge-pending"
			case "SUCCEEDED":
				return "badge badge-ok"
			case "FAILED":
				return "badge badge-fail"
			default:
				return "badge"
			}
		},
	})
	return t.ParseGlob("templates/*.html")
}

// Cliente gRPC

type PaymentClient struct {
	addr string
}

func NewPaymentClient(addr string) *PaymentClient { return &PaymentClient{addr: addr} }

func (c *PaymentClient) GetStatus(ctx context.Context, orderID int64) (*paymentpb.GetPaymentStatusResponse, error) {
	conn, err := grpc.DialContext(ctx, c.addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial payment: %w", err)
	}
	defer conn.Close()
	cl := paymentpb.NewPaymentClient(conn)
	return cl.GetPaymentStatus(ctx, &paymentpb.GetPaymentStatusRequest{OrderId: orderID})
}

func mapStateToString(s paymentpb.PaymentState) string {
	switch s {
	case paymentpb.PaymentState_PAYMENT_STATE_PENDING:
		return "PENDING"
	case paymentpb.PaymentState_PAYMENT_STATE_SUCCEEDED:
		return "SUCCEEDED"
	case paymentpb.PaymentState_PAYMENT_STATE_FAILED:
		return "FAILED"
	default:
		return "UNSPECIFIED"
	}
}

// HTTP server

type Server struct {
	cfg   Config
	tpl   *template.Template
	pgCli *PaymentClient
}

func NewServer(cfg Config) (*Server, error) {
	tpl, err := parseTemplates()
	if err != nil {
		return nil, err
	}
	return &Server{
		cfg:   cfg,
		tpl:   tpl,
		pgCli: NewPaymentClient(cfg.PaymentGRPCAddr),
	}, nil
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	// estáticos en /static/... y compatibilidad con /payment/static/... (para subdominio y prefijo)
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))
	mux.Handle("/payment/static/", http.StripPrefix("/payment/static/", fs))

	mux.HandleFunc("/", s.handleHome)
	// Soporta ambos paths: con prefijo (cuando se accede vía mybookstore.local/payment/...) y sin prefijo (cuando se usa subdominio)
	mux.HandleFunc("/payment/status", s.handlePaymentStatus)
	mux.HandleFunc("/status", s.handlePaymentStatus)

	return s.logRequests(mux)
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	// Redirigimos al path con prefijo para que el Ingress pueda enrutar correctamente bajo mybookstore.local/payment
	http.Redirect(w, r, "/payment/status", http.StatusFound)
}

func (s *Server) handlePaymentStatus(w http.ResponseWriter, r *http.Request) {
	type viewData struct {
		Title       string
		QueryOrder  string
		HasResult   bool
		OrderID     int64
		StateStr    string
		ProviderRef string
		UpdatedUnix int64
		ErrorMsg    string
		LoggedIn    bool
		UserName    string
	}
	data := viewData{Title: "Payment Status", LoggedIn: cookieUID(r) != 0, UserName: cookieUName(r)}

	if q := r.URL.Query().Get("order_id"); q != "" {
		data.QueryOrder = q
		var orderID int64
		if _, err := fmt.Sscan(q, &orderID); err != nil || orderID <= 0 {
			data.ErrorMsg = "order_id inválido (debe ser entero > 0)"
			s.render(w, "status.html", data)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		resp, err := s.pgCli.GetStatus(ctx, orderID)
		if err != nil {
			data.ErrorMsg = "No se pudo consultar el estado: " + err.Error()
		} else {
			data.HasResult = true
			data.OrderID = resp.GetOrderId()
			data.StateStr = mapStateToString(resp.GetState())
			data.ProviderRef = resp.GetProviderRef()
			data.UpdatedUnix = resp.GetUpdatedUnix()
		}
	}

	s.render(w, "status.html", data)
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("render error: %v", err)
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s) from %s", r.Method, r.URL.Path, time.Since(start), r.RemoteAddr)
	})
}

func main() {
	cfg := loadConfig()
	srv, err := NewServer(cfg)
	if err != nil {
		log.Fatalf("init: %v", err)
	}
	addr := ":" + cfg.Port
	log.Printf("[payment-frontend] http=%s payment_grpc=%s", addr, cfg.PaymentGRPCAddr)
	if err := http.ListenAndServe(addr, srv.routes()); err != nil {
		log.Fatal(err)
	}
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
