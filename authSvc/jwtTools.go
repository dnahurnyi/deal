package authSvc

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"io/ioutil"

	jwt "github.com/dgrijalva/jwt-go"
)

const (
	MySecret  = "secretWordDorJWTSign"
	userIdKey = "userId"
)

type TokenData struct {
	userId string
}

func loadKeys(createNewKeys bool) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	return nil, nil, nil
	if createNewKeys {
		err := createKeys()
		if err != nil {
			return nil, nil, err
		}
	}

	data, err := getFile("/usr/bin/private.pem")
	// data, err := getFile("private.pem")
	if err != nil {
		fmt.Println("Failed to get `private.pem` key file")
	}

	rKey, err := jwt.ParseRSAPrivateKeyFromPEM(data)
	if err != nil {
		fmt.Println("Failed to convert PEM file to RSA private key")
	}

	data, err = getFile("/usr/bin/public.pem")
	// data, err = getFile("public.pem")
	if err != nil {
		fmt.Println("Failed to get `public.pem` key file")
	}

	uKey, err := jwt.ParseRSAPublicKeyFromPEM(data)
	if err != nil {
		fmt.Println("Failed to convert PEM file to RSA public key")
	}

	return rKey, uKey, err
}

func createKeys() error {
	// Need to be implemented
	return nil
}

func getFile(filename string) ([]byte, error) {
	dat, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println("Error: ", err)
		return nil, err
	}
	return dat, err
}

func createToken(userID string, rKey interface{}) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		userIdKey: userID,
	})

	return token.SignedString(rKey)
}

func checkToken(uKey interface{}, tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return uKey, nil
	})
	if err != nil {
		fmt.Println("Error parsing token:", err)
		return "", err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if userId, ok := claims["userID"]; ok {
			return userId.(string), err
		}
		fmt.Println("Token is invalid")
	}
	return "", errors.New("Token is inappropriate")
}
