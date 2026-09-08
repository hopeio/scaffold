package validator

import "regexp"

const (
	Phone = iota
	Mail
	Unknown
)
var (
	emailRegexp = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)
	phoneRegexp = regexp.MustCompile(`^1[0-9]{10}$`)
)

// PhoneOrMail detects whether input is a phone number, email address, or neither.
// Returns Phone, Mail, or Unknown.
func PhoneOrMail(input string) int {
	if phoneRegexp.MatchString(input) {
		return Phone
	}
	if emailRegexp.MatchString(input) {
		return Mail
	}
	return Unknown
}

// IsPhone reports whether input matches the Chinese mobile phone number pattern.
func IsPhone(input string) bool {
	return phoneRegexp.MatchString(input)
}
