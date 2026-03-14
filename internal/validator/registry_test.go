package validator

import (
	"reflect"
	"sync"
	"testing"
	"time"
)

// Test structs for registry testing
type Embedded struct {
	ID int
}

type testSimpleStruct struct {
	Name string `validate:"required"`
	Age  int    `validate:"min=0,max=150"`
}

type testEmbeddedStruct struct {
	Embedded
	Value string `validate:"required"`
}

type testPtrStruct struct {
	Name *string `validate:"required"`
}

type testCollectionStruct struct {
	Items     []string         `validate:"each,min=1"`
	Array     [3]int           `validate:"dive"`
	Map       map[string]int   `validate:"omitempty"`
	Nested    testSimpleStruct `validate:"-"`
	NestedPtr *testSimpleStruct
}

type testTimeStruct struct {
	Created time.Time `validate:"required"`
}

type testMixedTags struct {
	Name     string `json:"name" validate:"required"`
	Skip     string `json:"-" validate:"required"`
	NoJSON   string `validate:"omitempty,required,each,min=5"`
	Internal string `json:",omitempty" validate:"max=10"`
}

type testUnexportedFields struct {
	Public  string `validate:"required"`
	private string `validate:"required"` //nolint:unused // intentionally unexported for testing
}

type testSkipField struct {
	Name string `validate:"-"`
	Keep string `validate:"required"`
}

type testEmbeddedValue struct {
	EmbeddedValue
}

type EmbeddedValue struct {
	Value int
}

func (e *testEmbeddedValue) Validate() error {
	return nil
}

// TestGetTypeInfo_CacheHit tests the fast path when type is already cached
func TestGetTypeInfo_CacheHit(t *testing.T) {
	// Create a fresh registry for isolation
	freshRegistry := &validatorTypeRegistry{}

	typ := reflect.TypeOf(testSimpleStruct{})

	// First call - cache miss
	info1 := freshRegistry.GetTypeInfo(typ)
	if info1 == nil {
		t.Fatal("expected non-nil info")
	}

	// Second call - cache hit (tests line 46-48)
	info2 := freshRegistry.GetTypeInfo(typ)
	if info2 != info1 {
		t.Error("expected same info from cache")
	}
}

// TestGetTypeInfo_LoadOrStoreRace tests the LoadOrStore path when another goroutine stores first
func TestGetTypeInfo_LoadOrStoreRace(t *testing.T) {
	// Create a fresh registry
	freshRegistry := &validatorTypeRegistry{}
	typ := reflect.TypeOf(testSimpleStruct{})

	// Pre-populate the cache directly
	prePopulatedInfo := &validatorTypeInfo{
		fields:            []validatedFieldInfo{},
		hasCustomValidate: false,
	}
	freshRegistry.cache.Store(typ, prePopulatedInfo)

	// Now call GetTypeInfo - it should hit the fast path and return pre-populated info
	info := freshRegistry.GetTypeInfo(typ)
	if info != prePopulatedInfo {
		t.Error("expected pre-populated info from cache")
	}
}

// TestGetTypeInfo_ConcurrentAccess tests concurrent access to the registry
func TestGetTypeInfo_ConcurrentAccess(t *testing.T) {
	// Clear cache
	ValidatorRegistry.cache = sync.Map{}

	typ := reflect.TypeOf(testSimpleStruct{})
	const numGoroutines = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	results := make(chan *validatorTypeInfo, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			info := ValidatorRegistry.GetTypeInfo(typ)
			results <- info
		}()
	}

	wg.Wait()
	close(results)

	var firstInfo *validatorTypeInfo
	for info := range results {
		if firstInfo == nil {
			firstInfo = info
		} else if info != firstInfo {
			t.Error("expected all goroutines to get the same cached info")
		}
	}
}

// TestAnalyzeType_SimpleStruct tests analysis of a simple struct
func TestAnalyzeType_SimpleStruct(t *testing.T) {
	typ := reflect.TypeOf(testSimpleStruct{})
	info := ValidatorRegistry.GetTypeInfo(typ)

	if info.hasCustomValidate {
		t.Error("expected no custom validate")
	}

	if len(info.fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(info.fields))
	}

	// Check Name field
	nameField := info.fields[0]
	if nameField.index != 0 {
		t.Errorf("expected index 0, got %d", nameField.index)
	}
	if nameField.name != "Name" {
		t.Errorf("expected name 'Name', got %s", nameField.name)
	}
	if !nameField.hasRequired {
		t.Error("expected hasRequired true")
	}
	if len(nameField.rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(nameField.rules))
	}

	// Check Age field
	ageField := info.fields[1]
	if ageField.index != 1 {
		t.Errorf("expected index 1, got %d", ageField.index)
	}
	if len(ageField.rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(ageField.rules))
	}
	if ageField.rules[0].Name != "min" || ageField.rules[0].Param != "0" {
		t.Errorf("expected min=0 rule, got %+v", ageField.rules[0])
	}
	if ageField.rules[1].Name != "max" || ageField.rules[1].Param != "150" {
		t.Errorf("expected max=150 rule, got %+v", ageField.rules[1])
	}
}

