package validator

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/zhtest"
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
	Array     [3]int           `validate:"each"`
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
	freshRegistry := &typeRegistry{}

	typ := reflect.TypeOf(testSimpleStruct{})

	// First call - cache miss
	info1 := freshRegistry.GetTypeInfo(typ)
	zhtest.AssertNotNil(t, info1)

	// Second call - cache hit (tests line 46-48)
	info2 := freshRegistry.GetTypeInfo(typ)
	zhtest.AssertEqual(t, info1, info2)
}

// TestGetTypeInfo_LoadOrStoreRace tests the LoadOrStore path when another goroutine stores first
func TestGetTypeInfo_LoadOrStoreRace(t *testing.T) {
	// Create a fresh registry
	freshRegistry := &typeRegistry{}
	typ := reflect.TypeOf(testSimpleStruct{})

	// Pre-populate the cache directly
	prePopulatedInfo := &typeInfo{
		fields:            []validatedFieldInfo{},
		hasCustomValidate: false,
	}
	freshRegistry.cache.Store(typ, prePopulatedInfo)

	// Now call GetTypeInfo - it should hit the fast path and return pre-populated info
	info := freshRegistry.GetTypeInfo(typ)
	zhtest.AssertEqual(t, prePopulatedInfo, info)
}

// TestGetTypeInfo_ConcurrentAccess tests concurrent access to the registry
func TestGetTypeInfo_ConcurrentAccess(t *testing.T) {
	// Clear cache
	Registry.cache = sync.Map{}

	typ := reflect.TypeOf(testSimpleStruct{})
	const numGoroutines = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	results := make(chan *typeInfo, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			info := Registry.GetTypeInfo(typ)
			results <- info
		}()
	}

	wg.Wait()
	close(results)

	var firstInfo *typeInfo
	for info := range results {
		if firstInfo == nil {
			firstInfo = info
		} else {
			zhtest.AssertEqual(t, firstInfo, info)
		}
	}
}

// TestAnalyzeType_SimpleStruct tests analysis of a simple struct
func TestAnalyzeType_SimpleStruct(t *testing.T) {
	typ := reflect.TypeOf(testSimpleStruct{})
	info := Registry.GetTypeInfo(typ)

	zhtest.AssertFalse(t, info.hasCustomValidate)
	zhtest.AssertEqual(t, 2, len(info.fields))

	// Check Name field
	nameField := info.fields[0]
	zhtest.AssertEqual(t, 0, nameField.index)
	zhtest.AssertEqual(t, "Name", nameField.name)
	zhtest.AssertTrue(t, nameField.hasRequired)
	zhtest.AssertEqual(t, 1, len(nameField.rules))

	// Check Age field
	ageField := info.fields[1]
	zhtest.AssertEqual(t, 1, ageField.index)
	zhtest.AssertEqual(t, 2, len(ageField.rules))
	zhtest.AssertEqual(t, "min", ageField.rules[0].Name)
	zhtest.AssertEqual(t, "0", ageField.rules[0].Param)
	zhtest.AssertEqual(t, "max", ageField.rules[1].Name)
	zhtest.AssertEqual(t, "150", ageField.rules[1].Param)
}

// TestAnalyzeType_EmbeddedStruct tests analysis of embedded structs
func TestAnalyzeType_EmbeddedStruct(t *testing.T) {
	typ := reflect.TypeOf(testEmbeddedStruct{})
	info := Registry.GetTypeInfo(typ)

	zhtest.AssertEqual(t, 2, len(info.fields))

	// Check embedded field
	embeddedField := info.fields[0]
	zhtest.AssertTrue(t, embeddedField.isEmbedded)
	zhtest.AssertFalse(t, embeddedField.isStruct)
	zhtest.AssertFalse(t, embeddedField.isSlice || embeddedField.isArray || embeddedField.isMap)

	// Check regular field
	valueField := info.fields[1]
	zhtest.AssertFalse(t, valueField.isEmbedded)
	zhtest.AssertEqual(t, "Value", valueField.name)
}

// TestAnalyzeType_PointerFields tests pointer field handling
func TestAnalyzeType_PointerFields(t *testing.T) {
	typ := reflect.TypeOf(testPtrStruct{})
	info := Registry.GetTypeInfo(typ)

	zhtest.AssertEqual(t, 1, len(info.fields))

	field := info.fields[0]
	zhtest.AssertTrue(t, field.isPtr)
	zhtest.AssertTrue(t, field.hasRequired)
}

// TestAnalyzeType_CollectionFields tests slice, array, and map fields
func TestAnalyzeType_CollectionFields(t *testing.T) {
	typ := reflect.TypeOf(testCollectionStruct{})
	info := Registry.GetTypeInfo(typ)

	zhtest.AssertEqual(t, 4, len(info.fields))

	// Check slice field
	sliceField := info.fields[0]
	zhtest.AssertTrue(t, sliceField.isSlice)
	zhtest.AssertEqual(t, 0, sliceField.eachIndex)

	// Check array field
	arrayField := info.fields[1]
	zhtest.AssertTrue(t, arrayField.isArray)

	// Check map field
	mapField := info.fields[2]
	zhtest.AssertTrue(t, mapField.isMap)
	zhtest.AssertTrue(t, mapField.omitempty)

	// Check nested struct field
	nestedField := info.fields[3]
	zhtest.AssertTrue(t, nestedField.isStruct)
	zhtest.AssertFalse(t, nestedField.isEmbedded)
}

