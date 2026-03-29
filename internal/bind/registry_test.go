package bind

import (
	"reflect"
	"sync"
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
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
	zhtest.AssertNoError(t, err)

	// Get bindable fields
	fields := info.GetBindableFields("form", false)
	zhtest.AssertGreater(t, len(fields), 0)

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
	zhtest.AssertGreater(t, len(fields2), 0)

	// Original path should be unchanged (value type semantics)
	zhtest.AssertEqual(t, fields2[0].Path, originalPath)

	// Verify the modification was isolated to the copy
	zhtest.AssertNotEqual(t, modifiedPath, originalPath)
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
	zhtest.AssertNoError(t, err)

	fields := info.GetBindableFields("form", false)
	zhtest.AssertEqual(t, 2, len(fields))

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
		zhtest.AssertTrue(t, fieldVal.IsValid())
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
	zhtest.AssertNoError(t, err)

	fields := info.GetBindableFields("form", false)

	// Should have Name, Default, Internal (3 fields)
	// Skipped should be excluded
	zhtest.AssertEqual(t, 3, len(fields))

	// Verify correct fields are included
	fieldNames := make(map[string]bool)
	for _, f := range fields {
		fieldNames[f.Tag] = true
	}

	zhtest.AssertTrue(t, fieldNames["name"])
	zhtest.AssertFalse(t, fieldNames["-"])
	zhtest.AssertTrue(t, fieldNames["default"])
	zhtest.AssertTrue(t, fieldNames["_"])
}

