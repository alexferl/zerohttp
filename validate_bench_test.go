package zerohttp

import (
	"fmt"
	"reflect"
	"testing"
)

// BenchmarkValidator_SimpleStruct measures basic struct validation overhead.
func BenchmarkValidator_SimpleStruct(b *testing.B) {
	type SimpleUser struct {
		Name  string `validate:"required"`
		Email string `validate:"required,email"`
		Age   int    `validate:"required,min=18,max=120"`
	}

	validUser := SimpleUser{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   30,
	}

	validator := NewValidator()

	b.Run("Valid", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = validator.Struct(validUser)
		}
	})

	b.Run("Invalid", func(b *testing.B) {
		invalidUser := SimpleUser{
			Name:  "",
			Email: "invalid-email",
			Age:   10,
		}

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = validator.Struct(invalidUser)
		}
	})
}

// BenchmarkValidator_FieldCount measures validation with different field counts.
func BenchmarkValidator_FieldCount(b *testing.B) {
	b.Run("5Fields", func(b *testing.B) {
		type SmallStruct struct {
			F1 string `validate:"required"`
			F2 string `validate:"required"`
			F3 int    `validate:"required,min=0"`
			F4 int    `validate:"required,max=100"`
			F5 string `validate:"email"`
		}

		s := SmallStruct{
			F1: "val1",
			F2: "val2",
			F3: 50,
			F4: 50,
			F5: "test@example.com",
		}

		validator := NewValidator()

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = validator.Struct(s)
		}
	})

	b.Run("20Fields", func(b *testing.B) {
		type LargeStruct struct {
			F1  string `validate:"required"`
			F2  string `validate:"required"`
			F3  string `validate:"required"`
			F4  string `validate:"required"`
			F5  string `validate:"required"`
			F6  int    `validate:"required,min=0"`
			F7  int    `validate:"required,min=0"`
			F8  int    `validate:"required,min=0"`
			F9  int    `validate:"required,min=0"`
			F10 int    `validate:"required,min=0"`
			F11 string `validate:"email"`
			F12 string `validate:"email"`
			F13 string `validate:"email"`
			F14 string `validate:"email"`
			F15 string `validate:"email"`
			F16 string `validate:"url"`
			F17 string `validate:"url"`
			F18 string `validate:"url"`
			F19 string `validate:"url"`
			F20 string `validate:"url"`
		}

		s := LargeStruct{
			F1: "val1", F2: "val2", F3: "val3", F4: "val4", F5: "val5",
			F6: 10, F7: 20, F8: 30, F9: 40, F10: 50,
			F11: "a@b.com", F12: "c@d.com", F13: "e@f.com", F14: "g@h.com", F15: "i@j.com",
			F16: "http://a.com", F17: "http://b.com", F18: "http://c.com", F19: "http://d.com", F20: "http://e.com",
		}

		validator := NewValidator()

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = validator.Struct(s)
		}
	})
}

// BenchmarkValidator_NestedStruct measures nested struct validation performance.
func BenchmarkValidator_NestedStruct(b *testing.B) {
	type Address struct {
		Street string `validate:"required"`
		City   string `validate:"required"`
		Zip    string `validate:"required,len=5"`
	}

	type Person struct {
		Name    string  `validate:"required"`
		Email   string  `validate:"required,email"`
		Address Address `validate:"required"`
	}

	type Company struct {
		Name    string  `validate:"required"`
		CEO     Person  `validate:"required"`
		CTO     Person  `validate:"required"`
		Address Address `validate:"required"`
	}

	company := Company{
		Name: "Acme Inc",
		CEO: Person{
			Name:  "John Doe",
			Email: "john@acme.com",
			Address: Address{
				Street: "123 Main St",
				City:   "New York",
				Zip:    "10001",
			},
		},
		CTO: Person{
			Name:  "Jane Smith",
			Email: "jane@acme.com",
			Address: Address{
				Street: "456 Tech Ave",
				City:   "San Francisco",
				Zip:    "94102",
			},
		},
		Address: Address{
			Street: "789 Corp Blvd",
			City:   "Chicago",
			Zip:    "60601",
		},
	}

	validator := NewValidator()

	b.Run("Shallow_1Level", func(b *testing.B) {
		person := company.CEO

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = validator.Struct(person)
		}
	})

	b.Run("Deep_3Levels", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = validator.Struct(company)
		}
	})
}

