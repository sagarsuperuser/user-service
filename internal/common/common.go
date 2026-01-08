package common

import (
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode"
)

var phoneRe = regexp.MustCompile(`^[0-9+()\-\s]{7,20}$`)

// NowFunc returns the current time. Override in tests to inject fake clocks.
type NowFunc func() time.Time

// NowUTC is the default clock implementation.
var NowUTC NowFunc = func() time.Time {
	return time.Now().UTC()
}

func ValidateEmail(email string) error {
	s := strings.TrimSpace(email)
	if s == "" {
		return nil // optional field
	}

	if len(email) > 255 {
		return errors.New("invalid email length")
	}

	_, err := mail.ParseAddress(email)
	if err != nil {
		return err
	}
	return nil
}

func ValidateTelephone(phone string) error {
	s := strings.TrimSpace(phone)
	if s == "" {
		return nil // optional field
	}

	if !phoneRe.MatchString(s) {
		return errors.New("invalid telephone format")
	}

	// Normalize → keep only digits
	digits := make([]rune, 0, len(s))
	for _, r := range s {
		if unicode.IsDigit(r) {
			digits = append(digits, r)
		}
	}

	if len(digits) < 7 || len(digits) > 15 {
		return errors.New("invalid telephone length")
	}

	return nil
}
