package validator

import (
	"errors"
	"testing"
)

func TestUniqueValidator(t *testing.T) {
	t.Run("unique strings", func(t *testing.T) {
		type TestUnique struct {
			Items []string `validate:"unique"`
		}
		tests := []struct {
			name    string
			items   []string
			wantErr bool
		}{
			{"empty", []string{}, false},
			{"unique items", []string{"a", "b", "c"}, false},
			{"duplicate items", []string{"a", "b", "a"}, true},
			{"single item", []string{"a"}, false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				input := TestUnique{Items: tt.items}
				err := New().Struct(&input)
				if tt.wantErr && err == nil {
					t.Errorf("expected error for duplicates in %v", tt.items)
				}
				if !tt.wantErr && err != nil {
					t.Errorf("unexpected error for %v: %v", tt.items, err)
				}
			})
		}
	})

	t.Run("unique ints", func(t *testing.T) {
		type TestUnique struct {
			Items []int `validate:"unique"`
		}
		tests := []struct {
			name    string
			items   []int
			wantErr bool
		}{
			{"empty", []int{}, false},
			{"unique items", []int{1, 2, 3}, false},
			{"duplicate items", []int{1, 2, 1}, true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				input := TestUnique{Items: tt.items}
				err := New().Struct(&input)
				if tt.wantErr && err == nil {
					t.Errorf("expected error for duplicates in %v", tt.items)
				}
				if !tt.wantErr && err != nil {
					t.Errorf("unexpected error for %v: %v", tt.items, err)
				}
			})
		}
	})

	t.Run("unique on string fails", func(t *testing.T) {
		type TestUnique struct {
			Value string `validate:"unique"`
		}
		input := TestUnique{Value: "hello"}
		err := New().Struct(&input)
		if err == nil {
			t.Error("expected error for unique on string type")
		}
	})

	t.Run("unique on empty slice", func(t *testing.T) {
		type TestUnique struct {
			Items []string `validate:"unique"`
		}
		input := TestUnique{Items: []string{}}
		err := New().Struct(&input)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("unique on struct slice fails", func(t *testing.T) {
		type Item struct {
			Name string
		}
		type TestUnique struct {
			Items []Item `validate:"unique"`
		}
		input := TestUnique{
			Items: []Item{{Name: "a"}, {Name: "a"}},
		}
		err := New().Struct(&input)
		if err == nil {
			t.Error("expected error for unique on struct slice")
		}
	})

	t.Run("unique on pointer slice compares addresses", func(t *testing.T) {
		type TestUnique struct {
			Items []*string `validate:"unique"`
		}
		a, b, c := "a", "b", "a"
		input := TestUnique{
			Items: []*string{&a, &b, &c},
		}
		err := New().Struct(&input)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("unique on map always passes", func(t *testing.T) {
		type TestUnique struct {
			Items map[string]int `validate:"unique"`
		}
		input := TestUnique{
			Items: map[string]int{"a": 1, "b": 2, "c": 3},
		}
		err := New().Struct(&input)
		if err != nil {
			t.Errorf("unexpected error for unique on map: %v", err)
		}
	})

	t.Run("unique on empty map passes", func(t *testing.T) {
		type TestUnique struct {
			Items map[string]int `validate:"unique"`
		}
		input := TestUnique{
			Items: map[string]int{},
		}
		err := New().Struct(&input)
		if err != nil {
			t.Errorf("unexpected error for unique on empty map: %v", err)
		}
	})

	t.Run("unique on nil map passes", func(t *testing.T) {
		type TestUnique struct {
			Items map[string]int `validate:"unique"`
		}
		input := TestUnique{
			Items: nil,
		}
		err := New().Struct(&input)
		if err != nil {
			t.Errorf("unexpected error for unique on nil map: %v", err)
		}
	})
}

func TestEachValidator(t *testing.T) {
	type Item struct {
		Name  string `validate:"required"`
		Price int    `validate:"min=0"`
	}

	t.Run("each with struct items", func(t *testing.T) {
		type TestEach struct {
			Items []Item `validate:"each"`
		}

		tests := []struct {
			name    string
			items   []Item
			wantErr bool
		}{
			{
				name:    "valid items",
				items:   []Item{{Name: "A", Price: 10}, {Name: "B", Price: 20}},
				wantErr: false,
			},
			{
				name:    "invalid item",
				items:   []Item{{Name: "A", Price: 10}, {Name: "", Price: -5}},
				wantErr: true,
			},
			{
				name:    "empty slice",
				items:   []Item{},
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				input := TestEach{Items: tt.items}
				err := New().Struct(&input)
				if tt.wantErr && err == nil {
					t.Error("expected error for invalid each item")
				}
				if !tt.wantErr && err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			})
		}
	})

	t.Run("each with string min/max", func(t *testing.T) {
		type TestTags struct {
			Tags []string `validate:"each,min=3,max=10"`
		}

		tests := []struct {
			name    string
			tags    []string
			wantErr bool
			errIdx  int
		}{
			{"valid tags", []string{"golang", "api", "http"}, false, -1},
			{"tag too short", []string{"go", "valid"}, true, 0},
			{"tag too long", []string{"this-is-way-too-long", "valid"}, true, 0},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				input := TestTags{Tags: tt.tags}
				err := New().Struct(&input)
				if tt.wantErr && err == nil {
					t.Errorf("expected error for invalid tags %v", tt.tags)
				}
				if tt.wantErr && tt.errIdx >= 0 {
					var ve ValidationErrors
					errors.As(err, &ve)
					fieldName := "Tags[" + string(rune('0'+tt.errIdx)) + "]"
					if errs := ve.FieldErrors(fieldName); len(errs) == 0 {
						t.Errorf("expected error on %s, got %v", fieldName, ve)
					}
				}
			})
		}
	})

	t.Run("each with email validator", func(t *testing.T) {
		type TestEmails struct {
			Emails []string `validate:"each,email"`
		}

		tests := []struct {
			name    string
			emails  []string
			wantErr bool
			errIdx  int
		}{
			{"valid emails", []string{"admin@example.com", "user@example.com"}, false, -1},
			{"invalid email", []string{"admin@example.com", "not-an-email"}, true, 1},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				input := TestEmails{Emails: tt.emails}
				err := New().Struct(&input)
				if tt.wantErr && err == nil {
					t.Error("expected error for invalid email")
				}
				if tt.wantErr && tt.errIdx >= 0 {
					var ve ValidationErrors
					errors.As(err, &ve)
					fieldName := "Emails[" + string(rune('0'+tt.errIdx)) + "]"
					if errs := ve.FieldErrors(fieldName); len(errs) == 0 {
						t.Errorf("expected error on %s, got %v", fieldName, ve)
					}
				}
			})
		}
	})

	t.Run("each with alphanum and len", func(t *testing.T) {
		type TestCodes struct {
			Codes []string `validate:"each,alphanum,len=5"`
		}

		tests := []struct {
			name    string
			codes   []string
			wantErr bool
		}{
			{"valid codes", []string{"ABC12", "XYZ99"}, false},
			{"invalid chars", []string{"ABC12", "test!"}, true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				input := TestCodes{Codes: tt.codes}
				err := New().Struct(&input)
				if tt.wantErr && err == nil {
					t.Error("expected error for non-alphanumeric code")
				}
				if !tt.wantErr && err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			})
		}
	})

	t.Run("each with array", func(t *testing.T) {
		type TestArray struct {
			Tags [3]string `validate:"each,min=2"`
		}

		tests := []struct {
			name    string
			tags    [3]string
			wantErr bool
			errIdx  int
		}{
			{"valid array", [3]string{"go", "api", "http"}, false, -1},
			{"short element", [3]string{"a", "api", "http"}, true, 0},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				input := TestArray{Tags: tt.tags}
				err := New().Struct(&input)
				if tt.wantErr && err == nil {
					t.Error("expected error for short tag in array")
				}
				if tt.wantErr && tt.errIdx >= 0 {
					var ve ValidationErrors
					errors.As(err, &ve)
					fieldName := "Tags[" + string(rune('0'+tt.errIdx)) + "]"
					if errs := ve.FieldErrors(fieldName); len(errs) == 0 {
						t.Errorf("expected error on %s, got %v", fieldName, ve)
					}
				}
			})
		}
	})

	t.Run("each with pointer elements", func(t *testing.T) {
		type PtrItem struct {
			Name string `validate:"required,min=2"`
		}
		type TestPtrSlice struct {
			Items []*PtrItem `validate:"each"`
		}

		tests := []struct {
			name    string
			items   []*PtrItem
			wantErr bool
		}{
			{"valid pointers", []*PtrItem{{Name: "item1"}, {Name: "item2"}}, false},
			{"invalid pointer", []*PtrItem{{Name: "i"}, {Name: "item2"}}, true},
			{"nil elements", []*PtrItem{nil, {Name: "item2"}}, false},
			{"all nil", []*PtrItem{nil, nil}, false},
			{"mixed nil and valid", []*PtrItem{nil, {Name: "valid"}, nil}, false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				input := TestPtrSlice{Items: tt.items}
				err := New().Struct(&input)
				if tt.wantErr && err == nil {
					t.Error("expected error for invalid pointer element")
				}
				if !tt.wantErr && err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			})
		}
	})

	t.Run("each with map values", func(t *testing.T) {
		type TestMap struct {
			Tags map[string]string `validate:"each,min=3"`
		}

		tests := []struct {
			name    string
			tags    map[string]string
			wantErr bool
		}{
			{"valid map", map[string]string{"a": "golang", "b": "python"}, false},
			{"short value", map[string]string{"a": "go", "b": "python"}, true},
			{"nil map", nil, false},
			{"empty map", map[string]string{}, false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				input := TestMap{Tags: tt.tags}
				err := New().Struct(&input)
				if tt.wantErr && err == nil {
					t.Error("expected error for short map value")
				}
				if !tt.wantErr && err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			})
		}
	})

	t.Run("each with map of structs", func(t *testing.T) {
		type ScoreEntry struct {
			Score int    `validate:"gte=0,lte=100"`
			Name  string `validate:"required"`
		}
		type TestMapStruct struct {
			Grades map[string]ScoreEntry `validate:"each"`
		}

		tests := []struct {
			name     string
			grades   map[string]ScoreEntry
			wantErr  bool
			errField string
		}{
			{
				name:    "valid struct values",
				grades:  map[string]ScoreEntry{"math": {Score: 90, Name: "Math"}, "science": {Score: 85, Name: "Science"}},
				wantErr: false,
			},
			{
				name:     "invalid score",
				grades:   map[string]ScoreEntry{"math": {Score: 105, Name: "Math"}},
				wantErr:  true,
				errField: "Grades[math].Score",
			},
			{
				name:     "missing required",
				grades:   map[string]ScoreEntry{"math": {Score: 90, Name: ""}},
				wantErr:  true,
				errField: "Grades[math].Name",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				input := TestMapStruct{Grades: tt.grades}
				err := New().Struct(&input)
				if tt.wantErr && err == nil {
					t.Error("expected error")
				}
				if tt.wantErr && tt.errField != "" {
					var ve ValidationErrors
					ok := errors.As(err, &ve)
					if !ok {
						t.Errorf("expected ValidationErrors, got %T", err)
						return
					}
					errs := ve.FieldErrors(tt.errField)
					if len(errs) == 0 {
						t.Errorf("expected error for %s, got %v", tt.errField, ve)
					}
				}
			})
		}
	})

	t.Run("each with numeric range on map", func(t *testing.T) {
		type TestMapDive struct {
			Scores map[string]int `validate:"each,gte=0,lte=100"`
		}

		tests := []struct {
			name     string
			scores   map[string]int
			wantErr  bool
			errField string
		}{
			{
				name:    "valid scores",
				scores:  map[string]int{"math": 90, "science": 85},
				wantErr: false,
			},
			{
				name:     "score too low",
				scores:   map[string]int{"math": -5, "science": 85},
				wantErr:  true,
				errField: "Scores[math]",
			},
			{
				name:     "score too high",
				scores:   map[string]int{"math": 90, "science": 105},
				wantErr:  true,
				errField: "Scores[science]",
			},
			{
				name:    "empty map",
				scores:  map[string]int{},
				wantErr: false,
			},
			{
				name:    "nil map",
				scores:  nil,
				wantErr: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				input := TestMapDive{Scores: tt.scores}
				err := New().Struct(&input)
				if tt.wantErr && err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				if tt.wantErr && tt.errField != "" {
					var ve ValidationErrors
					ok := errors.As(err, &ve)
					if !ok {
						t.Errorf("expected ValidationErrors, got %T", err)
						return
					}
					errs := ve.FieldErrors(tt.errField)
					if len(errs) == 0 {
						t.Errorf("expected error for %s, got none", tt.errField)
					}
				}
			})
		}
	})

	t.Run("each with pointer to primitive slice", func(t *testing.T) {
		type TestPtrSlice struct {
			Items []*string `validate:"each,min=2"`
		}

		str1 := "hello"
		str2 := "x"
		str3 := "world"

		tests := []struct {
			name    string
			items   []*string
			wantErr bool
		}{
			{"valid pointers", []*string{&str1, &str3}, false},
			{"invalid pointer value", []*string{&str1, &str2}, true},
			{"empty slice", []*string{}, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				input := TestPtrSlice{Items: tt.items}
				err := New().Struct(&input)
				if tt.wantErr && err == nil {
					t.Error("expected error for invalid pointer element")
				}
				if !tt.wantErr && err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			})
		}
	})
}

func TestMapValidation(t *testing.T) {
	type TestMap struct {
		Items map[string]int `validate:"min=1"`
	}

	tests := []struct {
		name    string
		items   map[string]int
		wantErr bool
	}{
		{"valid map", map[string]int{"a": 1, "b": 2}, false},
		{"empty map fails min", map[string]int{}, true},
		{"nil map fails min", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := TestMap{Items: tt.items}
			err := New().Struct(&input)
			if tt.wantErr && err == nil {
				t.Error("expected error for empty/nil map")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestArrayValidation(t *testing.T) {
	type TestArray struct {
		Items [3]int `validate:"len=3"`
	}

	input := TestArray{Items: [3]int{1, 2, 3}}
	err := New().Struct(&input)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
