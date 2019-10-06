package main

import (
	"fmt"
	"net/http"
)

type ProxyHandler struct {
	serverScheme string
	serverHost   string
	strategy     ProxyStrategy
}

func (p *ProxyHandler) ServeHTTP(clientResponseWriter http.ResponseWriter, clientRequest *http.Request) {
	serverResponse, serverResponseError := p.strategy.ForwardRequest(p.serverScheme, p.serverHost, clientRequest)
	if nil != serverResponseError {
		fmt.Printf("Error forwarding request: %s \n", serverResponseError.Error())

		responseStatusCode := http.StatusServiceUnavailable
		http.Error(clientResponseWriter, http.StatusText(responseStatusCode), responseStatusCode)

		return
	}
	defer func() {
		serverResponseBodyCloseError := serverResponse.Body.Close()
		if nil != serverResponseBodyCloseError {
			fmt.Printf("Error closing server response body: %s \n", serverResponseBodyCloseError.Error())
		}
	}()

	serverResponseStatusCode, serverResponseHeader, serverResponseBodyReader := p.strategy.ProcessResponse(serverResponse)

	p.strategy.ReturnResponse(clientResponseWriter, serverResponseStatusCode, serverResponseHeader, serverResponseBodyReader)
}

func NewProxyHandler(serverScheme, serverHost string, strategy ProxyStrategy) *ProxyHandler {
	return &ProxyHandler{
		serverScheme: serverScheme,
		serverHost:   serverHost,
		strategy:     strategy,
	}
}
