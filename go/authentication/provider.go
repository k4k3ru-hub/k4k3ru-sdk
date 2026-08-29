package authentication

import "context"

// CredentialProvider provides the current credential at request time.
type CredentialProvider interface {
	Credential(context.Context) (Credential, error)
}
