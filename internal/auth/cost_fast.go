//go:build fastbcrypt

package auth

import "golang.org/x/crypto/bcrypt"

// Only the test runs of CI build with -tags fastbcrypt: bcrypt at cost 12 takes seconds under the race detector, and
// the handler tests hash and check a password per test. A release build never carries this tag.
func init() { passwordCost = bcrypt.MinCost }
