// Copyright 2019 全栈编程@luboke.com  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// MiniPay.

package MiniPay

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/astaxie/beego/logs"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

//http/https客户端及方法

//http客户端
type HTTPClient struct {
	http.Client
}

//HTTPS客户端
type HTTPSClient struct {
	http.Client
}

var (
	HTTPC  *HTTPClient
	HTTPSC *HTTPSClient
)

func init() {
	HTTPC = &HTTPClient{}
	//HTTPSC = NewHTTPSClient([]byte{}, []byte{})
}

// NewHTTPSClient 获取默认https客户端
func NewHTTPSClient(certPEMBlock, keyPEMBlock []byte) *HTTPSClient {
	config := new(tls.Config)
	if len(certPEMBlock) != 0 && len(keyPEMBlock) != 0 {
		cert, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)
		if err != nil {
			panic("load x509 cert error:" + err.Error())
			return nil
		}
		config = &tls.Config{
			Certificates: []tls.Certificate{
				cert,
			},
		}
	} else {
		config = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	tr := &http.Transport{
		TLSClientConfig: config,
	}
	client := http.Client{
		Transport: tr,
		Timeout:   15 * time.Second,
	}
	return &HTTPSClient{
		Client: client,
	}
}

// https 提交post数据
func (httpsclient *HTTPSClient) PostData(url string, contentType string, data string) ([]byte, error) {
	resp, err := httpsclient.Post(url, contentType, strings.NewReader(data))
	logs.Debug("data-------", data)
	logs.Debug("PostData响应的结果-------", resp)
	logs.Debug("PostData响应的错误-------", err)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

// https get数据
func (c *HTTPSClient) GetData(url string) ([]byte, error) {
	resp, err := c.Get(url)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode != 200 {
		return []byte{}, errors.New("http.stateCode != 200 : " + fmt.Sprintf("%+v", resp))
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

// http post数据
func (c *HTTPClient) PostData(url, format string, data string) ([]byte, error) {
	resp, err := c.Post(url, format, strings.NewReader(data))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}
