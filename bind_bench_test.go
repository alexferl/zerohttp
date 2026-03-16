package zerohttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/alexferl/zerohttp/httpx"
)

// BenchmarkBinder_JSON_Baseline compares JSON binding vs stdlib json.Decoder
func BenchmarkBinder_JSON_Baseline(b *testing.B) {
	type User struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Age   int    `json:"age"`
	}

	jsonData := `{"name":"John Doe","email":"john@example.com","age":30}`

	b.Run("Stdlib_Decoder", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			var result User
			decoder := json.NewDecoder(strings.NewReader(jsonData))
			decoder.DisallowUnknownFields()
			_ = decoder.Decode(&result)
		}
	})

	b.Run("Binder_JSON", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			var result User
			_ = B.JSON(strings.NewReader(jsonData), &result)
		}
	})

	b.Run("Stdlib_Unmarshal", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			var result User
			_ = json.Unmarshal([]byte(jsonData), &result)
		}
	})
}

// BenchmarkBinder_JSON_PayloadSizes measures JSON binding with different payload sizes
func BenchmarkBinder_JSON_PayloadSizes(b *testing.B) {
	type Item struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Value string `json:"value"`
	}

	type Response struct {
		Items []Item `json:"items"`
		Count int    `json:"count"`
	}

	sizes := []struct {
		name string
		data string
	}{
		{
			name: "Tiny_100B",
			data: `{"items":[{"id":1,"name":"test","value":"a"}],"count":1}`,
		},
		{
			name: "Small_1KB",
			data: func() string {
				items := make([]Item, 10)
				for i := range 10 {
					items[i] = Item{ID: i, Name: fmt.Sprintf("Item %d", i), Value: fmt.Sprintf("Value %d", i)}
				}
				data, _ := json.Marshal(Response{Items: items, Count: 10})
				return string(data)
			}(),
		},
		{
			name: "Medium_10KB",
			data: func() string {
				items := make([]Item, 100)
				for i := range 100 {
					items[i] = Item{ID: i, Name: fmt.Sprintf("Item %d", i), Value: strings.Repeat("x", 50)}
				}
				data, _ := json.Marshal(Response{Items: items, Count: 100})
				return string(data)
			}(),
		},
		{
			name: "Large_100KB",
			data: func() string {
				items := make([]Item, 1000)
				for i := range 1000 {
					items[i] = Item{ID: i, Name: fmt.Sprintf("Item %d", i), Value: strings.Repeat("y", 50)}
				}
				data, _ := json.Marshal(Response{Items: items, Count: 1000})
				return string(data)
			}(),
		},
	}

	for _, s := range sizes {
		b.Run(s.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				var result Response
				_ = B.JSON(strings.NewReader(s.data), &result)
			}
		})
	}
}

// BenchmarkBinder_JSON_DataTypes compares binding different Go types from JSON
func BenchmarkBinder_JSON_DataTypes(b *testing.B) {
	b.Run("SimpleStruct", func(b *testing.B) {
		type Simple struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}
		data := `{"name":"test","value":42}`

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			var result Simple
			_ = B.JSON(strings.NewReader(data), &result)
		}
	})

	b.Run("NestedStruct", func(b *testing.B) {
		type Address struct {
			Street string `json:"street"`
			City   string `json:"city"`
		}
		type Person struct {
			Name    string  `json:"name"`
			Address Address `json:"address"`
		}
		data := `{"name":"John","address":{"street":"123 Main","city":"NYC"}}`

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			var result Person
			_ = B.JSON(strings.NewReader(data), &result)
		}
	})

	b.Run("Slice", func(b *testing.B) {
		type List struct {
			Items []string `json:"items"`
		}
		data := `{"items":["a","b","c","d","e"]}`

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			var result List
			_ = B.JSON(strings.NewReader(data), &result)
		}
	})

	b.Run("PointerFields", func(b *testing.B) {
		type WithPointers struct {
			Name  *string `json:"name"`
			Age   *int    `json:"age"`
			Email *string `json:"email"`
		}
		data := `{"name":"John","age":30,"email":"john@example.com"}`

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			var result WithPointers
			_ = B.JSON(strings.NewReader(data), &result)
		}
	})

	b.Run("MixedTypes", func(b *testing.B) {
		type Mixed struct {
			Name   string   `json:"name"`
			Age    int      `json:"age"`
			Active bool     `json:"active"`
			Score  float64  `json:"score"`
			Tags   []string `json:"tags"`
		}
		data := `{"name":"John","age":30,"active":true,"score":95.5,"tags":["go","web"]}`

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			var result Mixed
			_ = B.JSON(strings.NewReader(data), &result)
		}
	})
}

