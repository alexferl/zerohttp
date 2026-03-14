package bind

import (
	"reflect"
	"sync"
	"testing"
)

func TestFieldPathImmutability(t *testing.T) {
	type Inner struct {
		Name string `form:"name"`
	}
	type Outer struct {
		Inner // anonymous/embedded field (field.Anonymous = true), NOT a named field with tag
	}

	typ := reflect.TypeOf(Outer{})
	info, err := TypeRegistry.GetTypeInfo(typ)
	if err != nil {
		t.Fatalf("failed to get type info: %v", err)
	}

	// Get bindable fields
	fields := info.GetBindableFields("form", false)
	if len(fields) == 0 {
		t.Fatal("expected bindable fields")
	}

	// Verify path is a value type (struct with array), not a slice
	// The fieldPath struct is immutable by value - copying it creates a full copy
	originalPath := fields[0].Path

	// Create a "modified" copy - this does not affect the original
	modifiedPath := originalPath
	if modifiedPath.len > 0 {
		modifiedPath.indices[0] = 99
	}

	// Get fields again - should be identical to original
	fields2 := info.GetBindableFields("form", false)
	if len(fields2) == 0 {
		t.Fatal("expected bindable fields on second call")
	}

	// Original path should be unchanged (value type semantics)
	if originalPath != fields2[0].Path {
		t.Error("cached paths were affected by modification of copy")
	}

	// Verify the modification was isolated to the copy
	if modifiedPath == originalPath {
		t.Error("modification should have created a different value")
	}
}

func TestEmbeddedStructFieldOrdering(t *testing.T) {
	type Embedded struct {
		FieldB string `form:"field_b"`
		FieldA string `form:"field_a"`
	}
	type Outer struct {
		Embedded // anonymous/embedded field, no tag needed for recursion
	}

	typ := reflect.TypeOf(Outer{})
	info, err := TypeRegistry.GetTypeInfo(typ)
	if err != nil {
		t.Fatalf("failed to get type info: %v", err)
	}

	fields := info.GetBindableFields("form", false)
	if len(fields) != 2 {
		t.Fatalf("expected 2 bindable fields, got %d", len(fields))
	}

	// Verify we can access fields by their computed paths
	outerVal := Outer{
		Embedded: Embedded{
			FieldA: "value_a",
			FieldB: "value_b",
		},
	}
	v := reflect.ValueOf(outerVal)

	// Both fields should be accessible via FieldByIndex
	for _, f := range fields {
		fieldVal := v.FieldByIndex(f.Path.ToSlice())
		if !fieldVal.IsValid() {
			t.Errorf("invalid field for path %+v", f.Path)
		}
	}
}

func TestTagLookupConsolidation(t *testing.T) {
	type TestStruct struct {
		Name     string `form:"name"`
		Skipped  string `form:"-"`
		Default  string // no tag
		Internal string `form:"_"`
	}

	typ := reflect.TypeOf(TestStruct{})
	info, err := TypeRegistry.GetTypeInfo(typ)
	if err != nil {
		t.Fatalf("failed to get type info: %v", err)
	}

	fields := info.GetBindableFields("form", false)

	// Should have Name, Default, Internal (3 fields)
	// Skipped should be excluded
	if len(fields) != 3 {
		t.Errorf("expected 3 bindable fields, got %d", len(fields))
	}

	// Verify correct fields are included
	fieldNames := make(map[string]bool)
	for _, f := range fields {
		fieldNames[f.Tag] = true
	}

	if !fieldNames["name"] {
		t.Error("expected 'name' field")
	}
	if fieldNames["-"] {
		t.Error("skipped field should not be present")
	}
	if !fieldNames["default"] {
		t.Error("expected 'default' field with snake_case name")
	}
	if !fieldNames["_"] {
		t.Error("expected 'internal' field")
	}
}

func TestFileBindableFieldsAccess(t *testing.T) {
	type TestStruct struct {
		File  *FileHeader   `form:"file"`
		Files []*FileHeader `form:"files"`
		Name  string        `form:"name"`
	}

	typ := reflect.TypeOf(TestStruct{})
	info, err := TypeRegistry.GetTypeInfo(typ)
	if err != nil {
		t.Fatalf("failed to get type info: %v", err)
	}

	fileFields := info.FileBindableFields

	if len(fileFields) != 2 {
		t.Errorf("expected 2 file bindable fields, got %d", len(fileFields))
	}

	v := reflect.ValueOf(TestStruct{})
	for _, ff := range fileFields {
		fieldVal := v.FieldByIndex(ff.Path.ToSlice())
		if !fieldVal.IsValid() {
			t.Errorf("invalid field for path %+v", ff.Path)
		}
	}
}

