package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type ProxyStrategy interface {
	ForwardRequest(serverScheme, serverHost string, clientRequest *http.Request) (*http.Response, error)
	ProcessResponse(*http.Response) (serverResponseStatusCode int, serverResponseHeader http.Header, serverResponseBodyReader io.Reader)
	ReturnResponse(clientResponseWriter http.ResponseWriter, serverResponseStatusCode int, serverResponseHeader http.Header, serverResponseBodyReader io.Reader)
}

//----------------------------------------------------------------------------------------------------------------------

type SimpleRequestForwarder struct{}

func (_ *SimpleRequestForwarder) ForwardRequest(serverScheme, serverHost string, clientRequest *http.Request) (*http.Response, error) {
	clientRequest.Host = serverHost
	clientRequest.URL.Host = serverHost
	clientRequest.URL.Scheme = serverScheme

	return http.DefaultTransport.RoundTrip(clientRequest)
}

type SimpleResponseProcessor struct{}

func (_ *SimpleResponseProcessor) ProcessResponse(serverResponse *http.Response) (int, http.Header, io.Reader) {
	return serverResponse.StatusCode, serverResponse.Header, serverResponse.Body
}

type SimpleResponseReturner struct{}

func (_ *SimpleResponseReturner) ReturnResponse(
	clientResponseWriter http.ResponseWriter,
	serverResponseStatusCode int,
	serverResponseHeader http.Header,
	serverResponseBodyReader io.Reader,
) {
	for headerName, headerValues := range serverResponseHeader {
		for _, headerValue := range headerValues {
			clientResponseWriter.Header().Add(headerName, headerValue)
		}
	}

	clientResponseWriter.WriteHeader(serverResponseStatusCode)
	_, bodyCopyError := io.Copy(clientResponseWriter, serverResponseBodyReader)
	if nil != bodyCopyError {
		fmt.Printf("Error copying server response body: %s \n", bodyCopyError.Error())
	}
}

//----------------------------------------------------------------------------------------------------------------------

const (
	AuthCookieName = "AuthToken"
	AuthHeaderName = "X-Auth-Token"
)

type loginResponse struct {
	Token string `json:"token"`
}

type AuthCookieToHeaderRequestForwarder struct{}

func (_ *AuthCookieToHeaderRequestForwarder) ForwardRequest(serverScheme, serverHost string, clientRequest *http.Request) (*http.Response, error) {
	clientRequest.Host = serverHost
	clientRequest.URL.Host = serverHost
	clientRequest.URL.Scheme = serverScheme

	authCookie, _ := clientRequest.Cookie(AuthCookieName)
	if nil != authCookie {
		clientRequest.Header.Add(AuthHeaderName, authCookie.Value)
	}

	return http.DefaultTransport.RoundTrip(clientRequest)
}

type AuthHeaderToCookieResponseProcessor struct{}

func (_ *AuthHeaderToCookieResponseProcessor) ProcessResponse(serverResponse *http.Response) (int, http.Header, io.Reader) {
	var serverResponseBodyReader io.Reader

	if 200 == serverResponse.StatusCode {
		bodyBytes, serverResponseBodyReadError := ioutil.ReadAll(serverResponse.Body)
		if serverResponseBodyReadError != nil {
			fmt.Printf("Error processing response: %s", serverResponseBodyReadError.Error())

			return http.StatusInternalServerError, http.Header{}, strings.NewReader("")
		}

		loginResponseObj := &loginResponse{}
		unmarshalError := json.Unmarshal(bodyBytes, loginResponseObj)
		if nil != unmarshalError {
			fmt.Printf("Error unmarshaling Servers response: %s", unmarshalError.Error())

			return http.StatusInternalServerError, http.Header{}, strings.NewReader("")
		}

		cookie := &http.Cookie{
			Name:     AuthCookieName,
			Value:    loginResponseObj.Token,
			Expires:  time.Now().AddDate(0, 0, 1),
			HttpOnly: true,
		}
		if cookieValue := cookie.String(); "" != cookieValue {
			serverResponse.Header.Add("Set-Cookie", cookieValue)
		}

		serverResponseBodyReader = bytes.NewReader(bodyBytes)
	} else {
		serverResponseBodyReader = serverResponse.Body
	}

	return serverResponse.StatusCode, serverResponse.Header, serverResponseBodyReader
}

//----------------------------------------------------------------------------------------------------------------------

type ApiProxyStrategy struct {
	AuthCookieToHeaderRequestForwarder
	SimpleResponseProcessor
	SimpleResponseReturner
}

type LoginProxyStrategy struct {
	SimpleRequestForwarder
	AuthHeaderToCookieResponseProcessor
	SimpleResponseReturner
}
