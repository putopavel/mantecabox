package main

import (
	"crypto/sha512"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"mantecabox/models"
	"mantecabox/services"

	"github.com/alexflint/go-arg"
	"github.com/briandowns/spinner"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/json"
	"github.com/go-http-utils/headers"
	"github.com/hako/durafmt"
	"github.com/howeyc/gopass"
	"github.com/nbutton23/zxcvbn-go"
	"github.com/zalando/go-keyring"
	"gopkg.in/resty.v1"
)

const (
	keyringServiceName = "mantecabox"
)

func init() {
	resty.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	resty.SetHostURL("https://localhost:10443")
	resty.SetHeader(headers.ContentType, "application/json")
	resty.SetHeader(headers.Accept, "application/json")
}

func signup(credentialsFunc func() models.Credentials) error {
	fmt.Println("Welcome to mantecabox!")
	credentials := credentialsFunc()

	strength := zxcvbn.PasswordStrength(credentials.Password, []string{credentials.Email}).Score
	fmt.Printf("Password's strength: %v (out of 4).\n", strength)
	if strength <= 2 {
		return errors.New("password too guessable")
	}

	credentials.Password = hashAndEncodePassword(credentials.Password)

	err := services.ValidateCredentials(&credentials)
	if err != nil {
		return err
	}

	var result models.UserDto
	var serverError models.ServerError
	response, err := resty.R().
		SetBody(&credentials).
		SetResult(&result).
		SetError(&serverError).
		Post("/register")

	if err != nil {
		return err
	}
	if serverError.Message != "" {
		return errors.New(serverError.Message)
	}
	if response.StatusCode() != http.StatusCreated {
		return errors.New("server did not sent HTTP 201 Created status")
	}
	if result.Email != credentials.Email {
		return errors.New("username not registered properly")
	}

	fmt.Printf("User %v registered successfully!\n", result.Email)
	return nil
}

func login(credentialsFunc func() models.Credentials) error {
	fmt.Println("Nice to see you again!")
	credentials := credentialsFunc()
	credentials.Password = hashAndEncodePassword(credentials.Password)

	err := services.ValidateCredentials(&credentials)
	if err != nil {
		return err
	}

	var verificationResult models.ServerError
	var serverError models.ServerError
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Start()
	response, err := resty.R().
		SetBody(&credentials).
		SetResult(&verificationResult).
		SetError(&serverError).
		Post("/2fa-verification")
	s.Stop()

	if err != nil {
		return err
	}
	if serverError.Message != "" {
		return errors.New(serverError.Message)
	}
	if response.StatusCode() != http.StatusOK {
		return errors.New("server did not sent HTTP 200 OK status")
	}

	var twoFactorAuth string
	fmt.Println(verificationResult.Message)
	fmt.Print("Verification Code: M-")
	fmt.Scanln(&twoFactorAuth)

	var result models.JwtResponse
	response, err = resty.R().
		SetBody(gin.H{"username": credentials.Email, "password": credentials.Password}).
		SetQueryParam("verification_code", twoFactorAuth).
		SetResult(&result).
		SetError(&serverError).
		Post("/login")

	if err != nil {
		return err
	}
	if serverError.Message != "" {
		return errors.New(serverError.Message)
	}
	if response.StatusCode() != http.StatusOK {
		return errors.New("server did not sent HTTP 200 OK status")
	}

	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	err = keyring.Set(keyringServiceName, credentials.Email, string(bytes))
	if err != nil {
		return err
	}

	fmt.Printf("Successfully logged for %v", durafmt.ParseShort(result.Expire.Sub(time.Now())))
	return nil
}

func hashAndEncodePassword(password string) string {
	sum512 := sha512.Sum512([]byte(password))
	uppercasedHash := strings.ToUpper(hex.EncodeToString(sum512[:]))
	return base64.URLEncoding.EncodeToString([]byte(uppercasedHash))
}

func readCredentials() models.Credentials {
	var credentials models.Credentials
	fmt.Print("Email: ")
	fmt.Scanln(&credentials.Email)
	fmt.Print("Password: ")
	pass, err := gopass.GetPasswdMasked()
	if err != nil {
		panic(err)
	}
	credentials.Password = string(pass)
	return credentials
}

func main() {
	var args struct {
		Operation string `arg:"positional, required" help:"<signup>, <login>, <help>"`
	}
	parser := arg.MustParse(&args)
	if args.Operation == "signup" {
		err := signup(readCredentials)
		if err != nil {
			fmt.Fprintf(os.Stderr, "An error ocurred during signup: %v\n", err)
		}
	} else if args.Operation == "login" {
		err := login(readCredentials)
		if err != nil {
			fmt.Fprintf(os.Stderr, "An error ocurred during login: %v\n", err)
		}
	} else if args.Operation == "help" {
		parser.WriteHelp(os.Stdin)
	} else {
		parser.Fail(fmt.Sprintf(`Operation "%v" not recognized`, args.Operation))
	}
}