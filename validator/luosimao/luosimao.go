/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

package luosimao

import (
	"errors"
	"net/http"

	"github.com/hopeio/gox/net/http/client"
)

var Error = errors.New("captcha verification failed")

type Result struct {
	Error int    `json:"error"`
	Res   string `json:"res"`
	Msg   string `json:"msg"`
}

// CheckError returns Error when the Luosimao verification result is not "success".
func (l *Result) CheckError() error {
	if l.Res != "success" {
		return Error
	}
	return nil
}

// Verify calls the Luosimao API to validate the captcha response token from the frontend.
func Verify(reqURL, apiKey, response string) error {
	if reqURL == "" || apiKey == "" {
		// captcha is disabled when no API key is configured
		return nil
	}
	if response == "" {
		return Error
	}

	req := struct {
		ApiKey   string `json:"api_key"`
		Response string `json:"response"`
	}{
		ApiKey:   apiKey,
		Response: response,
	}
	result := new(Result)

	err := client.NewRequest(http.MethodPost, reqURL).ContentType(client.ContentTypeForm).Do(&req, result)
	if err != nil {
		return err
	}
	return result.CheckError()
}
