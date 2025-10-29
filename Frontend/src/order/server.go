package main

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	orderpb "github.com/ahinestrog/mybookstore/proto/gen/order"
	paymentpb "github.com/ahinestrog/mybookstore/proto/gen/payment"
)

//go:embed templates/*.html
var tplFS embed.FS

//go:embed static/*
var staticFS embed.FS

type app struct {
	addr        string
	orderAddr   string
	paymentAddr string
	tpls        *template.Template
	httpServer  *http.Server
}

func main() {
	a := &app{
		addr:        getEnv("FRONTEND_ADDR", ":8083"),
		orderAddr:   getEnv("ORDER_SVC_ADDR", "localhost:50056"),
		paymentAddr: getEnv("PAYMENT_SVC_ADDR", "localhost:50053"),
	}
	a.loadTemplates()
	mux := http.NewServeMux()

	// static
	mux.Handle("/static/", http.FileServer(http.FS(staticFS)))

	// routes
	mux.HandleFunc("/", a.handleIndex)
	mux.HandleFunc("/create", a.handleCreate)
	mux.HandleFunc("/status", a.handleStatus)

	// enlaces rápidos a otras vistas del frontend como el catalogo o el carrito de compras
	mux.HandleFunc("/catalog", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/catalog/", http.StatusFound) })
	mux.HandleFunc("/cart", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/cart/", http.StatusFound) })
	mux.HandleFunc("/inventory", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/inventory/", http.StatusFound) })

	a.httpServer = &http.Server{
		Addr:              a.addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("[order-frontend] http listening on %s (order svc: %s, payment svc: %s)", a.addr, a.orderAddr, a.paymentAddr)
	if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func (a *app) loadTemplates() {
	tpls := template.Must(template.ParseFS(tplFS, "templates/*.html"))
	a.tpls = tpls
}

func (a *app) dialOrder() (*grpc.ClientConn, orderpb.OrderClient, error) {
	cc, err := grpc.Dial(a.orderAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	return cc, orderpb.NewOrderClient(cc), nil
}

func (a *app) dialPayment() (*grpc.ClientConn, paymentpb.PaymentClient, error) {
	cc, err := grpc.Dial(a.paymentAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	return cc, paymentpb.NewPaymentClient(cc), nil
}

// Función render que ejecuta el layout y las plantillas
func render(w http.ResponseWriter, tpls *template.Template, layout, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Execute the layout template so that defined blocks (e.g. content/title)
	// inside other files (index.html/status.html) are rendered within the layout.
	// Previously this executed the inner 'name' template which only contained
	// definitions and produced an empty response.
	if err := tpls.ExecuteTemplate(w, layout, data); err != nil {
		// Log the error server-side and return an informative 500 body for quicker debugging
		log.Printf("template execute error: %v (layout=%s name=%s)", err, layout, name)
		http.Error(w, fmt.Sprintf("error renderizando %s: %v", name, err), http.StatusInternalServerError)
	}
}

// GET /
func (a *app) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data := map[string]any{
		"LoggedIn": cookieUID(r) != 0,
		"UserName": cookieUName(r),
	}
	render(w, a.tpls, "layout.html", "index.html", data)
}

// POST /create
func (a *app) handleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	r.ParseForm()
	uidStr := r.Form.Get("user_id")
	uid, err := strconv.ParseInt(uidStr, 10, 64)
	if err != nil || uid <= 0 {
		render(w, a.tpls, "layout.html", "index.html", map[string]any{"Error": "user_id inválido"})
		return
	}

	cc, client, err := a.dialOrder()
	if err != nil {
		render(w, a.tpls, "layout.html", "index.html", map[string]any{"Error": "No se pudo conectar al servicio de órdenes"})
		return
	}
	defer cc.Close()

	ctx, cancel := timeoutCtx(r.Context(), 5*time.Second)
	defer cancel()

	resp, err := client.CreateOrder(ctx, &orderpb.CreateOrderRequest{UserId: uid})
	if err != nil {
		render(w, a.tpls, "layout.html", "index.html", map[string]any{"Error": fmt.Sprintf("CreateOrder falló: %v", err)})
		return
	}

	// Adaptamos respuesta para la tabla
	type row struct {
		Title      string
		Qty        int32
		Unit, Line string
	}
	rows := make([]row, 0, len(resp.Items))
	for _, it := range resp.Items {
		rows = append(rows, row{
			Title: it.Title,
			Qty:   it.Qty,
			Unit:  centsToStr(it.UnitPrice.GetCents()),
			Line:  centsToStr(it.LineTotal.GetCents()),
		})
	}

	data := map[string]any{
		"Created": true,
		"OrderID": resp.GetOrderId(),
		"Status":  orderStatusToText(resp.GetStatus()),
		"Total":   centsToStr(resp.GetTotal().GetCents()),
		"Items":   rows,
	}
	render(w, a.tpls, "layout.html", "index.html", data)
}

// GET /status
func (a *app) handleStatus(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("id")
	data := map[string]any{"QueryOrderID": q, "LoggedIn": cookieUID(r) != 0, "UserName": cookieUName(r)}
	if q == "" {
		render(w, a.tpls, "layout.html", "status.html", data)
		return
	}
	oid, err := strconv.ParseInt(q, 10, 64)
	if err != nil || oid <= 0 {
		data["Error"] = "order_id inválido"
		render(w, a.tpls, "layout.html", "status.html", data)
		return
	}

	cc, client, err := a.dialOrder()
	if err != nil {
		data["Error"] = "No se pudo conectar al servicio de órdenes"
		render(w, a.tpls, "layout.html", "status.html", data)
		return
	}
	defer cc.Close()

	ctx, cancel := timeoutCtx(r.Context(), 4*time.Second)
	defer cancel()

	resp, err := client.GetOrderStatus(ctx, &orderpb.GetOrderStatusRequest{OrderId: oid})
	if err != nil {
		data["Error"] = fmt.Sprintf("GetOrderStatus falló: %v", err)
		render(w, a.tpls, "layout.html", "status.html", data)
		return
	}

	data["Found"] = true
	data["OrderID"] = resp.GetOrderId()
	data["Status"] = orderStatusToText(resp.GetStatus())
	data["Total"] = centsToStr(resp.GetTotal().GetCents())
	data["UpdatedUnix"] = resp.GetUpdatedUnix()

	// Check if this is a newly created order (coming from checkout)
	// If the order status is CREATED and there's no 'from' query param, assume it's just created
	if resp.GetStatus() == orderpb.OrderStatus_ORDER_STATUS_CREATED && r.URL.Query().Get("from") == "" {
		data["JustCreated"] = true
	}

	// Get payment status
	pcc, pclient, err := a.dialPayment()
	if err != nil {
		log.Printf("[order] failed to dial payment service: %v", err)
		// Continue without payment info
	} else {
		defer pcc.Close()
		pctx, pcancel := timeoutCtx(r.Context(), 3*time.Second)
		defer pcancel()

		presp, err := pclient.GetPaymentStatus(pctx, &paymentpb.GetPaymentStatusRequest{OrderId: oid})
		if err != nil {
			log.Printf("[order] GetPaymentStatus failed: %v", err)
		} else {
			data["PaymentState"] = paymentStateToText(presp.GetState())
			data["ProviderRef"] = presp.GetProviderRef()
			data["PaymentPending"] = presp.GetState() == paymentpb.PaymentState_PAYMENT_STATE_PENDING
		}
	}

	render(w, a.tpls, "layout.html", "status.html", data)
}

func orderStatusToText(s orderpb.OrderStatus) string {
	switch s {
	case orderpb.OrderStatus_ORDER_STATUS_CREATED:
		return "CREATED"
	case orderpb.OrderStatus_ORDER_STATUS_PAID:
		return "PAID"
	case orderpb.OrderStatus_ORDER_STATUS_CANCELLED:
		return "CANCELLED"
	case orderpb.OrderStatus_ORDER_STATUS_FAILED:
		return "FAILED"
	default:
		return "UNSPECIFIED"
	}
}

func paymentStateToText(s paymentpb.PaymentState) string {
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

func centsToStr(c int64) string {
	sign := ""
	if c < 0 {
		sign = "-"
		c = -c
	}
	d := c / 100
	ct := c % 100
	return fmt.Sprintf("%s$%d.%02d", sign, d, ct)
}

func timeoutCtx(parent context.Context, d time.Duration) (context.Context, func()) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, d)
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
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
