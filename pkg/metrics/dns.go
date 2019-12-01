package metrics

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"
)

type GravityStatus struct {
	FileExist  bool        `json:"file_exists"`
	AbsoluteTS uint64      `json:"absolute"`
	RelativeTS interface{} `json:"relative"`
}

type PiHoleStatus struct {
	BlockedDomains      uint64        `json:"domains_being_blocked"`
	TotalQueriesToday   uint64        `json:"dns_queries_today"`
	BlockedQueriesToday uint64        `json:"ads_blocked_today"`
	BlockedPcntToday    float64       `json:"ads_percentage_today"`
	UniqueDomains       uint64        `json:"unique_domains"`
	ForwardedQueries    uint64        `json:"queries_forwarded"`
	CachedQueries       uint64        `json:"queries_cached"`
	ClientsEverSeen     uint64        `json:"clients_ever_seen"`
	UniqueClients       uint64        `json:"clients_unique"`
	TotalQueries        uint64        `json:"dns_queries_all_types"`
	NODATAReplies       uint64        `json:"reply_NODATA"`
	NXDOMAINReplies     uint64        `json:"reply_NXDOMAIN"`
	CNAMEReplies        uint64        `json:"reply_CNAME"`
	IPReplies           uint64        `json:"reply_IP"`
	PrivacyLevel        uint64        `json:"privacy_level"`
	Status              string        `json:"status"`
	GravityStatus       GravityStatus `json:"gravity_last_updated"`
}

func (e *PiHoleStatus) UpdateDNS(target string) error {
	var httpClient = &http.Client{
		Timeout: time.Second * 5,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 5 * time.Second,
		},
	}
	req, err := http.NewRequest("GET", "http://"+target+"/admin/api.php?summaryRaw", nil)
	if err != nil {
		return err
	}
	response, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	json.Unmarshal([]byte(body), &e.status)
	return nil
}
