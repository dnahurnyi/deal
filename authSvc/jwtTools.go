package authSvc

import (
	"fmt"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

const mySecret = "secretWordDorJWTSign"

func createToken(data string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"foo":  "bar",
		"nbf":  time.Date(2015, 10, 10, 12, 0, 0, 0, time.UTC).Unix(),
		"data": data,
	})

	// Sign and get the complete encoded token as a string using the secret
	return token.SignedString([]byte(mySecret))
}

func checkToken(tokenStr string) error {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(mySecret), nil
	})

	if err != nil {
		fmt.Println("Error parsing token:", err)
		return err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		fmt.Println(claims["foo"], claims["data"])
	} else {
		fmt.Println(err)
	}
	return err
}
