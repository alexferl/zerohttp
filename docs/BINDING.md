# Request Binding

Parse and bind HTTP request bodies to Go structs with automatic type conversion.

## Table of Contents

- [JSON Binding](#json-binding)
- [Form Binding](#form-binding)
- [Multipart Form Binding](#multipart-form-binding)
- [Query Parameter Binding](#query-parameter-binding)
- [Path Parameters](#path-parameters)
- [Individual Parameter Extraction](#individual-parameter-extraction)
- [Custom Binders](#custom-binders)
- [Type Conversion](#type-conversion)
- [Short Aliases](#short-aliases)

## JSON Binding

Parse JSON request bodies with strict validation:

```go
app.POST("/api/users", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    var user struct {
        Name  string `json:"name"`
        Email string `json:"email"`
        Age   int    `json:"age"`
    }

    // Bind JSON with unknown field validation
    if err := zh.Bind.JSON(r, &user); err != nil {
        return err // Returns Problem Detail automatically
    }

    return zh.Render.JSON(w, http.StatusCreated, user)
}))
```

The JSON binder uses `DisallowUnknownFields()` to reject requests with unexpected fields.

## Form Binding

Parse `application/x-www-form-urlencoded` data:

```go
app.POST("/login", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    var form struct {
        Username string   `form:"username"`
        Password string   `form:"password"`
        Remember bool     `form:"remember"`
        Tags     []string `form:"tags"`       // Supports slices
    }

    if err := zh.Bind.Form(r, &form); err != nil {
        return err
    }

    return zh.Render.JSON(w, http.StatusOK, zh.M{"user": form.Username})
}))
```

## Multipart Form Binding

Handle file uploads with `multipart/form-data`:

```go
app.POST("/upload", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    var form struct {
        Description string           `form:"description"`
        Document    *zh.FileHeader   `form:"document"`    // Single file
        Images      []*zh.FileHeader `form:"images"`      // Multiple files
    }

    // maxMemory: bytes to store in memory before temp files
    if err := zh.Bind.MultipartForm(r, &form, 32<<20); err != nil {
        return err
    }

    // Access uploaded files
    if form.Document != nil {
        file, err := form.Document.Open()
        if err != nil {
            return err
        }
        defer file.Close()

        data, _ := io.ReadAll(file)
        // Process file data...
    }

    return zh.Render.JSON(w, http.StatusOK, zh.M{
        "description": form.Description,
        "files":       len(form.Images),
    }))
}))
```

## Query Parameter Binding

Bind query parameters to structs with `query` tags:

```go
app.GET("/search", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    var req struct {
        Query    string   `query:"q"`
        Category string   `query:"category"`
        Tags     []string `query:"tags"`      // Multiple values: ?tags=a&tags=b
        Page     int      `query:"page"`
        Limit    int      `query:"limit"`
        IsActive *bool    `query:"is_active"` // Pointer = optional
    }

    if err := zh.Bind.Query(r, &req); err != nil {
        return err
    }

    // Set defaults
    if req.Page < 1 {
        req.Page = 1
    }
    if req.Limit < 1 {
        req.Limit = 20
    }

    return zh.Render.JSON(w, http.StatusOK, req)
}))
```

### Embedded Structs

Reuse common patterns like pagination:

```go
type Pagination struct {
    Page  int `query:"page"`
    Limit int `query:"limit"`
}

type ListRequest struct {
    Pagination
    Search string `query:"search"`
}

app.GET("/items", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    var req ListRequest
    if err := zh.Bind.Query(r, &req); err != nil {
        return err
    }
    return zh.Render.JSON(w, http.StatusOK, req)
}))
```

## Path Parameters

Type-safe path parameter extraction:

```go
// Basic string extraction
app.GET("/users/{id}", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    id := zh.Param(r, "id")
    return zh.Render.JSON(w, http.StatusOK, zh.M{"user_id": id})
}))

// Typed extraction
app.GET("/items/{itemID}", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    itemID, err := zh.ParamAs[int](r, "itemID")
    if err != nil {
        return zh.NewProblemDetail(http.StatusBadRequest, "Invalid itemID").Render(w)
    }
    return zh.Render.JSON(w, http.StatusOK, zh.M{"item_id": itemID})
}))

// With default value
app.GET("/products/{category}", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    category := zh.ParamOrDefault(r, "category", "all")
    return zh.Render.JSON(w, http.StatusOK, zh.M{"category": category})
}))
```

## Individual Parameter Extraction

Extract single query parameters:

```go
app.GET("/search", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
    // With type conversion
    userID, err := zh.QueryParamAs[int](r, "user_id")
    if err != nil {
        return zh.NewProblemDetail(http.StatusBadRequest, "Invalid user_id").Render(w)
    }

    // With default value
    page := zh.QueryParamAsOrDefault(r, "page", 1)
    limit := zh.QueryParamAsOrDefault(r, "limit", 20)

    // Simple string
    sort := zh.QueryParam(r, "sort")

    return zh.Render.JSON(w, http.StatusOK, zh.M{
        "user_id": userID,
        "page":    page,
        "limit":   limit,
        "sort":    sort,
    })
}))
```

## Custom Binders

Implement the `Binder` interface for custom binding logic:

```go
type MyBinder struct{}

func (b *MyBinder) JSON(r *http.Request, dst any) error {
    decoder := json.NewDecoder(r.Body)
    decoder.UseNumber() // Use json.Number instead of float64
    return decoder.Decode(dst)
}

func (b *MyBinder) Form(r *http.Request, dst any) error {
    // Custom form binding
    return nil
}

func (b *MyBinder) MultipartForm(r *http.Request, dst any, maxMemory int64) error {
    // Custom multipart form binding
    return nil
}

func (b *MyBinder) Query(r *http.Request, dst any) error {
    // Custom query binding
    return nil
}

// Replace default binder
zh.Bind = &MyBinder{}
```

## Type Conversion

Form and query binders automatically convert string values to Go types:

| Type       | Example Input          | Result        |
|------------|------------------------|---------------|
| `string`   | `name=John`            | `"John"`      |
| `int`      | `age=25`               | `25`          |
| `bool`     | `active=true`          | `true`        |
| `[]string` | `tags=a&tags=b`        | `["a", "b"]`  |
| `[]int`    | `ids=1&ids=2`          | `[1, 2]`      |
| `*string`  | `optional=` or missing | `nil` or `""` |

**Supported Types:** `string`, `int`, `int8`, `int16`, `int32`, `int64`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `float32`, `float64`, `bool`, slices, and pointers.

## Short Aliases

- `zh.B` - Short for `zh.Bind`