func TestFileBindableFieldsAccess(t *testing.T) {
	type TestStruct struct {
		File  *FileHeader   `form:"file"`
		Files []*FileHeader `form:"files"`
		Name  string        `form:"name"`
	}

	typ := reflect.TypeOf(TestStruct{})
	info, err := TypeRegistry.GetTypeInfo(typ)
	zhtest.AssertNoError(t, err)

	fileFields := info.FileBindableFields

	zhtest.AssertEqual(t, 2, len(fileFields))

	v := reflect.ValueOf(TestStruct{})
	for _, ff := range fileFields {
		fieldVal := v.FieldByIndex(ff.Path.ToSlice())
		zhtest.AssertTrue(t, fieldVal.IsValid())
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
	zhtest.AssertNoError(t, err)

	fields := info.GetBindableFields("form", false)
	zhtest.AssertEqual(t, 1, len(fields))

	// Path should be [1, 0, 0]: C.B.A.X (B at index 0 in C, A at index 0 in B, X at index 0 in A)
	// Actually: C has B at 0, B has A at 0, A has X at 0 -> path [0, 0, 0]
	zhtest.AssertEqual(t, 3, fields[0].Path.len)

	// Verify FieldByIndex works correctly
	c := C{
		B: B{
			A: A{X: "value_x"},
		},
	}
	v := reflect.ValueOf(c)
	fieldVal := v.FieldByIndex(fields[0].Path.ToSlice())
	zhtest.AssertTrue(t, fieldVal.IsValid())
	zhtest.AssertEqual(t, "value_x", fieldVal.String())
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
	zhtest.AssertNoError(t, err)

	fields := info.GetBindableFields("form", false)

	// The exported field "Name" should be bindable, even within unexported embedded struct
	zhtest.AssertEqual(t, 1, len(fields))

	// Verify the path works with FieldByIndex
	outer := Outer{inner: inner{Name: "test_value"}}
	v := reflect.ValueOf(outer)
	fieldVal := v.FieldByIndex(fields[0].Path.ToSlice())
	zhtest.AssertTrue(t, fieldVal.IsValid())
	zhtest.AssertEqual(t, "test_value", fieldVal.String())
}

func TestFileFieldsExcludedFromNonFileBinding(t *testing.T) {
	type TestStruct struct {
		Name  string        `form:"name"`
		File  *FileHeader   `form:"file"`
		Files []*FileHeader `form:"files"`
	}

	typ := reflect.TypeOf(TestStruct{})
	info, err := TypeRegistry.GetTypeInfo(typ)
	zhtest.AssertNoError(t, err)

	// formBindableFields (allowFiles=false) should NOT include file fields
	formFields := info.formBindableFields
	for _, f := range formFields {
		zhtest.AssertNotEqual(t, "file", f.Tag)
		zhtest.AssertNotEqual(t, "files", f.Tag)
	}

	// queryBindableFields should NOT include file fields
	queryFields := info.queryBindableFields
	for _, f := range queryFields {
		zhtest.AssertNotEqual(t, "file", f.Tag)
		zhtest.AssertNotEqual(t, "files", f.Tag)
	}

	// Verify counts
	zhtest.AssertEqual(t, 1, len(formFields))
	zhtest.AssertEqual(t, "name", formFields[0].Tag)
	zhtest.AssertEqual(t, 1, len(queryFields))
	zhtest.AssertEqual(t, "name", queryFields[0].Tag)

	// POSITIVE: formWithFilesFields SHOULD include file fields
	withFilesFields := info.formWithFilesFields
	zhtest.AssertEqual(t, 3, len(withFilesFields))
	tags := make(map[string]bool)
	for _, f := range withFilesFields {
		tags[f.Tag] = true
	}
	zhtest.AssertTrue(t, tags["file"])
	zhtest.AssertTrue(t, tags["files"])
	zhtest.AssertTrue(t, tags["name"])
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
		zhtest.AssertNoError(t, err)
	}

	// All results should be identical (same pointer)
	first := results[0]
	zhtest.AssertNotNil(t, first)

	for i, result := range results {
		zhtest.AssertEqual(t, first, result)
		_ = i // Use i to avoid unused variable warning
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
	zhtest.AssertNoError(t, err)

	fields := info.GetBindableFields("form", false)
	zhtest.AssertEqual(t, 0, len(fields))
}

func TestTagWithOptionsSuffix(t *testing.T) {
	type TestStruct struct {
		WithOptions string `form:"name,omitempty"`
	}

	typ := reflect.TypeOf(TestStruct{})
	info, err := TypeRegistry.GetTypeInfo(typ)
	zhtest.AssertNoError(t, err)

	fields := info.GetBindableFields("form", false)
	zhtest.AssertEqual(t, 1, len(fields))

	// Tag options should be stripped: "name,omitempty" -> "name"
	zhtest.AssertEqual(t, "name", fields[0].Tag)
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
	zhtest.AssertNoError(t, err)

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
	zhtest.AssertTrue(t, formTags["form_field"])
	zhtest.AssertTrue(t, formTags["form_both"])
	zhtest.AssertTrue(t, formTags["neither"]) // snake_case

	// Query tags should use query tag or snake_case
	zhtest.AssertTrue(t, queryTags["query_field"])
	zhtest.AssertTrue(t, queryTags["query_both"])
	zhtest.AssertTrue(t, queryTags["neither"]) // snake_case

	// Verify form-only field uses snake_case for query (no query tag)
	zhtest.AssertFalse(t, queryTags["form_field"])
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
		zhtest.AssertNoError(t, err)
		results = append(results, info)
	}

	// All should have identical field content
	first := results[0]
	for i, info := range results {
		zhtest.AssertEqual(t, len(first.formBindableFields), len(info.formBindableFields))
		zhtest.AssertEqual(t, len(first.FileBindableFields), len(info.FileBindableFields))
		zhtest.AssertEqual(t, len(first.fields), len(info.fields))

		// Check field tags are identical
		for j, f := range info.formBindableFields {
			if j >= len(first.formBindableFields) {
				break
			}
			zhtest.AssertEqual(t, first.formBindableFields[j].Tag, f.Tag)
		}
		_ = i // Use i to avoid unused variable warning
	}
}