func TestThreeLevelEmbedding(t *testing.T) {
	type A struct {
		X string `form:"x"`
	}
	type B struct {
		A // embedded
	}
	type C struct {
		B // embedded
	}

	typ := reflect.TypeOf(C{})
	info, err := TypeRegistry.GetTypeInfo(typ)
	if err != nil {
		t.Fatalf("failed to get type info: %v", err)
	}

	fields := info.GetBindableFields("form", false)
	if len(fields) != 1 {
		t.Fatalf("expected 1 bindable field, got %d", len(fields))
	}

	// Path should be [1, 0, 0]: C.B.A.X (B at index 0 in C, A at index 0 in B, X at index 0 in A)
	// Actually: C has B at 0, B has A at 0, A has X at 0 -> path [0, 0, 0]
	if fields[0].Path.len != 3 {
		t.Errorf("expected path length 3, got %d", fields[0].Path.len)
	}

	// Verify FieldByIndex works correctly
	c := C{
		B: B{
			A: A{X: "value_x"},
		},
	}
	v := reflect.ValueOf(c)
	fieldVal := v.FieldByIndex(fields[0].Path.ToSlice())
	if !fieldVal.IsValid() || fieldVal.String() != "value_x" {
		t.Errorf("FieldByIndex failed: got %v", fieldVal)
	}
}

// TestUnexportedEmbeddedWithExportedFields verifies exported fields within
// unexported embedded structs are still bindable. While the embedded struct
// field itself is not settable, its exported fields are accessible via
// FieldByIndex and can be bound correctly.
func TestUnexportedEmbeddedWithExportedFields(t *testing.T) {
	type inner struct { // unexported
		Name string `form:"name"`
		_    int
	}
	type Outer struct {
		inner // unexported embedded
	}

	typ := reflect.TypeOf(Outer{})
	info, err := TypeRegistry.GetTypeInfo(typ)
	if err != nil {
		t.Fatalf("failed to get type info: %v", err)
	}

	fields := info.GetBindableFields("form", false)

	// The exported field "Name" should be bindable, even within unexported embedded struct
	if len(fields) != 1 {
		t.Errorf("expected 1 bindable field (Name promoted from inner), got %d", len(fields))
	}

	// Verify the path works with FieldByIndex
	outer := Outer{inner: inner{Name: "test_value"}}
	v := reflect.ValueOf(outer)
	fieldVal := v.FieldByIndex(fields[0].Path.ToSlice())
	if !fieldVal.IsValid() || fieldVal.String() != "test_value" {
		t.Errorf("FieldByIndex failed: got %v", fieldVal)
	}
}

func TestFileFieldsExcludedFromNonFileBinding(t *testing.T) {
	type TestStruct struct {
		Name  string        `form:"name"`
		File  *FileHeader   `form:"file"`
		Files []*FileHeader `form:"files"`
	}

	typ := reflect.TypeOf(TestStruct{})
	info, err := TypeRegistry.GetTypeInfo(typ)
	if err != nil {
		t.Fatalf("failed to get type info: %v", err)
	}

	// formBindableFields (allowFiles=false) should NOT include file fields
	formFields := info.formBindableFields
	for _, f := range formFields {
		if f.Tag == "file" || f.Tag == "files" {
			t.Error("file fields should not appear in formBindableFields")
		}
	}

	// queryBindableFields should NOT include file fields
	queryFields := info.queryBindableFields
	for _, f := range queryFields {
		if f.Tag == "file" || f.Tag == "files" {
			t.Error("file fields should not appear in queryBindableFields")
		}
	}

	// Verify counts
	if len(formFields) != 1 || formFields[0].Tag != "name" {
		t.Errorf("expected only 'name' in form fields, got %v", formFields)
	}
	if len(queryFields) != 1 || queryFields[0].Tag != "name" {
		t.Errorf("expected only 'name' in query fields, got %v", queryFields)
	}

	// POSITIVE: formWithFilesFields SHOULD include file fields
	withFilesFields := info.formWithFilesFields
	if len(withFilesFields) != 3 {
		t.Errorf("expected 3 fields with files allowed, got %d", len(withFilesFields))
	}
	tags := make(map[string]bool)
	for _, f := range withFilesFields {
		tags[f.Tag] = true
	}
	if !tags["file"] || !tags["files"] || !tags["name"] {
		t.Error("formWithFilesFields should include file, files, and name")
	}
}