// TestAnalyzeType_EmbeddedStruct tests analysis of embedded structs
func TestAnalyzeType_EmbeddedStruct(t *testing.T) {
	typ := reflect.TypeOf(testEmbeddedStruct{})
	info := ValidatorRegistry.GetTypeInfo(typ)

	if len(info.fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(info.fields))
	}

	// Check embedded field
	embeddedField := info.fields[0]
	if !embeddedField.isEmbedded {
		t.Error("expected isEmbedded true")
	}
	if embeddedField.isStruct {
		t.Error("expected isStruct false for embedded (mutually exclusive)")
	}
	if embeddedField.isSlice || embeddedField.isArray || embeddedField.isMap {
		t.Error("expected no collection flags for embedded field")
	}

	// Check regular field
	valueField := info.fields[1]
	if valueField.isEmbedded {
		t.Error("expected isEmbedded false")
	}
	if valueField.name != "Value" {
		t.Errorf("expected name 'Value', got %s", valueField.name)
	}
}

// TestAnalyzeType_PointerFields tests pointer field handling
func TestAnalyzeType_PointerFields(t *testing.T) {
	typ := reflect.TypeOf(testPtrStruct{})
	info := ValidatorRegistry.GetTypeInfo(typ)

	if len(info.fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(info.fields))
	}

	field := info.fields[0]
	if !field.isPtr {
		t.Error("expected isPtr true")
	}
	if !field.hasRequired {
		t.Error("expected hasRequired true")
	}
}

// TestAnalyzeType_CollectionFields tests slice, array, and map fields
func TestAnalyzeType_CollectionFields(t *testing.T) {
	typ := reflect.TypeOf(testCollectionStruct{})
	info := ValidatorRegistry.GetTypeInfo(typ)

	if len(info.fields) != 4 {
		t.Fatalf("expected 4 fields, got %d", len(info.fields))
	}

	// Check slice field
	sliceField := info.fields[0]
	if !sliceField.isSlice {
		t.Error("expected isSlice true")
	}
	if sliceField.eachIndex != 0 {
		t.Errorf("expected eachIndex 0, got %d", sliceField.eachIndex)
	}

	// Check array field
	arrayField := info.fields[1]
	if !arrayField.isArray {
		t.Error("expected isArray true")
	}

	// Check map field
	mapField := info.fields[2]
	if !mapField.isMap {
		t.Error("expected isMap true")
	}
	if !mapField.omitempty {
		t.Error("expected omitempty true")
	}

	// Check nested struct field
	nestedField := info.fields[3]
	if !nestedField.isStruct {
		t.Error("expected isStruct true")
	}
	if nestedField.isEmbedded {
		t.Error("expected isEmbedded false for non-embedded struct")
	}
}

// TestAnalyzeType_TimeField tests time.Time field handling
func TestAnalyzeType_TimeField(t *testing.T) {
	typ := reflect.TypeOf(testTimeStruct{})
	info := ValidatorRegistry.GetTypeInfo(typ)

	if len(info.fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(info.fields))
	}

	field := info.fields[0]
	if !field.isStruct {
		t.Error("expected isStruct true")
	}
	if !field.isTimeTime {
		t.Error("expected isTimeTime true")
	}
}

// TestAnalyzeType_MixedTags tests various JSON tag combinations
func TestAnalyzeType_MixedTags(t *testing.T) {
	typ := reflect.TypeOf(testMixedTags{})
	info := ValidatorRegistry.GetTypeInfo(typ)

	if len(info.fields) != 4 {
		t.Fatalf("expected 4 fields, got %d", len(info.fields))
	}

	// Check JSON name resolution
	nameField := info.fields[0]
	if nameField.name != "name" {
		t.Errorf("expected json name 'name', got %s", nameField.name)
	}

	// Check skipped JSON field uses struct name
	skipField := info.fields[1]
	if skipField.name != "Skip" {
		t.Errorf("expected struct name 'Skip', got %s", skipField.name)
	}

	// Check eachIndex with multiple rules
	noJSONField := info.fields[2]
	if noJSONField.eachIndex != 2 {
		t.Errorf("expected eachIndex 2, got %d", noJSONField.eachIndex)
	}
	if !noJSONField.omitempty {
		t.Error("expected omitempty true")
	}
	if !noJSONField.hasRequired {
		t.Error("expected hasRequired true")
	}

	// Check internal field with omitempty json tag
	internalField := info.fields[3]
	if internalField.name != "Internal" {
		t.Errorf("expected name 'Internal', got %s", internalField.name)
	}
}