// TestAnalyzeType_TimeField tests time.Time field handling
func TestAnalyzeType_TimeField(t *testing.T) {
	typ := reflect.TypeOf(testTimeStruct{})
	info := Registry.GetTypeInfo(typ)

	zhtest.AssertEqual(t, 1, len(info.fields))

	field := info.fields[0]
	zhtest.AssertTrue(t, field.isStruct)
	zhtest.AssertTrue(t, field.isTimeTime)
}

// TestAnalyzeType_MixedTags tests various JSON tag combinations
func TestAnalyzeType_MixedTags(t *testing.T) {
	typ := reflect.TypeOf(testMixedTags{})
	info := Registry.GetTypeInfo(typ)

	zhtest.AssertEqual(t, 4, len(info.fields))

	// Check JSON name resolution
	nameField := info.fields[0]
	zhtest.AssertEqual(t, "name", nameField.name)

	// Check skipped JSON field uses struct name
	skipField := info.fields[1]
	zhtest.AssertEqual(t, "Skip", skipField.name)

	// Check eachIndex with multiple rules
	noJSONField := info.fields[2]
	zhtest.AssertEqual(t, 2, noJSONField.eachIndex)
	zhtest.AssertTrue(t, noJSONField.omitempty)
	zhtest.AssertTrue(t, noJSONField.hasRequired)

	// Check internal field with omitempty json tag
	internalField := info.fields[3]
	zhtest.AssertEqual(t, "Internal", internalField.name)
}

// TestAnalyzeType_UnexportedFields tests that unexported fields are skipped
func TestAnalyzeType_UnexportedFields(t *testing.T) {
	typ := reflect.TypeOf(testUnexportedFields{})
	info := Registry.GetTypeInfo(typ)

	zhtest.AssertEqual(t, 1, len(info.fields))
	zhtest.AssertEqual(t, "Public", info.fields[0].name)
}

// TestAnalyzeType_SkipField tests that fields with "-" validate tag are skipped
func TestAnalyzeType_SkipField(t *testing.T) {
	typ := reflect.TypeOf(testSkipField{})
	info := Registry.GetTypeInfo(typ)

	zhtest.AssertEqual(t, 1, len(info.fields))
	zhtest.AssertEqual(t, "Keep", info.fields[0].name)
}

// TestAnalyzeType_HasCustomValidate tests detection of Validate() method
func TestAnalyzeType_HasCustomValidate(t *testing.T) {
	// Struct without Validate()
	simpleTyp := reflect.TypeOf(testSimpleStruct{})
	simpleInfo := Registry.GetTypeInfo(simpleTyp)
	zhtest.AssertFalse(t, simpleInfo.hasCustomValidate)

	// Struct with Validate()
	customTyp := reflect.TypeOf(testEmbeddedValue{})
	customInfo := Registry.GetTypeInfo(customTyp)
	zhtest.AssertTrue(t, customInfo.hasCustomValidate)
}

// TestEachIndex_Default tests that eachIndex defaults to -1
func TestEachIndex_Default(t *testing.T) {
	typ := reflect.TypeOf(testSimpleStruct{})
	info := Registry.GetTypeInfo(typ)

	for _, field := range info.fields {
		zhtest.AssertEqual(t, -1, field.eachIndex)
	}
}

// TestEmptyStruct tests analysis of empty struct
func TestEmptyStruct(t *testing.T) {
	type emptyStruct struct{}

	typ := reflect.TypeOf(emptyStruct{})
	info := Registry.GetTypeInfo(typ)

	zhtest.AssertEqual(t, 0, len(info.fields))
	zhtest.AssertFalse(t, info.hasCustomValidate)
}

// TestStructWithOnlyUnexportedFields tests struct with only unexported fields
func TestStructWithOnlyUnexportedFields(t *testing.T) {
	type onlyUnexported struct {
		private1 string //nolint:unused // intentionally unexported for testing
		private2 int    //nolint:unused // intentionally unexported for testing
	}

	typ := reflect.TypeOf(onlyUnexported{})
	info := Registry.GetTypeInfo(typ)

	zhtest.AssertEqual(t, 0, len(info.fields))
}

// TestStructWithOnlySkippedFields tests struct with only skipped fields
func TestStructWithOnlySkippedFields(t *testing.T) {
	type onlySkipped struct {
		Field1 string `validate:"-"`
		Field2 int    `validate:"-"`
	}

	typ := reflect.TypeOf(onlySkipped{})
	info := Registry.GetTypeInfo(typ)

	zhtest.AssertEqual(t, 0, len(info.fields))
}

// TestNestedPointerStruct tests nested pointer to struct
func TestNestedPointerStruct(t *testing.T) {
	type nestedPtr struct {
		Nested *testSimpleStruct
	}

	typ := reflect.TypeOf(nestedPtr{})
	info := Registry.GetTypeInfo(typ)

	zhtest.AssertEqual(t, 1, len(info.fields))

	field := info.fields[0]
	zhtest.AssertTrue(t, field.isPtr)
}
