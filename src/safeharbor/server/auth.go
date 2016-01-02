/*******************************************************************************
 * Authentication and authorization.
 */

package server

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	//"crypto/tls"
	"crypto/x509"
	"time"
	"errors"
	"crypto/sha512"
	"hash"
	
	"safeharbor/apitypes"
)

type AuthService struct {
	Service string
	Sessions map[string]*apitypes.Credentials  // map session key to apitypes.Credentials.
	//DockerRegistry2AuthServerName string
	//DockerRegistry2AuthPort int
	//DockerRegistry2AuthSvc *http.Client
	secretSalt []byte
}

/*******************************************************************************
 * 
 */
func NewAuthService(serviceName string, authServerName string, authPort int,
	certPool *x509.CertPool, secretSalt string) *AuthService {

	return &AuthService{
		Service: serviceName,
		Sessions: make(map[string]*apitypes.Credentials),
		//DockerRegistry2AuthServerName: authServerName,
		//DockerRegistry2AuthPort: authPort,
		//DockerRegistry2AuthSvc: connectToAuthServer(certPool),
		secretSalt: []byte(secretSalt),
	}
}

/*******************************************************************************
 * Compute a salted hash of the specified clear text password. The hash is suitable
 * for storage and later use for validation of input passwords, using the
 * companion function PasswordHashIsValid. Thus, the hash is required to be 
 * cryptographically secure. The 512-bit SHA-2 algorithm, aka "SHA-512",
 * is used.
 */
func (authSvc *AuthService) CreatePasswordHash(pswd string) []byte {
	return authSvc.computeHash(pswd).Sum([]byte{})
}

/*******************************************************************************
 * Validate session Id: return true if valid, false otherwise. Thus, a return
 * of true indicates that the sessionId is recognized as having been created
 * by this server and that it is not expired and is still considered to represent
 * an active session.
 */
func (authSvc *AuthService) sessionIdIsValid(sessionId string) bool {
	
	return authSvc.validateSessionId(sessionId)
}

/*******************************************************************************
 * Create a new user session. This presumes that the credentials have been verified.
 */
func (authSvc *AuthService) createSession(creds *apitypes.Credentials) *apitypes.SessionToken {
	
	var sessionId string = authSvc.createUniqueSessionId()
	var token *apitypes.SessionToken = apitypes.NewSessionToken(sessionId, creds.UserId)
	
	// Cache the new session token, so that this Server can recognize it in future
	// exchanges during this session.
	authSvc.Sessions[sessionId] = creds
	fmt.Println("Created session for session id " + sessionId)
	
	return token
}

/*******************************************************************************
 * Remove the specified session Id from the set of authenticated session Ids.
 * This effectively logs out the owner of that session.
 */
func (authSvc *AuthService) invalidateSessionId(sessionId string) {
	authSvc.Sessions[sessionId] = nil
}

/*******************************************************************************
 * Clear all sessions that are cached in the auth service. The effect is that,
 * after calling this method, no user is logged in.
 */
func (authSvc *AuthService) clearAllSessions() {
	authSvc.Sessions = make(map[string]*apitypes.Credentials)
}

/*******************************************************************************
 * Verify that a request belongs to a valid session:
 * Obtain the SessionId cookie, if any, and validate it; return nil if no SessionId
 * cookie is found or the SessionId is not valid.
 */
func (authSvc *AuthService) authenticateRequestCookie(httpReq *http.Request) *apitypes.SessionToken {
	
	var sessionToken *apitypes.SessionToken = nil
	
	fmt.Println("authenticating request...")
	var sessionId = getSessionIdFromCookie(httpReq)
	fmt.Println("obtained session id:", sessionId)
	if sessionId != "" {
		sessionToken = authSvc.identifySession(sessionId)  // returns nil if invalid
	}
	
	return sessionToken
}

/*******************************************************************************
 * 
 */
func (authService *AuthService) addSessionIdToResponse(sessionToken *apitypes.SessionToken,
	writer http.ResponseWriter) {
	
	// Set cookie containing the session Id.
	var cookie = &http.Cookie{
		Name: "SessionId",
		Value: sessionToken.UniqueSessionId,
		//Path: 
		//Domain: 
		//Expires: 
		//RawExpires: 
		MaxAge: 86400,
		Secure: false,  //....change to true later.
		HttpOnly: true,
		//Raw: 
		//Unparsed: 
	}
	
	http.SetCookie(writer, cookie)
}

