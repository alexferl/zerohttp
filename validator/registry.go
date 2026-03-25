package validator

import (
	"reflect"
	"sync"
	"time"
)

// validatedFieldInfo stores cached reflection information for a single struct field.
type validatedFieldInfo struct {
	index       int
	name        string           // resolved JSON name (via getJSONFieldName)
	rules       []validationRule // pre-parsed; never call parseTag at validation time
	omitempty   bool
	hasRequired bool
	eachIndex   int // index of the "each" rule in rules; -1 if absent

	isEmbedded bool
	isPtr      bool
	isStruct   bool
	isTimeTime bool
	isSlice    bool
	isArray    bool
	isMap      bool
}

// typeInfo stores cached reflection information for a struct type.
type typeInfo struct {
	fields            []validatedFieldInfo
	hasCustomValidate bool
}

// typeRegistry caches reflection information for struct types using sync.Map.
// This avoids repeated reflection overhead during struct validation.
type typeRegistry struct {
	cache sync.Map // map[reflect.Type]*typeInfo
}

// Registry is the package-level type registry instance.
var Registry = &typeRegistry{}

// GetTypeInfo retrieves cached type information for the given type.
// If the type hasn't been analyzed yet, it analyzes and caches it.
func (tr *typeRegistry) GetTypeInfo(t reflect.Type) *typeInfo {
	// Fast path: sync.Map Load
	if info, ok := tr.cache.Load(t); ok {
		return info.(*typeInfo)
	}

	// Slow path: create type info
	info := tr.analyzeType(t)

	// Store in cache (if another goroutine stored first, use that)
	if existing, loaded := tr.cache.LoadOrStore(t, info); loaded {
		return existing.(*typeInfo)
	}
	return info
}

// validateInterface is the interface for structs with custom Validate methods.
var validateInterface = reflect.TypeFor[interface{ Validate() error }]()

// timeType is the cached reflect.Type for time.Time.
var timeType = reflect.TypeOf(time.Time{})

// analyzeType analyzes a struct type and extracts all field information.
func (tr *typeRegistry) analyzeType(t reflect.Type) *typeInfo {
	info := &typeInfo{
		fields: make([]validatedFieldInfo, 0, t.NumField()),
	}

	// Check if the struct implements Validate() error
	// We check reflect.PointerTo(t) because Validate() is typically defined on the pointer receiver
	info.hasCustomValidate = reflect.PointerTo(t).Implements(validateInterface)

	// Loop over all fields
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		fi := validatedFieldInfo{
			index:     i,
			eachIndex: -1,
		}

		// Handle embedded structs
		fi.isEmbedded = field.Anonymous && field.Type.Kind() == reflect.Struct

		// Unwrap pointer if needed
		fieldType := field.Type
		if fieldType.Kind() == reflect.Ptr {
			fi.isPtr = true
			fieldType = fieldType.Elem()
		}

		// Set type flags after unwrapping pointer
		// Only set these for non-embedded fields; embedded fields have separate handling
		if !fi.isEmbedded {
			switch fieldType.Kind() {
			case reflect.Struct:
				fi.isStruct = true
				fi.isTimeTime = fieldType == timeType
			case reflect.Slice:
				fi.isSlice = true
			case reflect.Array:
				fi.isArray = true
			case reflect.Map:
				fi.isMap = true
			}
		}

		// Parse validate tag
		tag := field.Tag.Get("validate")
		if tag == "-" {
			continue // Skip fields with "-" tag
		}

		// Get JSON field name
		fi.name = getJSONFieldName(field)

		// Parse validation rules
		if tag != "" {
			fi.rules = parseTag(tag)

			// Scan rules once to set omitempty, hasRequired, and eachIndex
			for j, rule := range fi.rules {
				switch rule.Name {
				case "omitempty":
					fi.omitempty = true
				case "required":
					fi.hasRequired = true
				case "each":
					fi.eachIndex = j
				}
			}
		}

		info.fields = append(info.fields, fi)
	}

	return info
}