// BenchmarkValidator_SliceValidation measures slice/collection validation.
func BenchmarkValidator_SliceValidation(b *testing.B) {
	type Tag struct {
		Name string `validate:"required"`
	}

	type Post struct {
		Title string `validate:"required"`
		Tags  []Tag  `validate:"dive"`
	}

	type Blog struct {
		Posts []Post `validate:"dive"`
	}

	b.Run("SmallSlice_5Items", func(b *testing.B) {
		tags := make([]Tag, 5)
		for i := range 5 {
			tags[i] = Tag{Name: fmt.Sprintf("tag%d", i)}
		}
		post := Post{Title: "Test Post", Tags: tags}

		validator := NewValidator()

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = validator.Struct(post)
		}
	})

	b.Run("LargeSlice_100Items", func(b *testing.B) {
		tags := make([]Tag, 100)
		for i := range 100 {
			tags[i] = Tag{Name: fmt.Sprintf("tag%d", i)}
		}
		post := Post{Title: "Test Post", Tags: tags}

		validator := NewValidator()

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = validator.Struct(post)
		}
	})

	b.Run("NestedSlice", func(b *testing.B) {
		posts := make([]Post, 10)
		for i := range 10 {
			tags := make([]Tag, 5)
			for j := range 5 {
				tags[j] = Tag{Name: fmt.Sprintf("tag%d-%d", i, j)}
			}
			posts[i] = Post{Title: fmt.Sprintf("Post %d", i), Tags: tags}
		}
		blog := Blog{Posts: posts}

		validator := NewValidator()

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = validator.Struct(blog)
		}
	})
}

// BenchmarkValidator_IndividualValidators measures specific validator performance.
func BenchmarkValidator_IndividualValidators(b *testing.B) {
	validator := NewValidator()

	type RequiredField struct {
		Value string `validate:"required"`
	}

	type EmailField struct {
		Value string `validate:"email"`
	}

	type URLField struct {
		Value string `validate:"url"`
	}

	type UUIDField struct {
		Value string `validate:"uuid"`
	}

	type NumericField struct {
		Value int `validate:"min=10,max=100"`
	}

	type AlphaField struct {
		Value string `validate:"alpha"`
	}

	type AlphanumericField struct {
		Value string `validate:"alphanum"`
	}

	type LenField struct {
		Value string `validate:"len=10"`
	}

	tests := []struct {
		name     string
		value    any
		validVal any
	}{
		{"Required", RequiredField{"test"}, RequiredField{""}},
		{"Email", EmailField{"test@example.com"}, EmailField{"invalid-email"}},
		{"URL", URLField{"https://example.com"}, URLField{"not-a-url"}},
		{"UUID", UUIDField{"550e8400-e29b-41d4-a716-446655440000"}, UUIDField{"not-a-uuid"}},
		{"NumericMinMax", NumericField{50}, NumericField{5}},
		{"Alpha", AlphaField{"abcdef"}, AlphaField{"abc123"}},
		{"Alphanum", AlphanumericField{"abc123"}, AlphanumericField{"abc-123"}},
		{"Len", LenField{"1234567890"}, LenField{"short"}},
	}

	for _, tc := range tests {
		b.Run(tc.name+"_Valid", func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				_ = validator.Struct(tc.value)
			}
		})
	}
}

// BenchmarkValidator_MapValidation measures map validation performance.
func BenchmarkValidator_MapValidation(b *testing.B) {
	type Config struct {
		Settings map[string]string `validate:"required"`
	}

	b.Run("SmallMap_5Items", func(b *testing.B) {
		settings := make(map[string]string, 5)
		for i := range 5 {
			settings[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
		}
		cfg := Config{Settings: settings}

		validator := NewValidator()

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = validator.Struct(cfg)
		}
	})

	b.Run("LargeMap_100Items", func(b *testing.B) {
		settings := make(map[string]string, 100)
		for i := range 100 {
			settings[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
		}
		cfg := Config{Settings: settings}

		validator := NewValidator()

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = validator.Struct(cfg)
		}
	})
}