// TestConcurrentRegistration verifies LoadOrStore works correctly under concurrency.
// NOTE: This test must be run with `go test -race` to validate the sync.Map guarantee.
// Without -race, it only tests pointer equality under cooperative scheduling.
func TestConcurrentRegistration(t *testing.T) {
	type TestStruct struct {
		Field string `form:"field"`
	}

	typ := reflect.TypeOf(TestStruct{})

	// Clear cache first (create fresh registry)
	freshRegistry := &typeRegistry{}

	const numGoroutines = 100
	var wg sync.WaitGroup
	results := make([]*typeInfo, numGoroutines)
	errChan := make(chan error, numGoroutines)

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			info, err := freshRegistry.GetTypeInfo(typ)
			if err != nil {
				errChan <- err
				return
			}
			results[idx] = info
		}(i)
	}
	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			t.Fatalf("failed to get type info: %v", err)
		}
	}

	// All results should be identical (same pointer)
	first := results[0]
	if first == nil {
		t.Fatal("expected non-nil typeInfo")
	}

	for i, result := range results {
		if result != first {
			t.Errorf("goroutine %d: expected same pointer, got different", i)
		}
	}
}

// TestPointerToStructEmbedded verifies pointer-to-struct embeds are excluded.
// Pointer-to-struct fields are not bindable primitives and pointer embeds
// are not expanded (Kind() == Ptr bypasses isEmbedded check), so they are skipped.
func TestPointerToStructEmbedded(t *testing.T) {
	type Inner struct {
		Name string `form:"name"`
	}
	type Outer struct {
		*Inner // pointer to struct - should be excluded
	}

	typ := reflect.TypeOf(Outer{})
	info, err := TypeRegistry.GetTypeInfo(typ)
	if err != nil {
		t.Fatalf("failed to get type info: %v", err)
	}

	fields := info.GetBindableFields("form", false)
	if len(fields) != 0 {
		t.Errorf("expected 0 fields, pointer-to-struct embed should be excluded, got %d", len(fields))
	}
}

func TestTagWithOptionsSuffix(t *testing.T) {
	type TestStruct struct {
		WithOptions string `form:"name,omitempty"`
	}

	typ := reflect.TypeOf(TestStruct{})
	info, err := TypeRegistry.GetTypeInfo(typ)
	if err != nil {
		t.Fatalf("failed to get type info: %v", err)
	}

	fields := info.GetBindableFields("form", false)
	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(fields))
	}

	// Tag options should be stripped: "name,omitempty" -> "name"
	if fields[0].Tag != "name" {
		t.Errorf("expected tag 'name' (options stripped), got '%s'", fields[0].Tag)
	}
}

func TestQueryTagIndependentFromFormTag(t *testing.T) {
	type TestStruct struct {
		OnlyForm  string `form:"form_field"`
		OnlyQuery string `query:"query_field"`
		Both      string `form:"form_both" query:"query_both"`
		Neither   string // should get snake_case for both
	}

	typ := reflect.TypeOf(TestStruct{})
	info, err := TypeRegistry.GetTypeInfo(typ)
	if err != nil {
		t.Fatalf("failed to get type info: %v", err)
	}

	// Form binding
	formFields := info.GetBindableFields("form", false)
	formTags := make(map[string]bool)
	for _, f := range formFields {
		formTags[f.Tag] = true
	}

	// Query binding
	queryFields := info.GetBindableFields("query", false)
	queryTags := make(map[string]bool)
	for _, f := range queryFields {
		queryTags[f.Tag] = true
	}

	// Form tags should use form tag or snake_case
	if !formTags["form_field"] {
		t.Error("expected 'form_field' in form tags")
	}
	if !formTags["form_both"] {
		t.Error("expected 'form_both' in form tags")
	}
	if !formTags["neither"] { // snake_case
		t.Error("expected 'neither' in form tags")
	}

	// Query tags should use query tag or snake_case
	if !queryTags["query_field"] {
		t.Error("expected 'query_field' in query tags")
	}
	if !queryTags["query_both"] {
		t.Error("expected 'query_both' in query tags")
	}
	if !queryTags["neither"] { // snake_case
		t.Error("expected 'neither' in query tags")
	}

	// Verify form-only field uses snake_case for query (no query tag)
	if queryTags["form_field"] {
		t.Error("form_field should not appear in query tags (should be 'only_form' via snake_case)")
	}
}

