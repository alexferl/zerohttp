<img src="https://r2cdn.perplexity.ai/pplx-full-logo-primary-dark%402x.png" style="height:64px;margin-right:32px"/>

# don't need Standard Library Comparison and give me markdown

Here's the markdown section for static file serving:

## Static File Serving

Serve static files from embedded filesystems or directories:

```go
//go:embed static
var staticFiles embed.FS

//go:embed dist
var appFiles embed.FS

func main() {
app := zh.New()

    // API routes
    app.GET("/api/health", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
        return zh.R.JSON(w, 200, zh.M{"status": "healthy"})
    }))
    
    // Serve static assets (CSS, JS, images) from embedded FS
    app.Files("/static/", staticFiles, "static")
    
    // Serve files from directory (uploads, user content)
    app.FilesDir("/uploads/", "./uploads")
    
    // Serve web application with client-side routing support
    app.Static(appFiles, "dist", "/api/")
    
    // Or serve from directory for development
    // app.StaticDir("./dist", "/api/")
    
    log.Fatal(app.Start())
    }

```

### Static File Methods

- **`Files()`** - Serves files from embedded FS without fallback
- **`FilesDir()`** - Serves files from directory without fallback  
- **`Static()`** - Serves web app from embedded FS with index.html fallback for client-side routing
- **`StaticDir()`** - Serves web app from directory with index.html fallback for client-side routing

The `Static` methods support API prefix exclusions - requests matching specified prefixes return 404 instead of falling back to index.html, allowing API and static routes to coexist.
```
