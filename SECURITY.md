# Security policy

## Report a vulnerability

Send an email to support@klaracase.de. Use the subject line "Security: klaracase-go".
Do not open a public issue for a security problem.

Please include:

- the version of this client (`klaracase.Version`),
- the Go version,
- the steps to reproduce the problem.

Never send a real API token. If a token may have leaked, revoke it at
https://klaracase.de/settings/api-tokens and create a new one.

## What this client does with your token

- The client sends the token only as an `Authorization: Bearer` header to the base URL you set.
- The client never logs the token and never writes it to disk.
- An error message never contains the token.
