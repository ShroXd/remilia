package remilia

import (
	"encoding/base64"
	"time"
)

const (
	Basic  = "Basic"
	Bearer = "Bearer"
	APIKey = "apikey"
)

type AuthConfig struct {
	Type     string
	Username string
	Password string
	Token    string
	APIKey   string
}

type Token struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	Expiry       time.Time
}

type AuthManager struct {
	Config *AuthConfig
	Token  *Token
}

func (am *AuthManager) basicAuthHook() RequestHook {
	return func(req *Request) error {
		if am.Config.Type == Basic {
			auth := base64.StdEncoding.EncodeToString([]byte(am.Config.Username + ":" + am.Config.Password))
			req.Headers["Authorization"] = "Basic " + auth
		}
		return nil
	}
}

func (am *AuthManager) bearerAuthHook() RequestHook {
	return func(req *Request) error {
		if am.Config.Type == Bearer {
			req.Headers["Authorization"] = "Bearer " + am.Config.Token
		}
		return nil
	}
}

func (am *AuthManager) apiKeyAuthHook() RequestHook {
	return func(req *Request) error {
		if am.Config.Type == APIKey {
			// TODO: check if its right
			req.Headers["apikey"] = am.Config.APIKey
		}
		return nil
	}
}

func (am *AuthManager) GetAuthHook() RequestHook {
	switch am.Config.Type {
	case Basic:
		return am.basicAuthHook()
	case Bearer:
		return am.bearerAuthHook()
	case APIKey:
		return am.apiKeyAuthHook()
	default:
		// TODO: return error
		return nil
	}
}
