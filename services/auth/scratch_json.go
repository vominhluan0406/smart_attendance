package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-webauthn/webauthn/protocol"
)

func main() {
	options := protocol.CredentialCreation{
		Response: protocol.PublicKeyCredentialCreationOptions{
			Challenge: []byte("my-challenge"),
		},
	}
	b, _ := json.Marshal(options)
	fmt.Println(string(b))
}