/*******************************************************************************
 * Determine if a specified action is allowed on a specified resource.
 * All handlers call this function.
 * The set of ACLs owned by the resource are used to make the determination.
 * At most one field of the actionMask may be true.
 */
func authorized(server *Server, sessionToken *apitypes.SessionToken, actionMask []bool,
	resourceId string) (bool, error) {

	fmt.Println("entered authorized...")  // debug
	
	
	
	/* Rules:
	
	A party can access a resource if the party,
		has an ACL entry for the resource; or,
		the resource belongs to a repo or realm for which the party has an ACL entry.
	
	In this context, a user is a party if the user is explicitly the party or if
	the user belongs to a group that is explicitly the party.
	
	Groups may not belong to other groups.
	
	The user must have the required access mode (Create, Read, Write, Exec, Delete).
	No access mode implies any other access mode.
	
	*/
	
	// Identify the user.
	var userId string = sessionToken.AuthenticatedUserid
	fmt.Println("userid=", userId)
	var user User = server.dbClient.dbGetUserByUserId(userId)
	if user == nil {
		return false, errors.New("user object cannot be identified from user id " + userId)
	}
	fmt.Println("authorized:A")  // debug
	
	// Special case: Allow user all capabilities for their own user object.
	if user.getId() == resourceId { return true, nil }
	fmt.Println("authorized:B")  // debug

	// Verify that at most one field of the actionMask is true.
	var nTrue = 0
	for _, b := range actionMask {
		if b {
			if nTrue == 1 {
				return false, errors.New("More than one field in mask may not be true")
			}
			nTrue++
		}
	}
	fmt.Println("authorized:C")  // debug
	
	// Check if the user or a group that the user belongs to has the permission
	// that is specified by the actionMask.
	var party Party = user  // start with the user.
	var resource Resource
	var err error
	resource, err = server.dbClient.getResource(resourceId)
	fmt.Println("authorized:D")  // debug
	if err != nil { return false, err }
	if resource == nil {
		return false, errors.New("Resource with Id " + resourceId + " not found")
	}
	fmt.Println("authorized:E")  // debug
	var groupIds []string = user.getGroupIds()
	var groupIndex = -1
	for { // the user, and then each group that the user belongs to...
		// See if the party (user or group) has an ACL entry for the resource.
		fmt.Println("authorized:F")  // debug
		var partyCanAccessResourceDirectoy bool
		partyCanAccessResourceDirectoy, err = server.partyHasAccess(party, actionMask, resource)
		if partyCanAccessResourceDirectoy { return true, nil }
		
		// See if any of the party's parent resources have access.
		var parentId string = resource.getParentId()
		if parentId != "" {
			var parent Resource
			parent, err = server.dbClient.getResource(parentId)
			if err != nil { return false, err }
			var parentHasAccess bool
			parentHasAccess, err = server.partyHasAccess(party, actionMask, parent)
			if parentHasAccess { return true, nil }
		}
		
		groupIndex++
		if groupIndex == len(groupIds) { return false, nil }
		var err error
		party, err = server.dbClient.getParty(groupIds[groupIndex])  // check next group
		if err != nil { return false, err }
	}
	return false, nil  // no access rights found
}

/*******************************************************************************
 * Return the SHA-512 hash of the content of the specified file. Should not be salted
 * because the hash is intended to be reproducible by third parties, given the
 * original file.
 */
func (authSvc *AuthService) ComputeFileSignature(filepath string) ([]byte, error) {
	
	var file *os.File
	var err error
	file, err = os.Open(filepath)
	if err != nil { return nil, err }
	var buf = make([]byte, 100000)
	var hash hash.Hash = sha512.New()
	for {
		var numBytesRead int
		numBytesRead, err = file.Read(buf)
		if numBytesRead == 0 { break }
		hash.Write(buf)
		if err != nil { break }
		if numBytesRead < 100000 { break }
	}
	
	var empty = []byte{}
	return hash.Sum(empty), nil
}

/*******************************************************************************
 * Compute a SHA-512 has of the specified string. Salt the hash so that the
 * hash value cannot be forged or identified via a lookup table.
 */
func (authSvc *AuthService) computeHash(s string) hash.Hash {
	
	var hash hash.Hash = sha512.New()
	var bytes []byte = []byte(s)
	hash.Write(authSvc.secretSalt)
	hash.Write(bytes)
	return hash
}

