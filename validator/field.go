package validator

import "regexp"

const (
	Phone = iota
	Mail
	Unknown
)
const (
	emailPattern = `^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`
	phonePattern = `^1[0-9]{10}$`
)

// PhoneOrMail detects whether input is a phone number, email address, or neither.
// Returns Phone, Mail, or Unknown.
func PhoneOrMail(input string) int {
	phoneMatch, _ := regexp.MatchString(phonePattern, input)
	if phoneMatch {
		return Phone
	} else {
		emailMatch, _ := regexp.MatchString(emailPattern, input)
		if emailMatch {
			return Mail
		}
	}
	return Unknown
}

// IsPhone reports whether input matches the Chinese mobile phone number pattern.
func IsPhone(input string) bool {
	phoneMatch, _ := regexp.MatchString(phonePattern, input)
	return phoneMatch
}