// BenchmarkBinder_Form measures form data binding performance
func BenchmarkBinder_Form(b *testing.B) {
	type FormData struct {
		Name   string   `form:"name"`
		Email  string   `form:"email"`
		Age    int      `form:"age"`
		Active bool     `form:"active"`
		Tags   []string `form:"tags"`
	}

	formData := url.Values{
		"name":   []string{"John Doe"},
		"email":  []string{"john@example.com"},
		"age":    []string{"30"},
		"active": []string{"true"},
		"tags":   []string{"go", "web", "api"},
	}

	b.Run("Simple", func(b *testing.B) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(formData.Encode()))
		req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationFormURLEncoded)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			// Need fresh request body for each iteration
			req.Body = io.NopCloser(strings.NewReader(formData.Encode()))
			var result FormData
			_ = B.Form(req, &result)
		}
	})

	b.Run("FieldCount_5", func(b *testing.B) {
		type SmallForm struct {
			F1 string `form:"f1"`
			F2 string `form:"f2"`
			F3 int    `form:"f3"`
			F4 bool   `form:"f4"`
			F5 string `form:"f5"`
		}
		data := url.Values{
			"f1": []string{"val1"},
			"f2": []string{"val2"},
			"f3": []string{"10"},
			"f4": []string{"true"},
			"f5": []string{"val5"},
		}
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationFormURLEncoded)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			req.Body = io.NopCloser(strings.NewReader(data.Encode()))
			var result SmallForm
			_ = B.Form(req, &result)
		}
	})

	b.Run("FieldCount_20", func(b *testing.B) {
		type LargeForm struct {
			F1  string `form:"f1"`
			F2  string `form:"f2"`
			F3  string `form:"f3"`
			F4  string `form:"f4"`
			F5  string `form:"f5"`
			F6  int    `form:"f6"`
			F7  int    `form:"f7"`
			F8  int    `form:"f8"`
			F9  int    `form:"f9"`
			F10 int    `form:"f10"`
			F11 bool   `form:"f11"`
			F12 bool   `form:"f12"`
			F13 bool   `form:"f13"`
			F14 bool   `form:"f14"`
			F15 bool   `form:"f15"`
			F16 string `form:"f16"`
			F17 string `form:"f17"`
			F18 string `form:"f18"`
			F19 string `form:"f19"`
			F20 string `form:"f20"`
		}
		data := url.Values{
			"f1": []string{"v1"}, "f2": []string{"v2"}, "f3": []string{"v3"},
			"f4": []string{"v4"}, "f5": []string{"v5"}, "f6": []string{"6"},
			"f7": []string{"7"}, "f8": []string{"8"}, "f9": []string{"9"},
			"f10": []string{"10"}, "f11": []string{"true"}, "f12": []string{"false"},
			"f13": []string{"true"}, "f14": []string{"false"}, "f15": []string{"true"},
			"f16": []string{"v16"}, "f17": []string{"v17"}, "f18": []string{"v18"},
			"f19": []string{"v19"}, "f20": []string{"v20"},
		}
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationFormURLEncoded)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			req.Body = io.NopCloser(strings.NewReader(data.Encode()))
			var result LargeForm
			_ = B.Form(req, &result)
		}
	})
}

