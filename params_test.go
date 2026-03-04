package zerohttp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParam(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		path      string
		paramName string
		want      string
	}{
		{
			name:      "extract string param",
			pattern:   "/users/{id}",
			path:      "/users/123",
			paramName: "id",
			want:      "123",
		},
		{
			name:      "extract multiple params",
			pattern:   "/users/{userID}/posts/{postID}",
			path:      "/users/42/posts/99",
			paramName: "postID",
			want:      "99",
		},
		{
			name:      "missing param returns empty",
			pattern:   "/users",
			path:      "/users",
			paramName: "id",
			want:      "",
		},
		{
			name:      "extract slug",
			pattern:   "/blog/{slug}",
			path:      "/blog/hello-world",
			paramName: "slug",
			want:      "hello-world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			var got string
			mux.HandleFunc(tt.pattern, func(w http.ResponseWriter, r *http.Request) {
				got = Param(r, tt.paramName)
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if got != tt.want {
				t.Errorf("Param() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParamOrDefault(t *testing.T) {
	tests := []struct {
		name       string
		pattern    string
		path       string
		paramName  string
		defaultVal string
		want       string
	}{
		{
			name:       "returns param when present",
			pattern:    "/users/{id}",
			path:       "/users/123",
			paramName:  "id",
			defaultVal: "default",
			want:       "123",
		},
		{
			name:       "returns default when missing",
			pattern:    "/users",
			path:       "/users",
			paramName:  "id",
			defaultVal: "default",
			want:       "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			var got string
			mux.HandleFunc(tt.pattern, func(w http.ResponseWriter, r *http.Request) {
				got = ParamOrDefault(r, tt.paramName, tt.defaultVal)
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if got != tt.want {
				t.Errorf("ParamOrDefault() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParamAs(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		path      string
		paramName string
		value     string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid int",
			pattern:   "/users/{id}",
			path:      "/users/123",
			paramName: "id",
			value:     "123",
			wantErr:   false,
		},
		{
			name:      "invalid int",
			pattern:   "/users/{id}",
			path:      "/users/abc",
			paramName: "id",
			wantErr:   true,
			errMsg:    `parameter "id": invalid int`,
		},
		{
			name:      "missing param",
			pattern:   "/users",
			path:      "/users",
			paramName: "id",
			wantErr:   true,
			errMsg:    `parameter "id" not found`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			var gotErr error
			mux.HandleFunc(tt.pattern, func(w http.ResponseWriter, r *http.Request) {
				_, gotErr = ParamAs[int](r, tt.paramName)
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if tt.wantErr {
				if gotErr == nil {
					t.Errorf("ParamAs() expected error, got nil")
					return
				}
				if tt.errMsg != "" {
					if got := gotErr.Error(); got[:len(tt.errMsg)] != tt.errMsg {
						t.Errorf("ParamAs() error = %q, want prefix %q", got, tt.errMsg)
					}
				}
			} else {
				if gotErr != nil {
					t.Errorf("ParamAs() unexpected error: %v", gotErr)
				}
			}
		})
	}
}

func TestParamAs_Bool(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/flag/true", true},
		{"/flag/1", true},
		{"/flag/false", false},
		{"/flag/0", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/flag/{enabled}", func(w http.ResponseWriter, r *http.Request) {
				val, err := ParamAs[bool](r, "enabled")
				if err != nil {
					t.Errorf("ParamAs[bool]() error = %v", err)
					return
				}
				if val != tt.want {
					t.Errorf("ParamAs[bool]() = %v, want %v", val, tt.want)
				}
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
		})
	}
}

func TestParamAs_CommonTypes(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		testFn  func(*http.Request) error
	}{
		{
			name:    "int",
			pattern: "/users/{id}",
			path:    "/users/42",
			testFn: func(r *http.Request) error {
				val, err := ParamAs[int](r, "id")
				if err != nil {
					return err
				}
				if val != 42 {
					t.Errorf("ParamAs[int]() = %d, want 42", val)
				}
				return nil
			},
		},
		{
			name:    "int64",
			pattern: "/items/{id}",
			path:    "/items/9223372036854775807",
			testFn: func(r *http.Request) error {
				val, err := ParamAs[int64](r, "id")
				if err != nil {
					return err
				}
				if val != 9223372036854775807 {
					t.Errorf("ParamAs[int64]() = %d, want max int64", val)
				}
				return nil
			},
		},
		{
			name:    "uint",
			pattern: "/count/{n}",
			path:    "/count/100",
			testFn: func(r *http.Request) error {
				val, err := ParamAs[uint](r, "n")
				if err != nil {
					return err
				}
				if val != 100 {
					t.Errorf("ParamAs[uint]() = %d, want 100", val)
				}
				return nil
			},
		},
		{
			name:    "float64",
			pattern: "/price/{amount}",
			path:    "/price/19.99",
			testFn: func(r *http.Request) error {
				val, err := ParamAs[float64](r, "amount")
				if err != nil {
					return err
				}
				if val != 19.99 {
					t.Errorf("ParamAs[float64]() = %f, want 19.99", val)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			var gotErr error
			mux.HandleFunc(tt.pattern, func(w http.ResponseWriter, r *http.Request) {
				gotErr = tt.testFn(r)
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if gotErr != nil {
				t.Errorf("ParamAs[%s]() error = %v", tt.name, gotErr)
			}
		})
	}
}

func TestParamExtractor_Interface(t *testing.T) {
	// Ensure defaultParamsExtractor implements ParamExtractor
	var _ ParamExtractor = Params
}

func TestParamAsOrDefault(t *testing.T) {
	tests := []struct {
		name       string
		pattern    string
		path       string
		paramName  string
		defaultVal int
		want       int
	}{
		{
			name:       "returns param when present",
			pattern:    "/users/{id}",
			path:       "/users/123",
			paramName:  "id",
			defaultVal: 0,
			want:       123,
		},
		{
			name:       "returns default when param missing",
			pattern:    "/users",
			path:       "/users",
			paramName:  "id",
			defaultVal: 42,
			want:       42,
		},
		{
			name:       "returns default on invalid type",
			pattern:    "/users/{id}",
			path:       "/users/abc",
			paramName:  "id",
			defaultVal: 99,
			want:       99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			var got int
			mux.HandleFunc(tt.pattern, func(w http.ResponseWriter, r *http.Request) {
				got = ParamAsOrDefault(r, tt.paramName, tt.defaultVal)
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if got != tt.want {
				t.Errorf("ParamAsOrDefault() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestParamAs_Types(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		path      string
		paramName string
		expected  any
		fn        func(*http.Request, string) (any, error)
	}{
		{
			name:      "string type",
			pattern:   "/items/{name}",
			path:      "/items/widget",
			paramName: "name",
			expected:  "widget",
			fn: func(r *http.Request, n string) (any, error) {
				return ParamAs[string](r, n)
			},
		},
		{
			name:      "int8 type",
			pattern:   "/items/{id}",
			path:      "/items/100",
			paramName: "id",
			expected:  int8(100),
			fn: func(r *http.Request, n string) (any, error) {
				return ParamAs[int8](r, n)
			},
		},
		{
			name:      "int16 type",
			pattern:   "/items/{id}",
			path:      "/items/1000",
			paramName: "id",
			expected:  int16(1000),
			fn: func(r *http.Request, n string) (any, error) {
				return ParamAs[int16](r, n)
			},
		},
		{
			name:      "int32 type",
			pattern:   "/items/{id}",
			path:      "/items/100000",
			paramName: "id",
			expected:  int32(100000),
			fn: func(r *http.Request, n string) (any, error) {
				return ParamAs[int32](r, n)
			},
		},
		{
			name:      "uint type",
			pattern:   "/items/{count}",
			path:      "/items/50",
			paramName: "count",
			expected:  uint(50),
			fn: func(r *http.Request, n string) (any, error) {
				return ParamAs[uint](r, n)
			},
		},
		{
			name:      "uint8 type",
			pattern:   "/items/{count}",
			path:      "/items/200",
			paramName: "count",
			expected:  uint8(200),
			fn: func(r *http.Request, n string) (any, error) {
				return ParamAs[uint8](r, n)
			},
		},
		{
			name:      "uint16 type",
			pattern:   "/items/{count}",
			path:      "/items/5000",
			paramName: "count",
			expected:  uint16(5000),
			fn: func(r *http.Request, n string) (any, error) {
				return ParamAs[uint16](r, n)
			},
		},
		{
			name:      "uint32 type",
			pattern:   "/items/{count}",
			path:      "/items/100000",
			paramName: "count",
			expected:  uint32(100000),
			fn: func(r *http.Request, n string) (any, error) {
				return ParamAs[uint32](r, n)
			},
		},
		{
			name:      "uint64 type",
			pattern:   "/items/{id}",
			path:      "/items/9223372036854775807",
			paramName: "id",
			expected:  uint64(9223372036854775807),
			fn: func(r *http.Request, n string) (any, error) {
				return ParamAs[uint64](r, n)
			},
		},
		{
			name:      "float32 type",
			pattern:   "/items/{price}",
			path:      "/items/19.99",
			paramName: "price",
			expected:  float32(19.99),
			fn: func(r *http.Request, n string) (any, error) {
				return ParamAs[float32](r, n)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			var got any
			var err error
			mux.HandleFunc(tt.pattern, func(w http.ResponseWriter, r *http.Request) {
				got, err = tt.fn(r, tt.paramName)
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if err != nil {
				t.Errorf("ParamAs() error = %v", err)
				return
			}
			if got != tt.expected {
				t.Errorf("ParamAs() = %v (%T), want %v (%T)", got, got, tt.expected, tt.expected)
			}
		})
	}
}

func TestParamAs_InvalidValues(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr string
		fn      func(*http.Request) error
	}{
		{
			name:    "invalid int8",
			path:    "/val/999",
			wantErr: `parameter "val": invalid int8`,
			fn: func(r *http.Request) error {
				_, err := ParamAs[int8](r, "val")
				return err
			},
		},
		{
			name:    "invalid int16",
			path:    "/val/99999",
			wantErr: `parameter "val": invalid int16`,
			fn: func(r *http.Request) error {
				_, err := ParamAs[int16](r, "val")
				return err
			},
		},
		{
			name:    "invalid int32",
			path:    "/val/9999999999",
			wantErr: `parameter "val": invalid int32`,
			fn: func(r *http.Request) error {
				_, err := ParamAs[int32](r, "val")
				return err
			},
		},
		{
			name:    "invalid int64",
			path:    "/val/9999999999999999999",
			wantErr: `parameter "val": invalid int64`,
			fn: func(r *http.Request) error {
				_, err := ParamAs[int64](r, "val")
				return err
			},
		},
		{
			name:    "invalid uint",
			path:    "/val/abc",
			wantErr: `parameter "val": invalid uint`,
			fn: func(r *http.Request) error {
				_, err := ParamAs[uint](r, "val")
				return err
			},
		},
		{
			name:    "invalid uint8",
			path:    "/val/999",
			wantErr: `parameter "val": invalid uint8`,
			fn: func(r *http.Request) error {
				_, err := ParamAs[uint8](r, "val")
				return err
			},
		},
		{
			name:    "invalid uint16",
			path:    "/val/999999",
			wantErr: `parameter "val": invalid uint16`,
			fn: func(r *http.Request) error {
				_, err := ParamAs[uint16](r, "val")
				return err
			},
		},
		{
			name:    "invalid uint32",
			path:    "/val/99999999999",
			wantErr: `parameter "val": invalid uint32`,
			fn: func(r *http.Request) error {
				_, err := ParamAs[uint32](r, "val")
				return err
			},
		},
		{
			name:    "invalid uint64",
			path:    "/val/abc",
			wantErr: `parameter "val": invalid uint64`,
			fn: func(r *http.Request) error {
				_, err := ParamAs[uint64](r, "val")
				return err
			},
		},
		{
			name:    "invalid float32",
			path:    "/val/abc",
			wantErr: `parameter "val": invalid float32`,
			fn: func(r *http.Request) error {
				_, err := ParamAs[float32](r, "val")
				return err
			},
		},
		{
			name:    "invalid float64",
			path:    "/val/abc",
			wantErr: `parameter "val": invalid float64`,
			fn: func(r *http.Request) error {
				_, err := ParamAs[float64](r, "val")
				return err
			},
		},
		{
			name:    "invalid bool",
			path:    "/val/maybe",
			wantErr: `parameter "val": invalid bool`,
			fn: func(r *http.Request) error {
				_, err := ParamAs[bool](r, "val")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			var gotErr error
			mux.HandleFunc("/val/{val}", func(w http.ResponseWriter, r *http.Request) {
				gotErr = tt.fn(r)
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if gotErr == nil {
				t.Errorf("expected error, got nil")
				return
			}
			if got := gotErr.Error(); got[:len(tt.wantErr)] != tt.wantErr {
				t.Errorf("error = %q, want prefix %q", got, tt.wantErr)
			}
		})
	}
}

func TestPAlias(t *testing.T) {
	mux := http.NewServeMux()
	var got string
	mux.HandleFunc("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		got = P.Param(r, "id")
	})

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if got != "123" {
		t.Errorf("P.Param() = %q, want %q", got, "123")
	}
}

func TestDefaultParamsExtractor(t *testing.T) {
	t.Run("Param method", func(t *testing.T) {
		mux := http.NewServeMux()
		extractor := &defaultParamsExtractor{}
		var got string
		mux.HandleFunc("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
			got = extractor.Param(r, "id")
		})

		req := httptest.NewRequest(http.MethodGet, "/users/456", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if got != "456" {
			t.Errorf("Param() = %q, want %q", got, "456")
		}
	})

	t.Run("ParamOrDefault method with value", func(t *testing.T) {
		mux := http.NewServeMux()
		extractor := &defaultParamsExtractor{}
		var got string
		mux.HandleFunc("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
			got = extractor.ParamOrDefault(r, "id", "default")
		})

		req := httptest.NewRequest(http.MethodGet, "/users/789", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if got != "789" {
			t.Errorf("ParamOrDefault() = %q, want %q", got, "789")
		}
	})

	t.Run("ParamOrDefault method with default", func(t *testing.T) {
		mux := http.NewServeMux()
		extractor := &defaultParamsExtractor{}
		var got string
		mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
			got = extractor.ParamOrDefault(r, "id", "default")
		})

		req := httptest.NewRequest(http.MethodGet, "/users", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if got != "default" {
			t.Errorf("ParamOrDefault() = %q, want %q", got, "default")
		}
	})
}
