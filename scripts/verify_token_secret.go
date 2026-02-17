package main

import (
	"fmt"
	"os"

	"github.com/golang-jwt/jwt"
)

func main() {
	tokenString := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6ImRhbmdxdW9jcGhvbmcxNzAzQGdtYWlsLmNvbSIsInJvbGUiOiJWSUVXRVIiLCJpc3MiOiJzbWFwLWF1dGgtc2VydmljZSIsInN1YiI6IjMzZmZjZmQ5LWY3ODItNDlhOS1hOGE2LTM1NjNiYzAyNzUwZCIsImF1ZCI6WyJzbWFwLWFwaSJdLCJleHAiOjE3NzEzODAwNTAsImlhdCI6MTc3MTM1MTI1MCwianRpIjoiMTE2MWNiZGMtOTgwZC00NWNmLTk4ZDgtNDBjNzFhNTI2NjFjIn0.SPzMfKNaJaSaL2YGeCcORNw43FcpgUUdtcOdjpbM4y4"
	secretKey := "smap-secret-key-at-least-32-chars-long"

	f, _ := os.Create("verify_result.txt")
	defer f.Close()

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		fmt.Fprintf(f, "FAIL: %v\n", err)
		fmt.Printf("FAIL: %v\n", err)
	} else if token.Valid {
		fmt.Fprintf(f, "SUCCESS: Token verified!\n")
		fmt.Println("SUCCESS: Token verified!")
	} else {
		fmt.Fprintf(f, "FAIL: Invalid token structure\n")
		fmt.Println("FAIL: Invalid token structure")
	}
}
