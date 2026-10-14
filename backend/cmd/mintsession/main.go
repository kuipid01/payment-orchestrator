// Command mintsession issues one access token for local verification.
//
// It exists so an authenticated endpoint can be exercised with curl against a
// probe instance, without a browser and without a password: the token is signed
// with the same secret and claims the running API verifies.
//
//	JWT_SECRET=… go run ./cmd/mintsession <user_id> <merchant_id>
//
// It prints only the token, and it is a development tool: nothing in the server
// imports it.
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kuipid01/payconnect/internal/platform/token"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("usage: mintsession <user_id> <merchant_id>")
	}
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET must be set")
	}
	signer, err := token.New(secret, "payconnect", "payconnect-api", 15*time.Minute)
	if err != nil {
		log.Fatalf("signer: %v", err)
	}
	tokenString, _, err := signer.Issue(os.Args[1], os.Args[2])
	if err != nil {
		log.Fatalf("issue: %v", err)
	}
	fmt.Print(tokenString)
}
