# simplejwt [![Build](https://github.com/xyproto/simplejwt/actions/workflows/build.yml/badge.svg)](https://github.com/xyproto/simplejwt/actions/workflows/build.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/simplejwt)](https://goreportcard.com/report/github.com/xyproto/simplejwt) [![License](https://img.shields.io/badge/license-BSD-green.svg?style=flat)](https://raw.githubusercontent.com/xyproto/simplejwt/main/LICENSE)

A simple JWT package.

## Generate and verify JWT tokens

```go
package main

import (
    "fmt"
    "time"

    "github.com/xyproto/simplejwt"
)

func main() {
    // Set the JWT secret
    simplejwt.SetSecret("your-secret-key")

    // Generate a token
    payload := simplejwt.Payload{
        Subject: "1234567890",
        Expires: time.Now().Add(time.Hour),
    }

    token, err := simplejwt.Generate(payload, nil)
    if err != nil {
        fmt.Printf("Failed to generate token: %v\n", err)
        return
    }

    fmt.Printf("Generated token: %s\n", token)

    // Validate the token
    decodedPayload, err := simplejwt.Validate(token)
    if err != nil {
        fmt.Printf("Failed to validate token: %v\n", err)
        return
    }

    fmt.Printf("Decoded payload: %+v\n", decodedPayload)
}
```

* `Subject` is the user or system that the token is about.
* `Expires` is the expiration time of the token.
* The secret key is used when JWT tokens are generated or verified, together with the HMAC SHA256 algorithm.

This example is also available as `cmd/simple/main.go`.

## An even simpler example

```go
package main

import (
    "fmt"

    "github.com/xyproto/simplejwt"
)

func main() {
    // Set the secret that is used for generating and validating JWT tokens
    simplejwt.SetSecret("hunter1")

    // Generate a token by passing in a subject and for how many seconds the token should last
    token := simplejwt.SimpleGenerate("bob@zombo.com", 3600)
    if token == "" {
        fmt.Println("Failed to generate token")
        return
    }
    fmt.Printf("Generated token: %s\n", token)

    // Validate the token
    decodedSubject := simplejwt.SimpleValidate(token)
    if decodedSubject == "" {
        fmt.Println("Failed to validate token")
        return
    }
    fmt.Printf("Decoded payload, got subject: %s\n", decodedSubject)
}
```

## Set up a simple HTTP server

This is a simple HTTP server that can be accessed in a browser as `http://localhost:4000`.

It has the following endpoints:

* `/` - a HTML page with instructions for how to use `curl`.
* `/generate`  - for generating a JWT token.
* `/protected` - for retrieving protected data, but only if it is also given a valid JWT token.

```go
package main

import (
    "fmt"
    "net/http"
    "strings"
    "time"

    "github.com/xyproto/simplejwt"
)

func generateHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    payload := simplejwt.Payload{
        Subject: "1234567890",
        Expires: time.Now().Add(time.Hour),
    }
    token, err := simplejwt.Generate(payload, nil)
    if err != nil {
        http.Error(w, "Error generating token", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "text/plain")
    w.Write([]byte(token))
}

func protectedHandler(w http.ResponseWriter, r *http.Request) {
    authHeader := r.Header.Get("Authorization")
    if authHeader == "" {
        http.Error(w, "Authorization header not provided", http.StatusUnauthorized)
        return
    }

    token := strings.TrimPrefix(authHeader, "Bearer ")
    if token == "" {
        http.Error(w, "Token not provided", http.StatusUnauthorized)
        return
    }

    _, err := simplejwt.Validate(token)
    if err != nil {
        http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"message": "Access granted to protected data."}`))
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
    html := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Simple JWT Example</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 1rem;
        }
        pre {
            background-color: #f5f5f5;
            padding: 0.5rem;
            overflow-x: scroll;
        }
    </style>
</head>
<body>
    <h1>Simple JWT Example</h1>
    <p>Use the following curl commands to interact with the server:</p>
    <h2>1. Generate a JWT token</h2>
    <p>Send a POST request to <code>/generate</code> to generate a JWT token:</p>
    <pre>curl -X POST http://localhost:4000/generate</pre>
    <h2>2. Access protected data</h2>
    <p>Send a GET request to <code>/protected</code> with the token in the Authorization header to access protected data:</p>
    <pre>curl -H "Authorization: Bearer &lt;your_token_here&gt;" http://localhost:4000/protected</pre>
    <p>Replace <code>&lt;your_token_here&gt;</code> with the token you received from the previous command.</p>
</body>
</html>
`
    w.Header().Set("Content-Type", "text/html")
    w.Write([]byte(html))
}

func main() {
    http.HandleFunc("/", rootHandler)
    http.HandleFunc("/generate", generateHandler)
    http.HandleFunc("/protected", protectedHandler)
    fmt.Println("Server running on :4000")
    http.ListenAndServe(":4000", nil)
}
```

This example is also available as `cmd/server/main.go`.

## General info

* Version: 1.2.0
* License: BSD-3
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