func TestAnalyzeTypeDeterministic(t *testing.T) {
	type TestStruct struct {
		A string      `form:"a"`
		B string      `form:"b"`
		C *FileHeader `form:"c"`
	}

	typ := reflect.TypeOf(TestStruct{})

	// Call analyzeType multiple times
	var results []*typeInfo
	for i := 0; i < 10; i++ {
		// Create fresh registry to force re-analysis
		reg := &typeRegistry{}
		info, err := reg.GetTypeInfo(typ)
		if err != nil {
			t.Fatalf("failed to get type info: %v", err)
		}
		results = append(results, info)
	}

	// All should have identical field content
	first := results[0]
	for i, info := range results {
		if len(info.formBindableFields) != len(first.formBindableFields) {
			t.Errorf("run %d: formBindableFields length mismatch", i)
		}
		if len(info.FileBindableFields) != len(first.FileBindableFields) {
			t.Errorf("run %d: fileBindableFields length mismatch", i)
		}
		if len(info.fields) != len(first.fields) {
			t.Errorf("run %d: fields length mismatch", i)
		}

		// Check field tags are identical
		for j, f := range info.formBindableFields {
			if j >= len(first.formBindableFields) {
				break
			}
			if f.Tag != first.formBindableFields[j].Tag {
				t.Errorf("run %d: field %d tag mismatch: %s vs %s", i, j, f.Tag, first.formBindableFields[j].Tag)
			}
		}
	}
}

func TestEmptyStruct(t *testing.T) {
	type Empty struct{}

	typ := reflect.TypeOf(Empty{})
	info, err := TypeRegistry.GetTypeInfo(typ)
	if err != nil {
		t.Fatalf("failed to get type info: %v", err)
	}

	if len(info.fields) != 0 {
		t.Errorf("expected 0 fields, got %d", len(info.fields))
	}
	if len(info.formBindableFields) != 0 {
		t.Errorf("expected 0 form bindable fields, got %d", len(info.formBindableFields))
	}
	if len(info.FileBindableFields) != 0 {
		t.Errorf("expected 0 file bindable fields, got %d", len(info.FileBindableFields))
	}
}

func TestAllSkippedFields(t *testing.T) {
	type AllSkipped struct {
		A string `form:"-"`
		B string `form:"-"`
	}

	typ := reflect.TypeOf(AllSkipped{})
	info, err := TypeRegistry.GetTypeInfo(typ)
	if err != nil {
		t.Fatalf("failed to get type info: %v", err)
	}

	if len(info.formBindableFields) != 0 {
		t.Errorf("expected 0 fields when all skipped, got %d", len(info.formBindableFields))
	}
}

func TestFieldPathHelpers(t *testing.T) {
	// singleFieldPath
	fp := singleFieldPath(5)
	if fp.len != 1 || fp.indices[0] != 5 {
		t.Error("singleFieldPath failed")
	}

	// append
	fp2, ok := fp.append(3)
	if !ok || fp2.len != 2 || fp2.indices[0] != 5 || fp2.indices[1] != 3 {
		t.Error("append failed")
	}

	// original unchanged
	if fp.len != 1 {
		t.Error("original path was modified")
	}

	// toSlice
	slice := fp2.ToSlice()
	if len(slice) != 2 || slice[0] != 5 || slice[1] != 3 {
		t.Errorf("toSlice returned wrong values: %v", slice)
	}

	// Test chaining up to 4 levels
	fp3, ok := fp2.append(7)
	if !ok {
		t.Error("third append failed")
	}
	fp4, ok := fp3.append(9)
	if !ok || fp4.len != 4 {
		t.Errorf("chained append failed, expected len 4, got %d", fp4.len)
	}
}