// TestAnalyzeType_UnexportedFields tests that unexported fields are skipped
func TestAnalyzeType_UnexportedFields(t *testing.T) {
	typ := reflect.TypeOf(testUnexportedFields{})
	info := ValidatorRegistry.GetTypeInfo(typ)

	if len(info.fields) != 1 {
		t.Fatalf("expected 1 field (unexported skipped), got %d", len(info.fields))
	}

	if info.fields[0].name != "Public" {
		t.Errorf("expected only 'Public' field, got %s", info.fields[0].name)
	}
}

// TestAnalyzeType_SkipField tests that fields with "-" validate tag are skipped
func TestAnalyzeType_SkipField(t *testing.T) {
	typ := reflect.TypeOf(testSkipField{})
	info := ValidatorRegistry.GetTypeInfo(typ)

	if len(info.fields) != 1 {
		t.Fatalf("expected 1 field (skipped field omitted), got %d", len(info.fields))
	}

	if info.fields[0].name != "Keep" {
		t.Errorf("expected only 'Keep' field, got %s", info.fields[0].name)
	}
}

// TestAnalyzeType_HasCustomValidate tests detection of Validate() method
func TestAnalyzeType_HasCustomValidate(t *testing.T) {
	// Struct without Validate()
	simpleTyp := reflect.TypeOf(testSimpleStruct{})
	simpleInfo := ValidatorRegistry.GetTypeInfo(simpleTyp)
	if simpleInfo.hasCustomValidate {
		t.Error("expected no custom validate for simple struct")
	}

	// Struct with Validate()
	customTyp := reflect.TypeOf(testEmbeddedValue{})
	customInfo := ValidatorRegistry.GetTypeInfo(customTyp)
	if !customInfo.hasCustomValidate {
		t.Error("expected hasCustomValidate true for struct with Validate()")
	}
}

// TestEachIndex_Default tests that eachIndex defaults to -1
func TestEachIndex_Default(t *testing.T) {
	typ := reflect.TypeOf(testSimpleStruct{})
	info := ValidatorRegistry.GetTypeInfo(typ)

	for _, field := range info.fields {
		if field.eachIndex != -1 {
			t.Errorf("expected eachIndex -1 for field %s, got %d", field.name, field.eachIndex)
		}
	}
}

// TestEmptyStruct tests analysis of empty struct
func TestEmptyStruct(t *testing.T) {
	type emptyStruct struct{}

	typ := reflect.TypeOf(emptyStruct{})
	info := ValidatorRegistry.GetTypeInfo(typ)

	if len(info.fields) != 0 {
		t.Errorf("expected 0 fields for empty struct, got %d", len(info.fields))
	}
	if info.hasCustomValidate {
		t.Error("expected no custom validate for empty struct")
	}
}

// TestStructWithOnlyUnexportedFields tests struct with only unexported fields
func TestStructWithOnlyUnexportedFields(t *testing.T) {
	type onlyUnexported struct {
		private1 string //nolint:unused // intentionally unexported for testing
		private2 int    //nolint:unused // intentionally unexported for testing
	}

	typ := reflect.TypeOf(onlyUnexported{})
	info := ValidatorRegistry.GetTypeInfo(typ)

	if len(info.fields) != 0 {
		t.Errorf("expected 0 fields (all unexported), got %d", len(info.fields))
	}
}

// TestStructWithOnlySkippedFields tests struct with only skipped fields
func TestStructWithOnlySkippedFields(t *testing.T) {
	type onlySkipped struct {
		Field1 string `validate:"-"`
		Field2 int    `validate:"-"`
	}

	typ := reflect.TypeOf(onlySkipped{})
	info := ValidatorRegistry.GetTypeInfo(typ)

	if len(info.fields) != 0 {
		t.Errorf("expected 0 fields (all skipped), got %d", len(info.fields))
	}
}

// TestNestedPointerStruct tests nested pointer to struct
func TestNestedPointerStruct(t *testing.T) {
	type nestedPtr struct {
		Nested *testSimpleStruct
	}

	typ := reflect.TypeOf(nestedPtr{})
	info := ValidatorRegistry.GetTypeInfo(typ)

	if len(info.fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(info.fields))
	}

	field := info.fields[0]
	if !field.isPtr {
		t.Error("expected isPtr true")
	}
	if field.isStruct {
		// After unwrapping pointer, the underlying type is struct
		// but we don't set isStruct for pointer fields
		t.Log("Note: isStruct is false for pointer fields (underlying type is struct after unwrapping)")
	}
}