// BenchmarkValidator_Concurrent measures concurrent validation performance.
func BenchmarkValidator_Concurrent(b *testing.B) {
	type User struct {
		Name  string `validate:"required"`
		Email string `validate:"required,email"`
		Age   int    `validate:"min=18,max=120"`
	}

	user := User{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   30,
	}

	validator := NewValidator()

	b.Run("SharedValidator", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = validator.Struct(user)
			}
		})
	})

	b.Run("NewValidatorPerGoroutine", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			v := NewValidator()
			for pb.Next() {
				_ = v.Struct(user)
			}
		})
	})
}

// BenchmarkValidator_CustomValidator measures custom validator registration and use.
func BenchmarkValidator_CustomValidator(b *testing.B) {
	type CustomValidated struct {
		Value string `validate:"customRule"`
	}

	validator := NewValidator()
	validator.Register("customRule", func(value reflect.Value, param string) error {
		if value.String() == "forbidden" {
			return fmt.Errorf("value is forbidden")
		}
		return nil
	})

	s := CustomValidated{Value: "allowed"}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = validator.Struct(s)
	}
}

// BenchmarkValidator_OneOf measures oneof validation performance.
func BenchmarkValidator_OneOf(b *testing.B) {
	type StatusField struct {
		Status string `validate:"oneof=active inactive pending"`
	}

	validator := NewValidator()

	b.Run("MatchFirst", func(b *testing.B) {
		s := StatusField{Status: "active"}

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = validator.Struct(s)
		}
	})

	b.Run("MatchLast", func(b *testing.B) {
		s := StatusField{Status: "pending"}

		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = validator.Struct(s)
		}
	})
}

// BenchmarkValidator_PointerValidation measures pointer field validation.
func BenchmarkValidator_PointerValidation(b *testing.B) {
	name := "John"
	email := "john@example.com"
	age := 30

	type PointerFields struct {
		Name  *string `validate:"required"`
		Email *string `validate:"required,email"`
		Age   *int    `validate:"required,min=0"`
	}

	s := PointerFields{
		Name:  &name,
		Email: &email,
		Age:   &age,
	}

	validator := NewValidator()

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = validator.Struct(s)
	}
}

// BenchmarkValidator_StringValidation measures various string validators.
func BenchmarkValidator_StringValidation(b *testing.B) {
	validator := NewValidator()

	b.Run("Lowercase", func(b *testing.B) {
		type S struct {
			Field string `validate:"lowercase"`
		}
		s := S{Field: "hello world"}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = validator.Struct(s)
		}
	})

	b.Run("Uppercase", func(b *testing.B) {
		type S struct {
			Field string `validate:"uppercase"`
		}
		s := S{Field: "HELLO WORLD"}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = validator.Struct(s)
		}
	})

	b.Run("Contains", func(b *testing.B) {
		type S struct {
			Field string `validate:"contains=world"`
		}
		s := S{Field: "hello world"}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = validator.Struct(s)
		}
	})

	b.Run("ContainsAny", func(b *testing.B) {
		type S struct {
			Field string `validate:"containsany=xyz"`
		}
		s := S{Field: "hello world"}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = validator.Struct(s)
		}
	})

	b.Run("IP", func(b *testing.B) {
		type S struct {
			Field string `validate:"ip"`
		}
		s := S{Field: "192.168.1.1"}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = validator.Struct(s)
		}
	})

	b.Run("Base64", func(b *testing.B) {
		type S struct {
			Field string `validate:"base64"`
		}
		s := S{Field: "aGVsbG8="}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = validator.Struct(s)
		}
	})

	b.Run("Hexadecimal", func(b *testing.B) {
		type S struct {
			Field string `validate:"hexadecimal"`
		}
		s := S{Field: "deadbeef"}
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			_ = validator.Struct(s)
		}
	})
}
