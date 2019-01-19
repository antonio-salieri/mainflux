// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/mainflux/mainflux/errors"
)

const (
	maxEmailLength = 254 // based on RFC3696 errata: https://www.rfc-editor.org/errata_search.php?rfc=3696&eid=1690
	maxLocalLength = 64  // based on RFC3696

	// Allowed email address character class (ascii + non-ascii(utf8))
	emailCharClass = "[[:alnum:]!#$%&'*+-/=?^_`{|}~]" + // acceptable ascii characters: alpha, numeric, subset of special characters
		"[\\x01-\\x08\\x0b\\x0c\\x0e-\\x1f\\x7f]|\\x21|[\\x23-\\x5b]|[\\x5d-\\x7e]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}]" // covers non-ascii utf8 characters
)

var (
	emailRe = regexp.MustCompile(fmt.Sprintf("^(%s)+(\\.(%s))*@(%s)+(\\.(%s))*$", emailCharClass, emailCharClass, emailCharClass, emailCharClass))
)

// User represents a Mainflux user account. Each user is identified given its
// email and password.
type User struct {
	Email    string
	Password string
	Metadata map[string]interface{}
}

// Validate returns an error if user representation is invalid.
func (u User) Validate() errors.Error {
	if u.Email == "" || u.Password == "" {
		return ErrMalformedEntity

	}

	if !isEmail(u.Email) {
		return ErrMalformedEntity
	}

	return nil
}

// UserRepository specifies an account persistence API.
type UserRepository interface {
	// Save persists the user account. A non-nil error is returned to indicate
	// operation failure.
	Save(context.Context, User) errors.Error

	// Update updates the user metadata.
	UpdateUser(context.Context, User) errors.Error

	// RetrieveByID retrieves user by its unique identifier (i.e. email).
	RetrieveByID(context.Context, string) (User, errors.Error)

	// UpdatePassword updates password for user with given email
	UpdatePassword(_ context.Context, email, password string) errors.Error
}

func isEmail(email string) bool {
	at := strings.LastIndex(email, "@")
	if at < 1 {
		return false
	}

	localLength := len(email[:at])
	hostLength := len(email[at+1:])
	maxHostLength := maxEmailLength - localLength - 1 // maxEmailLength - localLength - len("@")

	if len(email) > maxEmailLength ||
		localLength > maxLocalLength ||
		hostLength > maxHostLength {
		return false
	}

	if !emailRe.MatchString(email) {
		return false
	}

	return true
}
