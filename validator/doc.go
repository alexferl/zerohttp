// Package validator provides struct tag-based validation with no external dependencies.
//
// # Built-in Validators
//
// Core validators:
//
//	required  - Field must not be empty/zero value
//	omitempty - Skip validation if field is empty
//	eq        - Equal to value
//	ne        - Not equal to value
//
// String validators:
//
//	min        - Minimum length (runes)
//	max        - Maximum length (runes)
//	len        - Exact length
//	contains   - Contains substring
//	startswith - Starts with prefix
//	endswith   - Ends with suffix
//	excludes   - Excludes substring
//	alpha      - Letters only (Unicode)
//	alphanum   - Letters and numbers only
//	lowercase  - All lowercase
//	uppercase  - All uppercase
//	ascii      - ASCII characters only
//	printascii - Printable ASCII only
//	numeric    - Numeric digits only
//	oneof      - One of allowed values (space-separated)
//
// Numeric validators:
//
//	min  - Minimum value
//	max  - Maximum value
//	gt   - Greater than
//	lt   - Less than
//	gte  - Greater than or equal
//	lte  - Less than or equal
//
// Format validators:
//
//	email       - Email address format
//	uuid        - UUID format
//	datetime    - Custom datetime format
//	base64      - Base64 encoded
//	hexadecimal - Hex string
//	hexcolor    - Hex color (#RGB, #RGBA, #RRGGBB, #RRGGBBAA)
//	e164        - E.164 phone number
//	semver      - Semantic version
//	jwt         - JWT format (3 base64 parts)
//	boolean     - Boolean string (true/false/yes/no/on/off/1/0)
//	json        - Valid JSON
//
// Network validators:
//
//	ip       - IP address (v4 or v6)
//	ipv4     - IPv4 address
//	ipv6     - IPv6 address
//	cidr     - CIDR notation
//	hostname - RFC 952 hostname
//	uri      - Absolute URI
//	url      - HTTP/HTTPS URL
//
// Collection validators:
//
//	unique - Unique elements in slice
//	each   - Validate each element
//
// # Usage
//
//	type User struct {
//	    Name  string `validate:"required,min=2,max=50"`
//	    Email string `validate:"required,email"`
//	    Age   int    `validate:"min=13,max=120"`
//	}
//
//	if err := validator.Struct(&user); err != nil {
//	    // Handle ValidationErrors
//	}
package validator
