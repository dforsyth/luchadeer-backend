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

package config

import (
	"time"
)

const ContentProviderHost = "www.giantbomb.com"
const ContentProviderApiPath = "/api"

// api keys

// cloud messaging api key. leave blank to not send push notifications
const GCMApiKey = ""

// giant bomb api keys

// The pull api key. use a subscriber key here to provide push notifications for subsriber content.
const PullApiKey = ""

// Api key used to proxy users to the content provider. Use a subscriber key here only if you want
// to proxy subscriber content... which you probably dont.
const ProxyApiKey = ""

// number of videos we check with each pull
const VideoPullSize = 1

// minimum client version before forcing an update (major, minor, bugfix)
var MinVersion = []int{0, 0, 0}

// redirect from /
const ClientDownloadURL = ""

var ValidVideoCategories = map[int]string{
	2:  "Reviews",
	3:  "Quick Looks",
	4:  "TANG",
	5:  "Endurance Run",
	6:  "Events",
	7:  "Trailers",
	8:  "Features",
	10: "Subscriber",
	11: "Extra Life",
	12: "Encyclopedia Bombastica",
	13: "Unfinished",
}

// enable
const ProxyRequests = true
const SearchProxyEnabled = true

const DefaultCacheTTL = time.Hour * 24

const ListRequestCacheTTL = time.Hour

const GameDetailCacheTTL = time.Hour * 24
const VideoDetailCacheTTL = time.Hour * 24 * 7

var BadRequestCacheTTL = time.Hour
