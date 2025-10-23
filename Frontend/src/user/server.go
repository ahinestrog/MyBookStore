package main

import (
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/ahinestrog/mybookstore/proto/gen/common"
	"github.com/ahinestrog/mybookstore/proto/gen/user"
	"google.golang.org/grpc"
)

var (
	tpl *template.Template
)

func main() {
	tpl = template.Must(template.ParseGlob("templates/*.html"))

	grpcAddr := env("USER_GRPC_ADDR", "user:50055")

	conn, err := grpc.Dial(grpcAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	client := userpb.NewUserClient(conn)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_ = tpl.ExecuteTemplate(w, "home.html", nil)
	})

	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = tpl.ExecuteTemplate(w, "register.html", nil)
		case http.MethodPost:
			name := r.FormValue("name")
			email := r.FormValue("email")
			pass := r.FormValue("password")
			resp, err := client.Register(r.Context(), &userpb.RegisterRequest{
				Name:     name,
				Email:    email,
				Password: pass,
			})
			if err != nil {
				_ = tpl.ExecuteTemplate(w, "register.html", map[string]any{"Error": err.Error()})
				return
			}
			http.Redirect(w, r, "/profile?user_id="+strconv.FormatInt(resp.GetUserId(), 10), http.StatusSeeOther)
		}
	})

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = tpl.ExecuteTemplate(w, "login.html", nil)
		case http.MethodPost:
			email := r.FormValue("email")
			pass := r.FormValue("password")
			resp, err := client.Authenticate(r.Context(), &userpb.AuthenticateRequest{
				Email:    email,
				Password: pass,
			})
			if err != nil || !resp.GetOk() {
				_ = tpl.ExecuteTemplate(w, "login.html", map[string]any{"Error": "Credenciales inválidas"})
				return
			}
			http.Redirect(w, r, "/profile?user_id="+strconv.FormatInt(resp.GetUserId(), 10), http.StatusSeeOther)
		}
	})

	http.HandleFunc("/profile", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("user_id")
		id, _ := strconv.ParseInt(idStr, 10, 64)
		prof, err := client.GetProfile(r.Context(), &commonpb.UserRef{UserId: id})
		if err != nil {
			http.Error(w, "no se pudo obtener perfil: "+err.Error(), 500)
			return
		}
		_ = tpl.ExecuteTemplate(w, "profile.html", prof)
	})

	addr := env("HTTP_ADDR", ":8080")
	log.Printf("[frontend-user] listening on %s (gRPC at %s)", addr, grpcAddr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func env(k, def string) string {
	if v := getenv(k); v != "" {
		return v
	}
	return def
}

func getenv(k string) string { return func() string { return "" }() } // placeholder so we don't import os in this tiny file