func TestFieldPathOverflow(t *testing.T) {
	// Build a path of length 4 (maximum)
	fp := singleFieldPath(0)
	var ok bool
	fp, ok = fp.append(1)
	if !ok {
		t.Fatal("first append failed")
	}
	fp, ok = fp.append(2)
	if !ok {
		t.Fatal("second append failed")
	}
	fp, ok = fp.append(3)
	if !ok {
		t.Fatal("third append failed")
	}
	if fp.len != 4 {
		t.Fatalf("expected len 4, got %d", fp.len)
	}

	// Next append should return false (not panic)
	_, ok = fp.append(4)
	if ok {
		t.Error("expected append to return false for fieldPath overflow, but got true")
	}
}

// TestSnakeCaseConversion verifies camelToSnake behavior.
// NOTE: Current implementation doesn't handle consecutive capitals well.
func TestSnakeCaseConversion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"UserName", "user_name"},
		{"Simple", "simple"},
		{"A", "a"},
		{"", ""},
		// These demonstrate the current behavior with consecutive capitals:
		{"UserID", "user_i_d"},           // NOT "user_id" - known limitation
		{"HTTPServer", "h_t_t_p_server"}, // NOT "http_server" - known limitation
		{"XMLData", "x_m_l_data"},        // NOT "xml_data" - known limitation
	}

	for _, tc := range tests {
		result := camelToSnake(tc.input)
		if result != tc.expected {
			t.Errorf("camelToSnake(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

func TestEmbeddedWithFormTag(t *testing.T) {
	type Inner struct {
		Field string `form:"inner_field"`
	}
	type Outer struct {
		Inner `form:"inner_tag"` // embedded with tag
	}

	typ := reflect.TypeOf(Outer{})
	info, err := TypeRegistry.GetTypeInfo(typ)
	if err != nil {
		t.Fatalf("failed to get type info: %v", err)
	}

	// Should recurse into embedded Inner and find inner_field
	fields := info.GetBindableFields("form", false)
	if len(fields) != 1 {
		t.Fatalf("expected 1 field from embedded Inner, got %d", len(fields))
	}
	if fields[0].Tag != "inner_field" {
		t.Errorf("expected 'inner_field', got %s", fields[0].Tag)
	}
}

func TestUnknownTagFallback(t *testing.T) {
	type TestStruct struct {
		A string `custom:"custom_a"`
		B string `custom:"-"`
		C string // no custom tag
	}

	typ := reflect.TypeOf(TestStruct{})
	info, err := TypeRegistry.GetTypeInfo(typ)
	if err != nil {
		t.Fatalf("failed to get type info: %v", err)
	}

	// Unknown tag triggers computeBindableFields on the fly
	fields := info.GetBindableFields("custom", false)

	// Should find A and C (B is skipped with "-")
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields for unknown tag, got %d", len(fields))
	}

	tags := make(map[string]bool)
	for _, f := range fields {
		tags[f.Tag] = true
	}
	if !tags["custom_a"] {
		t.Error("expected 'custom_a' field")
	}
	if !tags["c"] { // snake_case default
		t.Error("expected 'c' field (snake_case)")
	}
	if tags["-"] {
		t.Error("skipped field should not be present")
	}
}

func TestMixedEmbeddedAndRegular(t *testing.T) {
	type Inner struct {
		A string `form:"a"`
	}
	type Outer struct {
		Inner        // embedded - provides A
		B     string `form:"b"`
	}

	typ := reflect.TypeOf(Outer{})
	info, err := TypeRegistry.GetTypeInfo(typ)
	if err != nil {
		t.Fatalf("failed to get type info: %v", err)
	}

	fields := info.GetBindableFields("form", false)
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields (a from embedded + b), got %d", len(fields))
	}

	tags := make(map[string]bool)
	for _, f := range fields {
		tags[f.Tag] = true
	}
	if !tags["a"] || !tags["b"] {
		t.Errorf("expected both 'a' and 'b', got %v", tags)
	}
}

func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Name", "name"},
		{"UserName", "user_name"},
		{"EmailAddress", "email_address"},
		{"ID", "i_d"},
		{"Simple", "simple"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := camelToSnake(tt.input)
			if result != tt.expected {
				t.Errorf("camelToSnake(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
