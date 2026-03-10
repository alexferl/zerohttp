package config

// Bool returns a pointer to a bool value.
// Needed because Go doesn't allow &true. Used for *bool config fields that
// distinguish "not set" (nil) from "explicitly false".
func Bool(b bool) *bool {
	return &b
}
