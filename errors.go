package main

import (
	"fmt"
	"strings"
)

// UserNameError is a base error for errors that deal with user names
type UserNameError struct {
	Name string
}

// UserFormatError is an error for when the name format does not match what is required
type UserFormatError UserNameError

func (e UserFormatError) Error() string {
	return fmt.Sprintf("\"%s\", is in an invalid format", e.Name)
}

// UserTakenError is an error for when a user tries to join with a name that is already taken
type UserTakenError UserNameError

func (e UserTakenError) Error() string {
	return fmt.Sprintf("\"%s\", is already taken", e.Name)
}

// BannedUserError is an error for when a user tries to join with a banned ip address
type BannedUserError struct {
	Host, Name string
	Names      []string
}

func (e BannedUserError) Error() string {
	return fmt.Sprintf("banned user tried to connect with IP %s: %s (banned with name(s) %s)", e.Host, e.Name, strings.Join(e.Names, ", "))
}

func newBannedUserError(host, name string, names []string) BannedUserError {
	return BannedUserError{
		Host:  host,
		Name:  name,
		Names: names,
	}
}
