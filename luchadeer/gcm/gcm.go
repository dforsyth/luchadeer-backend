/*
 * Copyright (c) 2014, David Forsythe
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 *
 *  Redistributions of source code must retain the above copyright notice, this
 *   list of conditions and the following disclaimer.
 *
 *  Redistributions in binary form must reproduce the above copyright notice,
 *   this list of conditions and the following disclaimer in the documentation
 *   and/or other materials provided with the distribution.
 *
 *  Neither the name of Luchadeer nor the names of its
 *   contributors may be used to endorse or promote products derived from
 *   this software without specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
 * AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 * DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
 * FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
 * DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
 * SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
 * CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
 * OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
 * OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

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