// BenchmarkBinder_Form_DataTypes measures form binding with different data types
func BenchmarkBinder_Form_DataTypes(b *testing.B) {
	b.Run("Strings", func(b *testing.B) {
		type StringForm struct {
			Name  string `form:"name"`
			Email string `form:"email"`
		}
		data := url.Values{"name": []string{"John"}, "email": []string{"john@example.com"}}
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationFormURLEncoded)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			req.Body = io.NopCloser(strings.NewReader(data.Encode()))
			var result StringForm
			_ = B.Form(req, &result)
		}
	})

	b.Run("Integers", func(b *testing.B) {
		type IntForm struct {
			Age   int  `form:"age"`
			Count int  `form:"count"`
			Size  int8 `form:"size"`
		}
		data := url.Values{"age": []string{"30"}, "count": []string{"100"}, "size": []string{"127"}}
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationFormURLEncoded)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			req.Body = io.NopCloser(strings.NewReader(data.Encode()))
			var result IntForm
			_ = B.Form(req, &result)
		}
	})

	b.Run("Floats", func(b *testing.B) {
		type FloatForm struct {
			Price  float64 `form:"price"`
			Rating float32 `form:"rating"`
		}
		data := url.Values{"price": []string{"99.99"}, "rating": []string{"4.5"}}
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationFormURLEncoded)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			req.Body = io.NopCloser(strings.NewReader(data.Encode()))
			var result FloatForm
			_ = B.Form(req, &result)
		}
	})

	b.Run("Booleans", func(b *testing.B) {
		type BoolForm struct {
			Active     bool `form:"active"`
			Verified   bool `form:"verified"`
			Subscribed bool `form:"subscribed"`
		}
		data := url.Values{"active": []string{"true"}, "verified": []string{"1"}, "subscribed": []string{"false"}}
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationFormURLEncoded)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			req.Body = io.NopCloser(strings.NewReader(data.Encode()))
			var result BoolForm
			_ = B.Form(req, &result)
		}
	})

	b.Run("Slices", func(b *testing.B) {
		type SliceForm struct {
			Tags []string `form:"tags"`
			IDs  []int    `form:"ids"`
		}
		data := url.Values{"tags": []string{"go", "web", "api"}, "ids": []string{"1", "2", "3"}}
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationFormURLEncoded)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			req.Body = io.NopCloser(strings.NewReader(data.Encode()))
			var result SliceForm
			_ = B.Form(req, &result)
		}
	})
}

// BenchmarkBinder_Query measures query parameter binding performance
func BenchmarkBinder_Query(b *testing.B) {
	type QueryParams struct {
		Page   int      `query:"page"`
		Limit  int      `query:"limit"`
		Search string   `query:"search"`
		Active bool     `query:"active"`
		Tags   []string `query:"tags"`
	}

	b.Run("Simple", func(b *testing.B) {
		req := httptest.NewRequest(http.MethodGet, "/?page=1&limit=20&search=test", nil)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			var result QueryParams
			_ = B.Query(req, &result)
		}
	})

	b.Run("WithSlices", func(b *testing.B) {
		req := httptest.NewRequest(http.MethodGet, "/?tags=go&tags=web&tags=api&page=1&limit=10", nil)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			var result QueryParams
			_ = B.Query(req, &result)
		}
	})

	b.Run("EmptyQuery", func(b *testing.B) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			var result QueryParams
			_ = B.Query(req, &result)
		}
	})

	b.Run("ManyParams", func(b *testing.B) {
		type ManyParams struct {
			Param1  string `query:"param1"`
			Param2  string `query:"param2"`
			Param3  string `query:"param3"`
			Param4  string `query:"param4"`
			Param5  string `query:"param5"`
			Param6  int    `query:"param6"`
			Param7  int    `query:"param7"`
			Param8  bool   `query:"param8"`
			Param9  bool   `query:"param9"`
			Param10 string `query:"param10"`
		}
		req := httptest.NewRequest(http.MethodGet, "/?param1=a&param2=b&param3=c&param4=d&param5=e&param6=6&param7=7&param8=true&param9=false&param10=j", nil)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			var result ManyParams
			_ = B.Query(req, &result)
		}
	})
}