func (authSvc *AuthService) compareHashValues(h1, h2 []byte) bool {
	if len(h1) != len(h2) { return false }
	for i, b := range h1 { if b != h2[i] { return false } }
	return true
}


/***************************** Internal Functions ******************************/


/*******************************************************************************
 * Return true if the party has the right implied by the actionMask, for
 * the specified Resource, based on the ACLEntries that the resource has. Do not
 * attempt to determine if the resource''s owning Resource has applicable ACLEntries.
 * At most one elemente of the actionMask may be true.
 */
func (server *Server) partyHasAccess(party Party, actionMask []bool,
	resource Resource) (bool, error) {
	
	// Discover which field of the action mask is set.
	var action int = -1
	for i, entry := range actionMask {
		if entry {
			if action == -1 { return false, errors.New("More than one field set in action mask") }
			action = i
		}
	}
	if action == -1 { return false, nil }  // no action mask fields were set.
	
	var entries []string = party.getACLEntryIds()
	fmt.Println(fmt.Sprintf("Party " + party.getName() + " (" + party.getId() + // debug
		") has %d ACL entries", len(entries)))  // debug
	for _, entryId := range entries {  // for each of the party's ACL entries...
		
		var entry ACLEntry
		var err error
		entry, err = server.dbClient.getACLEntry(entryId)
		if err != nil { return false, err }
		
		// debug
		var rsc Resource
		rsc, err = server.dbClient.getResource(entry.getResourceId())
		if err != nil { fmt.Println(err.Error()) }
		fmt.Println(fmt.Sprintf("\tentry for resource %s (%s)", rsc.getName(), rsc.getId()))
		// end debug
		
		
		
		if entry.getResourceId() == resource.getId() {  // if the entry references the resource,
			var mask []bool = entry.getPermissionMask()
			if mask[action] { return true, nil }  // party has access to the resource
		}
	}
	return false, nil
}

/*******************************************************************************
 * Returns the "SessionId" cookie value, or "" if there is none.
 * Used by authenticateRequestCookie.
 */
func getSessionIdFromCookie(httpReq *http.Request) string {
	assertThat(httpReq != nil, "In getSessionIdFromCookie, httpReq is nil")
	assertThat(httpReq.Header != nil, "In getSessionIdFromCookie, httpReq.Header is nil")
	
	var cookie *http.Cookie
	var err error
	cookie, err = httpReq.Cookie("SessionId")
	if err != nil {
		fmt.Println("No SessionId cookie found.")
		return ""
	}
	
	var sessionId = cookie.Value
	
	if len(sessionId) == 0 { return "" }
	return sessionId
}

/*******************************************************************************
 * Validate the specified session id. If valid, return a apitypes.SessionToken with
 * the identity of the session owner.
 */
func (authSvc *AuthService) identifySession(sessionId string) *apitypes.SessionToken {
	
	var credentials *apitypes.Credentials = authSvc.Sessions[sessionId]
	
	if credentials == nil {
		fmt.Println("No session found for session id", sessionId)
		return nil
	}
	
	return apitypes.NewSessionToken(sessionId, credentials.UserId)
}

/*******************************************************************************
 * Validate session Id: return true if valid, false otherwise.
 * See also createUniqueSessionId.
 */
func (authSvc *AuthService) validateSessionId(sessionId string) bool {
	
	var parts []string = strings.Split(sessionId, ":")
	if len(parts) != 2 {
		fmt.Println("Ill-formatted sessionId:", sessionId)
		return false
	}
	
	var uniqueNonRandomValue string = parts[0]
	var untrustedHash string = parts[1]
	var empty = []byte{}
	var actualSaltedHashBytes []byte = authSvc.computeHash(uniqueNonRandomValue).Sum(empty)
	
	return untrustedHash == fmt.Sprintf("%x", actualSaltedHashBytes)
}

/*******************************************************************************
 * Return a session id that is guaranteed to be unique, and that is completely
 * opaque and unforgeable. See also validateSessionId.
 */
func (authSvc *AuthService) createUniqueSessionId() string {
	
	var uniqueNonRandomValue string = fmt.Sprintf("%d", time.Now().UnixNano())
	var empty = []byte{}
	var saltedHashBytes []byte =
		authSvc.computeHash(uniqueNonRandomValue).Sum(empty)
	return uniqueNonRandomValue + ":" + fmt.Sprintf("%x", saltedHashBytes)
}
