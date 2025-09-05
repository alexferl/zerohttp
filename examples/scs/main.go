package main

import (
	"encoding/gob"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
	zh "github.com/alexferl/zerohttp"
)

var (
	sessionManager *scs.SessionManager
	templates      *template.Template
)

func init() {
	gob.Register(time.Time{})
}

func main() {
	// Setup session manager with cookie configuration
	sessionManager = scs.New()
	sessionManager.Lifetime = 24 * time.Hour
	sessionManager.Cookie.Name = "session_id"
	sessionManager.Cookie.HttpOnly = true
	sessionManager.Cookie.Path = "/"
	sessionManager.Cookie.SameSite = http.SameSiteLaxMode
	sessionManager.Cookie.Secure = false // Set to true for HTTPS
	// Note: SCS uses memstore by default (in-memory), but cookies for session tokens

	templates = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
<head><title>{{.Title}}</title></head>
<body>
    <h1>{{.Title}}</h1>
    {{if .Message}}<p style="color: {{if .Error}}red{{else}}green{{end}};">{{.Message}}</p>{{end}}
    {{if .User}}
        <p>Welcome, {{.User}}! <a href="/logout">Logout</a></p>
    {{else}}
        <a href="/login">Login</a>
    {{end}}
    <hr>
    {{.Content}}
</body>
</html>
    `))

	app := zh.New()

	// Add SCS middleware
	app.Use(sessionMiddleware)

	app.GET("/", zh.HandlerFunc(homeHandler))
	app.GET("/login", zh.HandlerFunc(loginPageHandler))
	app.POST("/login", zh.HandlerFunc(loginHandler))
	app.GET("/profile", zh.HandlerFunc(profileHandler), authMiddleware)
	app.GET("/logout", zh.HandlerFunc(logoutHandler))

	log.Fatal(app.Start())
}

func sessionMiddleware(next http.Handler) http.Handler {
	return sessionManager.LoadAndSave(next)
}

type PageData struct {
	Title   string
	Message string
	Error   bool
	User    string
	Content template.HTML
}

func renderHTML(w http.ResponseWriter, title string, content template.HTML, r *http.Request, message string, isError bool) error {
	user := sessionManager.GetString(r.Context(), "username")

	data := PageData{
		Title:   title,
		Message: message,
		Error:   isError,
		User:    user,
		Content: content,
	}

	w.Header().Set("Content-Type", "text/html")
	return templates.ExecuteTemplate(w, "", data)
}

func homeHandler(w http.ResponseWriter, r *http.Request) error {
	content := template.HTML(`
        <h2>Session Demo</h2>
        <p>This demonstrates SCS sessions with zerohttp.</p>
        <a href="/profile">View Profile</a>
    `)
	return renderHTML(w, "Home", content, r, "", false)
}

func loginPageHandler(w http.ResponseWriter, r *http.Request) error {
	if sessionManager.Exists(r.Context(), "username") {
		return zh.R.Redirect(w, r, "/profile", 302)
	}

	content := template.HTML(`
        <form method="POST">
            <div>Username: <input type="text" name="username" value="admin" required></div>
            <div>Password: <input type="password" name="password" value="password" required></div>
            <button type="submit">Login</button>
        </form>
    `)
	return renderHTML(w, "Login", content, r, "", false)
}

func loginHandler(w http.ResponseWriter, r *http.Request) error {
	username := r.FormValue("username")
	password := r.FormValue("password")

	if username != "admin" || password != "password" {
		content := template.HTML(`
            <form method="POST">
                <div>Username: <input type="text" name="username" required></div>
                <div>Password: <input type="password" name="password" required></div>
                <button type="submit">Login</button>
            </form>
        `)
		return renderHTML(w, "Login", content, r, "Invalid credentials", true)
	}

	sessionManager.RenewToken(r.Context())
	sessionManager.Put(r.Context(), "username", username)
	sessionManager.Put(r.Context(), "login_time", time.Now())

	return zh.R.Redirect(w, r, "/profile", 302)
}

func profileHandler(w http.ResponseWriter, r *http.Request) error {
	username := sessionManager.GetString(r.Context(), "username")
	loginTime := sessionManager.GetTime(r.Context(), "login_time")

	content := template.HTML(`
        <h2>Profile</h2>
        <p><strong>Username:</strong> ` + username + `</p>
        <p><strong>Login Time:</strong> ` + loginTime.Format("2006-01-02 15:04:05") + `</p>
        <p><strong>Session Keys:</strong> ` + fmt.Sprintf("%v", sessionManager.Keys(r.Context())) + `</p>
    `)
	return renderHTML(w, "Profile", content, r, "", false)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) error {
	sessionManager.Destroy(r.Context())
	return zh.R.Redirect(w, r, "/?message=Logged+out", 302)
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !sessionManager.Exists(r.Context(), "username") {
			zh.R.Redirect(w, r, "/login", 302)
			return
		}
		next.ServeHTTP(w, r)
	})
}