// BenchmarkBinder_Query_DataTypes measures query binding with different types
func BenchmarkBinder_Query_DataTypes(b *testing.B) {
	b.Run("Strings", func(b *testing.B) {
		type StringQuery struct {
			Name  string `query:"name"`
			Email string `query:"email"`
		}
		req := httptest.NewRequest(http.MethodGet, "/?name=John&email=john@example.com", nil)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			var result StringQuery
			_ = B.Query(req, &result)
		}
	})

	b.Run("Integers", func(b *testing.B) {
		type IntQuery struct {
			Page  int `query:"page"`
			Limit int `query:"limit"`
		}
		req := httptest.NewRequest(http.MethodGet, "/?page=1&limit=20", nil)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			var result IntQuery
			_ = B.Query(req, &result)
		}
	})

	b.Run("Booleans", func(b *testing.B) {
		type BoolQuery struct {
			Active bool `query:"active"`
		}
		req := httptest.NewRequest(http.MethodGet, "/?active=true", nil)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			var result BoolQuery
			_ = B.Query(req, &result)
		}
	})

	b.Run("StringSlices", func(b *testing.B) {
		type SliceQuery struct {
			Tags []string `query:"tags"`
		}
		req := httptest.NewRequest(http.MethodGet, "/?tags=go&tags=web&tags=api", nil)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			var result SliceQuery
			_ = B.Query(req, &result)
		}
	})

	b.Run("IntSlices", func(b *testing.B) {
		type IntSliceQuery struct {
			IDs []int `query:"ids"`
		}
		req := httptest.NewRequest(http.MethodGet, "/?ids=1&ids=2&ids=3&ids=4&ids=5", nil)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			var result IntSliceQuery
			_ = B.Query(req, &result)
		}
	})
}

// BenchmarkBinder_MultipartForm measures multipart form binding performance
func BenchmarkBinder_MultipartForm(b *testing.B) {
	type MultipartData struct {
		Name  string `form:"name"`
		Email string `form:"email"`
		Age   int    `form:"age"`
	}

	b.Run("FormValuesOnly", func(b *testing.B) {
		// Pre-build multipart body
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		_ = writer.WriteField("name", "John")
		_ = writer.WriteField("email", "john@example.com")
		_ = writer.WriteField("age", "30")
		_ = writer.Close()

		contentType := writer.FormDataContentType()

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body.Bytes()))
			req.Header.Set(httpx.HeaderContentType, contentType)
			var result MultipartData
			_ = B.MultipartForm(req, &result, 32<<20)
		}
	})

	b.Run("SingleFile", func(b *testing.B) {
		type WithFile struct {
			Name     string      `form:"name"`
			Document *FileHeader `form:"document"`
		}

		// Pre-build multipart body with file
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		_ = writer.WriteField("name", "Test")
		fileWriter, _ := writer.CreateFormFile("document", "test.txt")
		_, _ = fileWriter.Write([]byte("Hello, World!"))
		_ = writer.Close()

		contentType := writer.FormDataContentType()

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body.Bytes()))
			req.Header.Set(httpx.HeaderContentType, contentType)
			var result WithFile
			_ = B.MultipartForm(req, &result, 32<<20)
		}
	})

	b.Run("MultipleFiles", func(b *testing.B) {
		type WithFiles struct {
			Name        string        `form:"name"`
			Attachments []*FileHeader `form:"attachments"`
		}

		// Pre-build multipart body with multiple files
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		_ = writer.WriteField("name", "Test")
		for i := range 3 {
			fileWriter, _ := writer.CreateFormFile("attachments", fmt.Sprintf("file%d.txt", i))
			_, _ = fmt.Fprintf(fileWriter, "Content %d", i)
		}
		_ = writer.Close()

		contentType := writer.FormDataContentType()

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body.Bytes()))
			req.Header.Set(httpx.HeaderContentType, contentType)
			var result WithFiles
			_ = B.MultipartForm(req, &result, 32<<20)
		}
	})
}

