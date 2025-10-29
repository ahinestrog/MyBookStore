package main

import (
	"context"
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	catalogpb "github.com/ahinestrog/mybookstore/proto/gen/catalog"
	commonpb "github.com/ahinestrog/mybookstore/proto/gen/common"
	inventorypb "github.com/ahinestrog/mybookstore/proto/gen/inventory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

type Server struct {
	tplList *template.Template
	tplBook *template.Template
	client  catalogpb.CatalogClient
	invCli  inventorypb.InventoryClient
}

func main() {
	addr := getenv("FRONTEND_CATALOG_ADDR", ":8081")
	grpcAddr := getenv("CATALOG_GRPC_ADDR", "localhost:50051")

	// gRPC client
	conn, err := grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial catalog grpc (%s): %v", grpcAddr, err)
	}
	defer conn.Close()
	client := catalogpb.NewCatalogClient(conn)

	// Inventory gRPC (for availability on book page)
	// Prefer INVENTORY_SERVICE_ADDR over INVENTORY_GRPC_ADDR
	invAddr := os.Getenv("INVENTORY_SERVICE_ADDR")
	if invAddr == "" {
		invAddr = getenv("INVENTORY_GRPC_ADDR", "inventory:50052")
	}
	invConn, err := grpc.Dial(invAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial inventory grpc (%s): %v", invAddr, err)
	}
	defer invConn.Close()
	invClient := inventorypb.NewInventoryClient(invConn)

	// Parse templates
	funcs := template.FuncMap{
		"add":   func(a, b int32) int32 { return a + b },
		"split": func(s, sep string) []string { return strings.Split(s, sep) },
		"year":  func() int { return time.Now().Year() },
	}

	tplLayout := template.Must(template.New("layout.html").Funcs(funcs).ParseFS(templatesFS, "templates/layout.html"))
	tplList := template.Must(template.Must(tplLayout.Clone()).ParseFS(templatesFS, "templates/index.html"))
	tplBook := template.Must(template.Must(tplLayout.Clone()).ParseFS(templatesFS, "templates/book.html"))

	s := &Server{tplList: tplList, tplBook: tplBook, client: client, invCli: invClient}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/book", s.handleBook)

	// static
	mux.Handle("/static/", http.FileServer(http.FS(staticFS)))

	log.Printf("Catalog Frontend listening on %s (Catalog gRPC → %s, Inventory gRPC → %s)", addr, grpcAddr, invAddr)
	if err := http.ListenAndServe(addr, withLog(mux)); err != nil {
		log.Fatal(err)
	}
}

func withLog(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		h.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	q := r.URL.Query().Get("q")
	page := atoiDefault(r.URL.Query().Get("page"), 1)
	if page < 1 {
		page = 1
	}
	size := atoiDefault(r.URL.Query().Get("page_size"), 12)
	if size <= 0 {
		size = 12
	}

	resp, err := s.client.ListBooks(ctx, &catalogpb.ListBooksRequest{
		Q: q,
		Page: &commonpb.PageRequest{
			Page:     int32(page),
			PageSize: int32(size),
		},
	})
	if err != nil {
		httpError(w, "No se pudo obtener el catálogo: "+err.Error(), http.StatusBadGateway)
		return
	}

	type listItem struct {
		Id       int64
		Title    string
		Author   string
		CoverUrl string
		PriceStr string
	}

	items := make([]listItem, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		// Some older proto objects may have nil Price; guard against nil
		priceCents := int64(0)
		if it.GetPrice() != nil {
			priceCents = it.GetPrice().GetCents()
		}
		// determine cover url: prefer embedded /static/... if present, else fallback to placeholder
		cover := it.GetCoverUrl()
		finalCover := cover
		if cover != "" {
			if cover[0] == '/' {
				// check if embedded static contains the file under "static" + cover
				tryPath := "static" + cover
				if f, err := staticFS.Open(tryPath); err == nil {
					f.Close()
					finalCover = "static" + cover
				} else {
					// fallback to a remote placeholder if image not embedded
					finalCover = "https://via.placeholder.com/200x300?text=Cover"
				}
			}
		} else {
			finalCover = "https://via.placeholder.com/200x300?text=Cover"
		}

		items = append(items, listItem{
			Id: it.GetId(), Title: it.GetTitle(), Author: it.GetAuthor(),
			CoverUrl: finalCover, PriceStr: "$ " + formatThousands(priceCents/100),
		})
	}

	data := struct {
		Query    string
		Items    []listItem
		Page     *commonpb.PageResponse
		PageURL  func(int) string
		LoggedIn bool
		UserName string
	}{
		Query: q,
		Items: items,
		Page:  resp.GetPage(),
		PageURL: func(p int) string {
			qs := r.URL.Query()
			qs.Set("page", strconv.Itoa(p))
			qs.Set("page_size", strconv.Itoa(size))
			// Use relative URL to keep current path prefix (e.g., /catalog)
			return "?" + qs.Encode()
		},
		LoggedIn: cookieUID(r) != 0,
		UserName: cookieUName(r),
	}

	if err := s.tplList.Execute(w, data); err != nil {
		log.Printf("template execute list error: %v", err)
		httpError(w, "Error renderizando página", http.StatusInternalServerError)
	}
}