func TestEmptyStruct(t *testing.T) {
	type Empty struct{}

	typ := reflect.TypeOf(Empty{})
	info, err := TypeRegistry.GetTypeInfo(typ)
	zhtest.AssertNoError(t, err)

	zhtest.AssertEqual(t, 0, len(info.fields))
	zhtest.AssertEqual(t, 0, len(info.formBindableFields))
	zhtest.AssertEqual(t, 0, len(info.FileBindableFields))
}

func TestAllSkippedFields(t *testing.T) {
	type AllSkipped struct {
		A string `form:"-"`
		B string `form:"-"`
	}

	typ := reflect.TypeOf(AllSkipped{})
	info, err := TypeRegistry.GetTypeInfo(typ)
	zhtest.AssertNoError(t, err)

	zhtest.AssertEqual(t, 0, len(info.formBindableFields))
}

func TestFieldPathHelpers(t *testing.T) {
	// singleFieldPath
	fp := singleFieldPath(5)
	zhtest.AssertEqual(t, 1, fp.len)
	zhtest.AssertEqual(t, 5, fp.indices[0])

	// append
	fp2, ok := fp.append(3)
	zhtest.AssertTrue(t, ok)
	zhtest.AssertEqual(t, 2, fp2.len)
	zhtest.AssertEqual(t, 5, fp2.indices[0])
	zhtest.AssertEqual(t, 3, fp2.indices[1])

	// original unchanged
	zhtest.AssertEqual(t, 1, fp.len)

	// toSlice
	slice := fp2.ToSlice()
	zhtest.AssertEqual(t, []int{5, 3}, slice)

	// Test chaining up to 4 levels
	fp3, ok := fp2.append(7)
	zhtest.AssertTrue(t, ok)
	fp4, ok := fp3.append(9)
	zhtest.AssertTrue(t, ok)
	zhtest.AssertEqual(t, 4, fp4.len)
}

func TestFieldPathOverflow(t *testing.T) {
	// Build a path of length 4 (maximum)
	fp := singleFieldPath(0)
	var ok bool
	fp, ok = fp.append(1)
	zhtest.AssertTrue(t, ok)
	fp, ok = fp.append(2)
	zhtest.AssertTrue(t, ok)
	fp, ok = fp.append(3)
	zhtest.AssertTrue(t, ok)
	zhtest.AssertEqual(t, 4, fp.len)

	// Next append should return false (not panic)
	_, ok = fp.append(4)
	zhtest.AssertFalse(t, ok)
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
		zhtest.AssertEqual(t, tc.expected, result)
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
	zhtest.AssertNoError(t, err)

	// Should recurse into embedded Inner and find inner_field
	fields := info.GetBindableFields("form", false)
	zhtest.AssertEqual(t, 1, len(fields))
	zhtest.AssertEqual(t, "inner_field", fields[0].Tag)
}

func TestUnknownTagFallback(t *testing.T) {
	type TestStruct struct {
		A string `custom:"custom_a"`
		B string `custom:"-"`
		C string // no custom tag
	}

	typ := reflect.TypeOf(TestStruct{})
	info, err := TypeRegistry.GetTypeInfo(typ)
	zhtest.AssertNoError(t, err)

	// Unknown tag triggers computeBindableFields on the fly
	fields := info.GetBindableFields("custom", false)

	// Should find A and C (B is skipped with "-")
	zhtest.AssertEqual(t, 2, len(fields))

	tags := make(map[string]bool)
	for _, f := range fields {
		tags[f.Tag] = true
	}
	zhtest.AssertTrue(t, tags["custom_a"])
	zhtest.AssertTrue(t, tags["c"]) // snake_case default
	zhtest.AssertFalse(t, tags["-"])
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
	zhtest.AssertNoError(t, err)

	fields := info.GetBindableFields("form", false)
	zhtest.AssertEqual(t, 2, len(fields))

	tags := make(map[string]bool)
	for _, f := range fields {
		tags[f.Tag] = true
	}
	zhtest.AssertTrue(t, tags["a"])
	zhtest.AssertTrue(t, tags["b"])
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
			zhtest.AssertEqual(t, tt.expected, result)
		})
	}
}
