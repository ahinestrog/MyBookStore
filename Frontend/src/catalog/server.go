package main

import (
	"context"
	"embed"
	"html/template"
	"strings"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	catalogpb "github.com/ahinestrog/mybookstore/proto/gen/catalog"
	commonpb  "github.com/ahinestrog/mybookstore/proto/gen/common"
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
}

func main() {
	port := getenv("PORT", "8080")
	grpcAddr := getenv("CATALOG_GRPC_ADDR", "localhost:50051")

	// gRPC client
	conn, err := grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial catalog grpc (%s): %v", grpcAddr, err)
	}
	defer conn.Close()
	client := catalogpb.NewCatalogClient(conn)

	// Parse templates
	funcs := template.FuncMap{
		"add": func(a, b int32) int32 { return a + b },
		"split": func(s, sep string) []string { return strings.Split(s, sep) },
		"year": func() int { return time.Now().Year() },
	}

	tplLayout := template.Must(template.New("layout.html").Funcs(funcs).ParseFS(templatesFS, "templates/layout.html"))
	tplList   := template.Must(template.Must(tplLayout.Clone()).ParseFS(templatesFS, "templates/index.html"))
	tplBook   := template.Must(template.Must(tplLayout.Clone()).ParseFS(templatesFS, "templates/book.html"))

	s := &Server{tplList: tplList, tplBook: tplBook, client: client}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/book", s.handleBook)

	// static
	mux.Handle("/static/", http.FileServer(http.FS(staticFS)))

	log.Printf("Catalog Frontend listening on :%s (Catalog gRPC → %s)", port, grpcAddr)
	if err := http.ListenAndServe(":"+port, withLog(mux)); err != nil {
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
	if page < 1 { page = 1 }
	size := atoiDefault(r.URL.Query().Get("page_size"), 12)
	if size <= 0 { size = 12 }

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
					finalCover = "/static" + cover
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
		Query      string
		Items      []listItem
		Page       *commonpb.PageResponse
		PageURL    func(int) string
	}{
		Query: q,
		Items: items,
		Page:  resp.GetPage(),
		PageURL: func(p int) string {
			qs := r.URL.Query()
			qs.Set("page", strconv.Itoa(p))
			qs.Set("page_size", strconv.Itoa(size))
			return "/?"+qs.Encode()
		},
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

	data := struct {
		Query string
		Book *catalogpb.Book
		// helper para formatear dinero: cents → "12.345,67" o "12,345.67" según preferencia
		FormatCOP func(int64) string
	}{
		Query: r.URL.Query().Get("q"),
		Book: b,
		FormatCOP: func(cents int64) string {
			// muy simple: pesos enteros
			pesos := cents / 100
			return "$ " + formatThousands(pesos)
		},
	}

	if err := s.tplBook.Execute(w, data); err != nil {
		log.Printf("template execute book error: %v", err)
		httpError(w, "Error renderizando página", http.StatusInternalServerError)
	}
}

// --- utils ---

func getenv(k, d string) string { if v := os.Getenv(k); v != "" { return v }; return d }

func atoiDefault(s string, d int) int {
	if s == "" { return d }
	i, err := strconv.Atoi(s)
	if err != nil { return d }
	return i
}

func atoi64Default(s string, d int64) int64 {
	if s == "" { return d }
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil { return d }
	return i
}

func httpError(w http.ResponseWriter, msg string, code int) {
	w.WriteHeader(code)
	_, _ = w.Write([]byte("<pre>"+template.HTMLEscapeString(msg)+"</pre>"))
}

func formatThousands(n int64) string {
	s := strconv.FormatInt(n, 10)
	neg := false
	if n < 0 { neg = true; s = s[1:] }
	out := ""
	for i, r := range s {
		if i != 0 && (len(s)-i)%3 == 0 {
			out += "."
		}
		out += string(r)
	}
	if neg { out = "-" + out }
	return out
}
