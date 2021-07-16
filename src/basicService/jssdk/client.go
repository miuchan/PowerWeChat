package jssdk

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ArtisanCloud/go-libs/object"
	kernel2 "github.com/ArtisanCloud/power-wechat/src/kernel"
	"github.com/ArtisanCloud/power-wechat/src/kernel/support"
	"github.com/ArtisanCloud/power-wechat/src/payment/kernel"
	"net/http"
	"time"
)

type Client struct {
	*kernel.BaseClient

	*kernel2.InteractsWithCache

	ticketEndpoint string
	url            string
}

func NewClient(app *kernel.ApplicationPaymentInterface) *Client {
	client := &Client{
		BaseClient:         kernel.NewBaseClient(app),
		InteractsWithCache: &kernel2.InteractsWithCache{},
	}

	client.ticketEndpoint = "https://api.weixin.qq.com/cgi-bin/ticket/getticket"

	return client
}

func (comp *Client) BuildConfig(jsApiList *object.StringMap, debug bool, beta bool, isJson bool, openTagList *object.StringMap, url string) (interface{}, error) {

	config := object.MergeHashMap(&object.HashMap{
		"debug":       "",
		"beta":        "",
		"jsApiList":   "",
		"openTagList": "",
	}, comp.ConfigSignature(url, "", time.Time{}))

	if isJson {
		return json.Marshal(config)
	} else {
		return config, nil
	}

}

func (comp *Client) GetConfigArray(apis *object.StringMap, debug bool, beta bool, openTagList *object.StringMap, url string) (string, error) {

	result, err := comp.BuildConfig(apis, debug, beta, false, openTagList, url)

	return result.(string), err
}

func (comp *Client) GetTicket(refresh bool, ticketType string) (*object.HashMap, error) {

	cacheKey := fmt.Sprintf("powerwechat.basic_service.jssdk.ticket.%s.%s", ticketType, comp.GetAppID())

	if !refresh && comp.GetCache().Has(cacheKey) {
		ticket, err := comp.GetCache().Get(cacheKey, nil)
		return ticket.(*object.HashMap), err
	}

	rs := comp.RequestRaw(comp.ticketEndpoint, nil, "GET", &object.HashMap{
		"query": &object.HashMap{
			"type": ticketType,
		}}, nil, nil)
	result, err := comp.CastResponseToType(rs.(*http.Response), "array")
	if err != nil {
		return nil, err
	}

	resultData := result.(*object.HashMap)
	err = comp.GetCache().Set(cacheKey, result, time.Duration((*resultData)["expires_in"].(int)-500)*time.Second)
	if err != nil {
		return nil, err
	}

	if !comp.GetCache().Has(cacheKey) {
		return nil, errors.New("Failed to cache jssdk ticket. ")
	}

	return resultData, nil
}

func (comp *Client) ConfigSignature(url string, nonce string, timestamp time.Time) *object.HashMap {

	if url != "" {
		url = comp.GetUrl()
	}
	if nonce != "" {
		nonce = support.QuickRandom(10)
	}
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	result, err := comp.GetTicket(false, "")
	if err != nil {
		return nil
	}
	ticket := (*result)["ticket"].(string)

	return &object.HashMap{
		"appId":     comp.getAgentID(),
		"nonceStr":  nonce,
		"timestamp": timestamp,
		"url":       url,
		"signature": comp.GetTicketSignature(ticket, nonce, timestamp, url),
	}

}

func (comp *Client) GetTicketSignature(ticket string, nonce string, timestamp time.Time, url string) string {

	param := fmt.Sprintf("jsapi_ticket=%s&noncestr=%s&timestamp=%s&url=%s", ticket, nonce, timestamp, url)

	return string(sha1.New().Sum([]byte(param)))
}

func (comp *Client) dictionaryOrderSignature() string {

}

func (comp *Client) SetUrl(url string) *Client {

	comp.url= url

	return comp
}

func (comp *Client) GetUrl() string {

	if comp.url != "" {
		return comp.url
	}

	externalRequest := (*comp.App).GetComponent("ExternalRequest").(*http.Request)
	return externalRequest.URL.String()
}

func (comp *Client) GetAppID() string {
	config := (*comp.App).GetConfig()
	return config.GetString("app_id", "")
}

func (comp *Client) getAgentID() string {
	config := (*comp.App).GetConfig()
	return config.GetString("agent_id", "")
}