// BenchmarkBinder_MultipartForm_FileSizes measures multipart with different file sizes
func BenchmarkBinder_MultipartForm_FileSizes(b *testing.B) {
	type WithFile struct {
		Name     string      `form:"name"`
		Document *FileHeader `form:"document"`
	}

	sizes := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"10KB", 10 * 1024},
		{"100KB", 100 * 1024},
		{"1MB", 1024 * 1024},
	}

	for _, s := range sizes {
		b.Run(s.name, func(b *testing.B) {
			// Pre-build multipart body
			fileContent := make([]byte, s.size)
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			_ = writer.WriteField("name", "Test")
			fileWriter, _ := writer.CreateFormFile("document", "test.bin")
			_, _ = fileWriter.Write(fileContent)
			_ = writer.Close()

			contentType := writer.FormDataContentType()
			bodyBytes := body.Bytes()

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
				req.Header.Set(httpx.HeaderContentType, contentType)
				var result WithFile
				_ = B.MultipartForm(req, &result, 32<<20)
			}
		})
	}
}

// BenchmarkBinder_NestedStruct measures binding to nested/embedded structs
func BenchmarkBinder_NestedStruct(b *testing.B) {
	b.Run("Embedded_Form", func(b *testing.B) {
		type Embedded struct {
			Name string `form:"name"`
		}
		type Container struct {
			Embedded
			Email string `form:"email"`
		}

		data := url.Values{"name": []string{"John"}, "email": []string{"john@example.com"}}
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationFormURLEncoded)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			req.Body = io.NopCloser(strings.NewReader(data.Encode()))
			var result Container
			_ = B.Form(req, &result)
		}
	})

	b.Run("Embedded_Query", func(b *testing.B) {
		type Embedded struct {
			Page  int `query:"page"`
			Limit int `query:"limit"`
		}
		type Container struct {
			Embedded
			Search string `query:"search"`
		}

		req := httptest.NewRequest(http.MethodGet, "/?page=1&limit=20&search=test", nil)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			var result Container
			_ = B.Query(req, &result)
		}
	})

	b.Run("DeepNested_Form", func(b *testing.B) {
		type Deep struct {
			Value string `form:"value"`
		}
		type Middle struct {
			Deep
			MiddleVal string `form:"middle"`
		}
		type Container struct {
			Middle
			Top string `form:"top"`
		}

		data := url.Values{"value": []string{"deep"}, "middle": []string{"mid"}, "top": []string{"topval"}}
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set(httpx.HeaderContentType, httpx.MIMEApplicationFormURLEncoded)

		b.ReportAllocs()
		b.ResetTimer()

		for b.Loop() {
			req.Body = io.NopCloser(strings.NewReader(data.Encode()))
			var result Container
			_ = B.Form(req, &result)
		}
	})
}

// BenchmarkBinder_Concurrent measures concurrent binding performance
func BenchmarkBinder_Concurrent(b *testing.B) {
	type User struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Age   int    `json:"age"`
	}

	jsonData := `{"name":"John Doe","email":"john@example.com","age":30}`

	b.Run("JSON_Parallel", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				var result User
				_ = B.JSON(strings.NewReader(jsonData), &result)
			}
		})
	})

	b.Run("Query_Parallel", func(b *testing.B) {
		type Query struct {
			Page  int    `query:"page"`
			Limit int    `query:"limit"`
			Sort  string `query:"sort"`
		}

		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			req := httptest.NewRequest(http.MethodGet, "/?page=1&limit=20&sort=name", nil)
			for pb.Next() {
				var result Query
				_ = B.Query(req, &result)
			}
		})
	})
}