func (s *Server) handleBook(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	id := atoi64Default(r.URL.Query().Get("id"), 0)
	if id <= 0 {
		httpError(w, "id inválido", http.StatusBadRequest)
		return
	}
	b, err := s.client.GetBook(ctx, &catalogpb.GetBookRequest{Id: id})
	if err != nil {
		httpError(w, "No se pudo obtener el libro: "+err.Error(), http.StatusBadGateway)
		return
	}

	finalCover := b.GetCoverUrl()
	if finalCover == "" {
		finalCover = "https://via.placeholder.com/200x300?text=Cover"
	} else if finalCover[0] == '/' {
		tryPath := "static" + finalCover
		if f, err := staticFS.Open(tryPath); err == nil {
			f.Close()
			finalCover = "/static" + finalCover
		} else {
			finalCover = "https://via.placeholder.com/200x300?text=Cover"
		}
	}
	b.CoverUrl = finalCover

	// Fetch availability from inventory
	avail := int32(-1)
	if s.invCli != nil {
		ictx, icancel := context.WithTimeout(ctx, 2*time.Second)
		defer icancel()
		if a, err := s.invCli.GetAvailability(ictx, &inventorypb.GetAvailabilityRequest{BookIds: []int64{id}}); err == nil {
			if len(a.GetItems()) > 0 {
				avail = a.GetItems()[0].GetAvailableQty()
			}
		}
	}

	data := struct {
		Query string
		Book  *catalogpb.Book
		// helper para formatear dinero: cents → "12.345,67" o "12,345.67" según preferencia
		FormatCOP func(int64) string
		LoggedIn  bool
		UserName  string
		Available int32
	}{
		Query: r.URL.Query().Get("q"),
		Book:  b,
		FormatCOP: func(cents int64) string {
			// muy simple: pesos enteros
			pesos := cents / 100
			return "$ " + formatThousands(pesos)
		},
		LoggedIn:  cookieUID(r) != 0,
		UserName:  cookieUName(r),
		Available: avail,
	}

	if err := s.tplBook.Execute(w, data); err != nil {
		log.Printf("template execute book error: %v", err)
		httpError(w, "Error renderizando página", http.StatusInternalServerError)
	}
}

// --- utils ---

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func atoiDefault(s string, d int) int {
	if s == "" {
		return d
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return d
	}
	return i
}

func atoi64Default(s string, d int64) int64 {
	if s == "" {
		return d
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return d
	}
	return i
}

func httpError(w http.ResponseWriter, msg string, code int) {
	w.WriteHeader(code)
	_, _ = w.Write([]byte("<pre>" + template.HTMLEscapeString(msg) + "</pre>"))
}

func formatThousands(n int64) string {
	s := strconv.FormatInt(n, 10)
	neg := false
	if n < 0 {
		neg = true
		s = s[1:]
	}
	out := ""
	for i, r := range s {
		if i != 0 && (len(s)-i)%3 == 0 {
			out += "."
		}
		out += string(r)
	}
	if neg {
		out = "-" + out
	}
	return out
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
