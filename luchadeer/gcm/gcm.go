package gcm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type GCMMessage struct {
	Data            map[string]interface{} `json:"data"`
	RegistrationIds []string               `json:"registration_ids"`
}

type GCMResult struct {
	MessageId      string `json:"message_id"`
	RegistrationId string `json:"registration_id"`
	Error          string `json:"error"`
}

type GCMResponse struct {
	MulticastId  int         `json:"multicast_id"`
	Success      int         `json:"success"`
	Failure      int         `json:"failure"`
	CanonicalIds int         `json:"canonical_ids"`
	Results      []GCMResult `json:"results"`
}

type GCM struct {
	apiKey string
	client *http.Client
}

var GCMSendURL = "https://android.googleapis.com/gcm/send"
var MethodPost = "POST"

func NewGCM(apiKey string, client *http.Client) *GCM {
	if client == nil {
		client = http.DefaultClient
	}

	return &GCM{
		apiKey: apiKey,
		client: client,
	}
}

func (gcm *GCM) Send(data map[string]interface{}, registrationIds []string) (*GCMResponse, error) {
	marshalled, err := json.Marshal(&GCMMessage{
		Data:            data,
		RegistrationIds: registrationIds,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(MethodPost, GCMSendURL, bytes.NewBuffer(marshalled))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", "key="+gcm.apiKey)
	req.Header.Add("Content-Type", "application/json")

	resp, err := gcm.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Non OK return status: %v", resp.StatusCode)
	}

	var gcmResponse GCMResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&gcmResponse); err != nil {
		return nil, err
	}

	return &gcmResponse, nil
}
