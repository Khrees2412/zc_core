package realtime

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/mitchellh/mapstructure"

	"zuri.chat/zccore/auth"
	"zuri.chat/zccore/utils"
)

var (
	validate = validator.New()
)

type CentrifugoConnectResult struct {
	User string `json:"user" bson:"user"`
}

type CentrifugoConnectResponse struct {
	Result CentrifugoConnectResult `json:"result" bson:"result"`
}

type CentrifugoRefreshResponse struct {
	Result CentrifugoRefreshResult `json:"result" bson:"result"`
}
type CentrifugoRefreshResult struct {
	ExpireAt string `json:"expire_at" bson:"expire_at"`
}

type CentrifugoConnectRequest struct {
	Client    string `json:"client" bson:"client"`
	Transport string `json:"transport" bson:"transport"`
	Protocol  string `json:"protocol" bson:"protocol"`
	Encoding  string `json:"encoding" bson:"encoding"`
}

func Auth(w http.ResponseWriter, r *http.Request) {

	// 1. Decode the request from centrifugo
	var creq CentrifugoConnectRequest
	err := json.NewDecoder(r.Body).Decode(&creq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Println(r.Header.Get("Authorization"))
	// 2. Authenticate user
	headerToken := ExtractHeaderToken(r)
	fmt.Println(creq, headerToken)

	if headerToken == "" {
		CentrifugoNotAuthenticatedResponse(w)
	}

	// 3. Generate a response object. In final version you have to
	// check that this person is authenticated
	// u, _ := uuid.NewV4()
	userID, err := CentifugoConnectAuth(r)
	if err != nil {
		CentrifugoNotAuthenticatedResponse(w)
	}

	data := CentrifugoConnectResponse{}
	// data.Result.User = u.String()
	data.Result.User = userID

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)

}

func Refresh(w http.ResponseWriter, r *http.Request) {

	data := CentrifugoRefreshResponse{}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(data)

}

func Test(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./realtime/test_rtc.html")
}

func PublishEvent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var event utils.Event
	err := utils.ParseJsonFromRequest(r, &event)
	if err != nil {
		utils.GetError(err, http.StatusUnprocessableEntity, w)
		return
	}
	if err = validate.Struct(event); err != nil {
		utils.GetError(err, http.StatusBadRequest, w)
		return
	}
	res := utils.Emitter(event)
	utils.GetSuccess("publish event status", res, w)

}

// Creates a 'not authenticated' response for given user connection request
func CentrifugoNotAuthenticatedResponse(w http.ResponseWriter) {
	data := CentrifugoConnectResponse{}
	data.Result.User = ""
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(data)
}

// CentifugoConnectAuth returns the user ID of an authenticated user from the
// bearer token in the original connect request or returns an error if the user
// is not found
func CentifugoConnectAuth(r *http.Request) (userID string, err error) {
	// 1. Validate the token
	configuration := utils.NewConfigurations()
	signingKey := configuration.HmacSampleSecret
	status, err, sessionData := auth.GetSessionDataFromToken(r, []byte(signingKey))

	// 2. Check for a user record that's assigned this token
	if status {
		userID, err = UserIDFromSession(sessionData, *configuration)
		if err != nil {
			return "", nil
		}
		// 3. Return user ID and nil error if user is found
		return userID, nil
	}
	return "", err
}

// Extract the token from the request header
func ExtractHeaderToken(r *http.Request) string {
	headerToken := r.Header.Get("Authorization")
	return headerToken
}

// Extract user id from token
func UserIDFromSession(sessionData auth.ResToken, conf utils.Configurations) (string, error) {
	var data map[string]interface{}
	mapstructure.Decode(sessionData, data)
	session, err := utils.GetMongoDbDoc(conf.SessionDbCollection, data)
	if err != nil {
		return "", err
	}
	return session["user_id"].(string), nil
}
