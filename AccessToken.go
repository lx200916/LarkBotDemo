package main

import (
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	"sync"
	"time"
)

const UpdateDuration = time.Minute * 50

type AppAccessToken struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`

	AppAccessToken    string `json:"app_access_token"`
	TenantAccessToken string `json:"tenant_access_token"`

	Expire int `json:"expire"`
}

type TokenService struct {
	AppId     string
	AppSecret string
	//Private
	accessToken string
	rwmutex     sync.RWMutex
}

var update_client *resty.Client

func init() {
	update_client = resty.New()
	update_client.SetHostURL(fmt.Sprintf("https://%s", DeveloperHost))

}

func RequestToken() (error, string) {
	token := &AppAccessToken{}
	_, err := update_client.R().SetResult(token).SetBody(map[string]string{
		"app_id":     AppID,
		"app_secret": AppSecret,
	}).Post("/open-apis/auth/v3/tenant_access_token/internal")
	if err != nil {
		return err, ""
	}

	if token.Code != 0 || token.TenantAccessToken == "" {
		return errors.New(fmt.Sprint("tenant_access_token Update Error: ", token.Msg)), ""
	}
	return nil, token.TenantAccessToken
}
func (s *TokenService) Token() string {
	s.rwmutex.RLock()
	defer s.rwmutex.RUnlock()
	return s.accessToken
}
func (s *TokenService) getToken() {
	var ticker = time.NewTicker(UpdateDuration)

	for {
		select {
		case <-ticker.C:
			s.rwmutex.Lock()
			err, token := RequestToken()
			if err != nil {
				s.rwmutex.Unlock()
				log.Error(err)
				ticker = time.NewTicker(time.Second * 5)
			} else {
				s.accessToken = token
				s.rwmutex.Unlock()
				ticker = time.NewTicker(UpdateDuration)
			}

		}

	}

}

func (s *TokenService) Start() error {
	err, token := RequestToken()
	if err != nil {
		return err
	}
	s.accessToken = token
	go s.getToken()
	return nil
}
