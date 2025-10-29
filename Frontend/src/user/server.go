package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	commonpb "github.com/ahinestrog/mybookstore/proto/gen/common"
	userpb "github.com/ahinestrog/mybookstore/proto/gen/user"
	"google.golang.org/grpc"
)

var (
	tpl *template.Template
)

func main() {
	// Parse templates with a container path fallback to local dev path
	if t, err := template.ParseGlob("/srv/templates/*.html"); err == nil {
		tpl = t
	} else {
		// local dev fallback
		if tt, err2 := template.ParseGlob(filepath.FromSlash("./templates/*.html")); err2 == nil {
			tpl = tt
		} else {
			log.Fatalf("template parse failed: %v / %v", err, err2)
		}
	}

	// allow overriding when running locally: prefer localhost by default
	grpcAddr := env("USER_GRPC_ADDR", "localhost:50055")

	conn, err := grpc.Dial(grpcAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	client := userpb.NewUserClient(conn)

	// Serve static assets from container path or fallback to local path in dev
	if _, err := os.Stat("/srv/static"); err == nil {
		http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("/srv/static"))))
	} else {
		http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"LoggedIn": cookieUID(r) != 0,
			"UserName": cookieUName(r),
		}
		_ = tpl.ExecuteTemplate(w, "home.html", data)
	})

	// Support trailing slash paths from ingress rewrites
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
			// Set simple session cookies (demo): uid and uname (path=/ so all apps can read)
			http.SetCookie(w, &http.Cookie{Name: "uid", Value: strconv.FormatInt(resp.GetUserId(), 10), Path: "/", MaxAge: 7 * 24 * 3600, HttpOnly: false})
			http.SetCookie(w, &http.Cookie{Name: "uname", Value: name, Path: "/", MaxAge: 7 * 24 * 3600, HttpOnly: false})
			http.Redirect(w, r, "/user/profile?user_id="+strconv.FormatInt(resp.GetUserId(), 10), http.StatusSeeOther)
		}
	})
	http.HandleFunc("/register/", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = "/register"
		http.DefaultServeMux.ServeHTTP(w, r)
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
			// Fetch profile to get display name
			prof, perr := client.GetProfile(r.Context(), &commonpb.UserRef{UserId: resp.GetUserId()})
			if perr != nil {
				prof = &userpb.UserProfile{}
			}
			// Set cookies for session
			http.SetCookie(w, &http.Cookie{Name: "uid", Value: strconv.FormatInt(resp.GetUserId(), 10), Path: "/", MaxAge: 7 * 24 * 3600, HttpOnly: false})
			http.SetCookie(w, &http.Cookie{Name: "uname", Value: prof.GetName(), Path: "/", MaxAge: 7 * 24 * 3600, HttpOnly: false})
			// Support redirect back if provided
			next := r.URL.Query().Get("from")
			if next == "" {
				next = "/user/profile?user_id=" + strconv.FormatInt(resp.GetUserId(), 10)
			}
			http.Redirect(w, r, next, http.StatusSeeOther)
		}
	})
	http.HandleFunc("/login/", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = "/login"
		http.DefaultServeMux.ServeHTTP(w, r)
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
	http.HandleFunc("/profile/", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = "/profile"
		http.DefaultServeMux.ServeHTTP(w, r)
	})

	// Simple logout: clear cookies and redirect to user home
	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "uid", Value: "", Path: "/", MaxAge: -1})
		http.SetCookie(w, &http.Cookie{Name: "uname", Value: "", Path: "/", MaxAge: -1})
		http.Redirect(w, r, "/user/", http.StatusSeeOther)
	})
	http.HandleFunc("/logout/", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = "/logout"
		http.DefaultServeMux.ServeHTTP(w, r)
	})

	// HTTP_ADDR is expected in the form ":8080"; for convenience also allow PORT
	addr := env("HTTP_ADDR", "")
	if addr == "" {
		if p := os.Getenv("PORT"); p != "" {
			addr = ":" + p
		} else {
			addr = ":8080"
		}
	}
	log.Printf("[frontend-user] listening on %s (gRPC at %s)", addr, grpcAddr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func env(k, def string) string {
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
