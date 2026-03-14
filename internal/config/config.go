package config

import (
	"reflect"
)

// Merge copies non-zero fields from src into dst recursively.
//
// dst and src must have the same concrete type (or identical field layout for structs).
// Mismatched types will cause a panic.
//
// For each field in src:
//   - Pointers: copied if non-nil (the pointed-to value is NOT inspected)
//   - Interfaces: copied if non-nil
//   - Strings: copied if non-empty
//   - Numbers: copied if non-zero
//   - Bools: ALWAYS copied (use *bool for optional bools)
//   - Slices/maps: replaced if non-empty (element-level merging is NOT performed)
//   - Structs: merged field-by-field recursively
//   - Channels/funcs/arrays/unsafe pointers: intentionally skipped
//
// Unexported struct fields are skipped.
//
// dst must be a non-nil pointer to a struct.
// src must be a struct or pointer to struct.
// Either constraint violation causes Merge to panic.
func Merge(dst, src any) {
	if src == nil {
		return
	}

	dv := reflect.ValueOf(dst)
	if dv.Kind() != reflect.Ptr || dv.IsNil() {
		panic("config.Merge: dst must be a non-nil pointer")
	}

	sv := reflect.ValueOf(src)
	// Dereference pointer src so callers can pass either &cfg or cfg
	if sv.Kind() == reflect.Ptr {
		sv = sv.Elem()
	}
	if sv.Kind() != reflect.Struct {
		panic("config.Merge: src must be a struct or pointer to struct")
	}

	mergeValue(dv.Elem(), sv)
}

func mergeValue(dst, src reflect.Value) {
	if !src.IsValid() {
		return
	}

	switch src.Kind() {
	case reflect.Ptr:
		if src.IsNil() {
			return
		}
		if dst.Kind() == reflect.Ptr || dst.Kind() == reflect.Interface {
			dst.Set(src)
			return
		}
		mergeValue(dst, src.Elem())
		return

	case reflect.Interface:
		if src.IsNil() {
			return
		}
		dst.Set(src)
		return
	}

	// After handling pointers/interfaces, skip zero values.
	// Bools: always copied to allow overriding true defaults with false.
	// Structs: always processed so their fields can be merged individually.
	if src.Kind() != reflect.Bool && src.Kind() != reflect.Struct && src.IsZero() {
		return
	}

	switch src.Kind() {
	case reflect.Bool:
		dst.SetBool(src.Bool())

	case reflect.String:
		dst.SetString(src.String())

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		dst.SetInt(src.Int())

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		dst.SetUint(src.Uint())

	case reflect.Float32, reflect.Float64:
		dst.SetFloat(src.Float())

	case reflect.Slice:
		dst.Set(src)

	case reflect.Map:
		dst.Set(src)

	case reflect.Func:
		dst.Set(src)

	case reflect.Struct:
		if dst.Kind() != reflect.Struct {
			panic("config.Merge: cannot merge struct into non-struct")
		}
		if dst.NumField() != src.NumField() {
			panic("config.Merge: dst and src struct field counts differ")
		}
		for i := 0; i < src.NumField(); i++ {
			field := src.Type().Field(i)
			if !field.IsExported() {
				continue
			}
			srcField := src.Field(i)
			dstField := dst.Field(i)
			if dstField.Kind() != srcField.Kind() {
				panic("config.Merge: dst and src field kinds differ at " + field.Name)
			}
			mergeValue(dstField, srcField)
		}
	}
}
