/*******************************************************************************
 * 
 */

package main

import (
	"fmt"
	"net/http"
	"net/url"
	"io"
	"strings"
	"crypto/tls"
	"crypto/x509"
	"time"
	"errors"
)

type AuthService struct {
	Service string
	Sessions map[string]*Credentials  // map session key to Credentials.
	AuthServerName string
	AuthPort int
	AuthClient *http.Client
}

/*******************************************************************************
 * 
 */
func NewAuthService(serviceName string, authServerName string, authPort int,
	certPool *x509.CertPool) *AuthService {
	return &AuthService{
		Service: serviceName,
		Sessions: make(map[string]*Credentials),
		AuthServerName: authServerName,
		AuthPort: authPort,
		AuthClient: connectToAuthServer(certPool),
	}
}

/*******************************************************************************
 * 
 */
func (authSvc *AuthService) authenticateRequest(httpReq *http.Request) *SessionToken {
	var sessionToken *SessionToken = nil
	
	/***********************
	Commented out until I complete the authentication mechanism with Cesanta.
	var sessionId = getSessionId(httpReq)
	if sessionId != "" {
		sessionToken = authSvc.validateSessionId(sessionId)
	}
	if sessionToken == nil { // authenticate basic credentials
		var creds *Credentials = getSessionBasicAuthCreds(httpReq)
		if creds != nil {
			sessionToken = authSvc.authenticated(creds)
		}
	}
	***********************/
	// Temporary code - 
	var sessionId string = authSvc.createUniqueSessionId()
	sessionToken = NewSessionToken(sessionId, "testuser1")
	//........Remove the above two lines!!!!!!!!!
	
	return sessionToken
}

/*******************************************************************************
 * Verify that the credentials match a registered user. If so, return a session
 * token that can be used to validate subsequent requests.
 */
func (authSvc *AuthService) authenticated(creds *Credentials) *SessionToken {
	
	// Access the auth server to authenticate the credentials.
	if ! authSvc.sendQueryToAuthServer(creds, authSvc.Service,
		creds.userid, "", "", []string{}) { return nil }
	
	var sessionId string = authSvc.createUniqueSessionId()
	var token *SessionToken = NewSessionToken(sessionId, creds.userid)
	
	// Cache the new session token, so that this Server can recognize it in future
	// exchanges during this session.
	authSvc.Sessions[sessionId] = creds
	
	return token
}

/*******************************************************************************
 * Check if the specified account is allowed to have access to the specified
 * resource. This function does not authenticate the
 * account - that is done by authenticated().
 * https://stackoverflow.com/questions/24496344/golang-send-http-request-with-certificate
 */
func (authSvc *AuthService) authorized(creds *Credentials, account string, 
	scope_type string, scope_name string, scope_actions []string) bool {

	return true
	//....Remove!!!!!!!!!!!!!!


//	return authSvc.sendQueryToAuthServer(creds, authSvc.Service,
//		creds.userid, scope_type, scope_name, scope_actions)
}




/***************************** Internal Functions ******************************
 *******************************************************************************
 * Returns the session id header value, or "" if there is none.
 */
func getSessionId(httpReq *http.Request) string {
	var sessionId string = httpReq.Header["SessionId"][0]
	if len(sessionId) == 0 { return "" }
	if len(sessionId) > 1 { panic(errors.New("Ill-formed session id")) }
	return sessionId
}

/*******************************************************************************
 * Return the userid and password from the HTTP header, or nil if not present.
 */
func getSessionBasicAuthCreds(httpReq *http.Request) *Credentials {
	userid, password, ok := httpReq.BasicAuth()
	if !ok { return nil }
	return NewCredentials(userid, password)
}

/*******************************************************************************
 * Validate the specified session id. If valid, return a SessionToken with
 * the identity of the session's owner.
 */
func (authSvc *AuthService) validateSessionId(sessionId string) *SessionToken {
	var credentials *Credentials = authSvc.Sessions[sessionId]
	if credentials == nil { return nil }
	
	return NewSessionToken(sessionId, credentials.userid)
}

/*******************************************************************************
 * Establish a TLS connection with the authentication/authorization server.
 * This connection is maintained.
 */
func connectToAuthServer(certPool *x509.CertPool) *http.Client {
	
	var tr *http.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: certPool},
		DisableCompression: true,
	}
	return &http.Client{Transport: tr}
}

/*******************************************************************************
 * Send an authentication or authorization request to the auth server. If successful,
 * return true, otherwise return false. This function encapsulates the HTTP messsage
 * format required by the auth server.
 */
func (authSvc *AuthService) sendQueryToAuthServer(creds *Credentials, 
	service string, account string,
	scope_type string, scope_name string, scope_actions []string) bool {
	
	// https://github.com/cesanta/docker_auth/blob/master/auth_server/server/server.go
	var urlstr string = fmt.Sprintf(
		"https://%s:%s/auth",
		authSvc.AuthServerName, authSvc.AuthPort)
	
	var request *http.Request
	var err error
	var actions string = strings.Join(scope_actions, ",")
	var scope string = fmt.Sprintf("%s%s%s", scope_type, scope_name, actions)
	var data url.Values = url.Values {
		"service": []string{service},
		"scope": []string{scope},
		"account": []string{creds.userid},
	}
	var reader io.Reader = strings.NewReader(data.Encode())
	
	request, err = http.NewRequest("POST", urlstr, reader)
		if err != nil { panic(err) }
	request.SetBasicAuth(creds.userid, creds.pswd)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	var resp *http.Response
	resp, err = authSvc.AuthClient.Do(request)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	if resp.StatusCode != 200 {
		fmt.Println(fmt.Sprintf("Response code %s", resp.StatusCode))
		return false
	}
	
	defer resp.Body.Close()
	
	return true
}

/*******************************************************************************
 * Return a session id that is guaranteed to be unique, and that is completely
 * opaque and unforgeable. ....To do: append a monotonically increasing value
 * (created atomically) to the string prior to encryption.
 */
func (authSvc *AuthService) createUniqueSessionId() string {
	return encrypt(time.Now().Local().String())
}

/*******************************************************************************
 * Encrypt the specified string. For now, just return the string.
 * ....To do: Need to complete this to use the Server's private key.
 */
func encrypt(s string) string {
	return s
}