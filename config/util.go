package config

// Bool returns a pointer to a bool value.
// Needed because Go doesn't allow &true. Used for *bool config fields that
// distinguish "not set" (nil) from "explicitly false".
func Bool(b bool) *bool {
	return &b
}

// String returns a pointer to a string value.
// Used for *string config fields that distinguish "not set" (nil) from
// "explicitly set to empty string".
func String(s string) *string {
	return &s
}

// BoolOrDefault dereferences a *bool, returning defaultVal if nil.
// Used for *bool config fields to get the value or a default.
func BoolOrDefault(b *bool, defaultVal bool) bool {
	if b == nil {
		return defaultVal
	}
	return *b
}

// StringOrDefault dereferences a *string, returning defaultVal if nil.
// Used for *string config fields to get the value or a default.
func StringOrDefault(s *string, defaultVal string) string {
	if s == nil {
		return defaultVal
	}
	return *s
}
