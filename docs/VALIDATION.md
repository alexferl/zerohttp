# Validation

zerohttp includes a built-in struct tag-based validation system with no external dependencies.

## Table of Contents

- [Basic Usage](#basic-usage)
- [Available Validators](#available-validators)
  - [Core](#core)
  - [String](#string)
  - [Numeric](#numeric)
  - [Format](#format)
  - [Network](#network)
  - [Collection](#collection)
- [Combining Validators](#combining-validators)
- [Nested Structs](#nested-structs)
- [Pointer Fields](#pointer-fields)
- [Custom Validators](#custom-validators)
- [Error Handling](#error-handling)
- [Short Aliases](#short-aliases)

## Basic Usage

```go
import zh "github.com/alexferl/zerohttp"

type User struct {
    Name  string `validate:"required,min=2,max=50"`
    Email string `validate:"required,email"`
    Age   int    `validate:"min=13,max=120"`
}

user := User{Name: "John", Email: "john@example.com", Age: 25}
if err := zh.Validate.Struct(&user); err != nil {
    // Handle validation error
}
```

## Available Validators

### Core

| Validator   | Description                        | Example                      |
|-------------|------------------------------------|------------------------------|
| `required`  | Field must not be empty/zero value | `validate:"required"`        |
| `omitempty` | Skip validation if field is empty  | `validate:"omitempty,min=5"` |
| `eq`        | Equal to value                     | `validate:"eq=5"`            |
| `ne`        | Not equal to value                 | `validate:"ne=0"`            |

### String

| Validator    | Description            | Example                                    |
|--------------|------------------------|--------------------------------------------|
| `min`        | Minimum length (runes) | `validate:"min=5"`                         |
| `max`        | Maximum length (runes) | `validate:"max=100"`                       |
| `len`        | Exact length           | `validate:"len=8"`                         |
| `contains`   | Contains substring     | `validate:"contains=admin"`                |
| `startswith` | Starts with prefix     | `validate:"startswith=pre"`                |
| `endswith`   | Ends with suffix       | `validate:"endswith=suffix"`               |
| `excludes`   | Excludes substring     | `validate:"excludes=password"`             |
| `alpha`      | Letters only (Unicode) | `validate:"alpha"`                         |
| `alphanum`   | Letters and numbers    | `validate:"alphanum"`                      |
| `lowercase`  | All lowercase          | `validate:"lowercase"`                     |
| `uppercase`  | All uppercase          | `validate:"uppercase"`                     |
| `ascii`      | ASCII characters       | `validate:"ascii"`                         |
| `printascii` | Printable ASCII only   | `validate:"printascii"`                    |
| `numeric`    | Numeric digits only    | `validate:"numeric"`                       |
| `oneof`      | One of allowed values  | `validate:"oneof=active inactive pending"` |

### Numeric

| Validator | Description           | Example              |
|-----------|-----------------------|----------------------|
| `min`     | Minimum value         | `validate:"min=0"`   |
| `max`     | Maximum value         | `validate:"max=100"` |
| `gt`      | Greater than          | `validate:"gt=0"`    |
| `lt`      | Less than             | `validate:"lt=100"`  |
| `gte`     | Greater than or equal | `validate:"gte=0"`   |
| `lte`     | Less than or equal    | `validate:"lte=100"` |

### Format

| Validator     | Description                                   | Example                          |
|---------------|-----------------------------------------------|----------------------------------|
| `email`       | Email address                                 | `validate:"email"`               |
| `uuid`        | UUID format                                   | `validate:"uuid"`                |
| `datetime`    | Custom datetime format                        | `validate:"datetime=2006-01-02"` |
| `base64`      | Base64 encoded                                | `validate:"base64"`              |
| `hexadecimal` | Hex string                                    | `validate:"hexadecimal"`         |
| `hexcolor`    | Hex color (#RGB, #RGBA, #RRGGBB, #RRGGBBAA)   | `validate:"hexcolor"`            |
| `e164`        | E.164 phone number                            | `validate:"e164"`                |
| `semver`      | Semantic version                              | `validate:"semver"`              |
| `jwt`         | JWT format (3 base64 parts)                   | `validate:"jwt"`                 |
| `boolean`     | Boolean string (true/false/yes/no/on/off/1/0) | `validate:"boolean"`             |
| `json`        | Valid JSON                                    | `validate:"json"`                |

### Network

| Validator  | Description           | Example               |
|------------|-----------------------|-----------------------|
| `ip`       | IP address (v4 or v6) | `validate:"ip"`       |
| `ipv4`     | IPv4 address          | `validate:"ipv4"`     |
| `ipv6`     | IPv6 address          | `validate:"ipv6"`     |
| `cidr`     | CIDR notation         | `validate:"cidr"`     |
| `hostname` | RFC 952 hostname      | `validate:"hostname"` |
| `uri`      | Absolute URI          | `validate:"uri"`      |
| `url`      | HTTP/HTTPS URL        | `validate:"url"`      |

### Collection

| Validator | Description              | Example                 |
|-----------|--------------------------|-------------------------|
| `unique`  | Unique elements in slice | `validate:"unique"`     |
| `each`    | Validate each element    | `validate:"each,min=3"` |

## Combining Validators

Multiple validators can be combined with commas. They are evaluated in order:

```go
type Product struct {
    Name     string   `validate:"required,min=2,max=100"`
    Price    float64  `validate:"required,gt=0"`
    Tags     []string `validate:"unique,each,min=2,max=20"`
}
```

## Nested Structs

Validation automatically recurses into nested structs:

```go
type Address struct {
    Street string `validate:"required"`
    City   string `validate:"required"`
}

type Person struct {
    Name    string  `validate:"required"`
    Address Address // validated recursively
}
```

For slices/maps of structs, use the `each` validator:

```go
type Order struct {
    Items []LineItem `validate:"each"` // validates each LineItem
}
```

## Pointer Fields

Pointer fields are dereferenced before validation. Use `omitempty` to make optional:

```go
type User struct {
    Name     *string `validate:"omitempty,min=2"` // nil or valid
    Nickname *string `validate:"required,min=2"`  // must not be nil
}
```

## Custom Validators

Register custom validators with `V.Register`:

```go
import (
    "reflect"
    zh "github.com/alexferl/zerohttp"
)

// Register a custom validator
zh.Validate.Register("even", func(value reflect.Value, param string) error {
    if value.Kind() != reflect.Int {
        return fmt.Errorf("even only validates integers")
    }
    if value.Int()%2 != 0 {
        return fmt.Errorf("must be even")
    }
    return nil
})

// Use in struct tags
type Config struct {
    Port int `validate:"required,even"`
}
```

## Error Handling

`Validate.Struct()` returns a `ValidationErrors` map keyed by field name:

```go
if err := zh.Validate.Struct(&user); err != nil {
    var ve zh.ValidationErrors
    if errors.As(err, &ve) {
        // Get all errors for a field
        errs := ve.FieldErrors("Email")
        for _, e := range errs {
            fmt.Println(e) // "required" or "must be a valid email"
        }

        // Check if any errors exist
        if ve.HasErrors() {
            // Render as Problem Details
            pd := zh.NewValidationProblemDetail("Validation failed", ve)
            pd.Render(w)
        }
    }
}
```

Errors use JSON field names when available:

```go
type User struct {
    UserName string `json:"user_name" validate:"required"`
}
// Error will be keyed as "user_name" not "UserName"
```

## Short Aliases

- `zh.V` - Short for `zh.Validate`
