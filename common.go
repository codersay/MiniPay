// Copyright 2019 全栈编程@luboke.com  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// MiniPay.

package MiniPay

//封装
import (
	"bytes"
	"crypto/md5"
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"github.com/shopspring/decimal"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"
)

var (
	httpsclient *HTTPSClient
)

//微信小程序支付公共参数
type MiniPayParams struct {
	AppID       string       // appID
	MchID       string       // 商户号
	Key         string       // 密钥
}


// Charge 发起小程序预下单的支付参数，基他参数在发起拼接时临时改，比如minipay.go文件 第57行 payHandle["nonce_str"] = RandomStr()
//https://pay.weixin.qq.com/wiki/doc/api/wxa/wxa_api.php?chapter=9_1
//支付需要的参数，这里是变化的参数，即服务端传递的
type PayArg struct {
	//Golang中，如果指定一个field序列化成JSON的变量名字为-，则序列化的时候自动忽略这个field。
	//而omitempty的作用是当一个field的值是empty的时候，序列化JSON时候忽略这个field（Newtonsoft.Json的类似用法参考这里和例子）。
	//https://ethancai.github.io/2016/06/23/bad-parts-about-json-serialization-in-Golang/
	//使用omitempty熟悉，如果该字段为nil或0值（数字0,字符串"",空数组[]等），则打包的JSON结果不会有这个字段。
	//https://blog.csdn.net/tiaotiaoyly/article/details/38942311
	//type Message struct {
	//	Name string `json:"msg_name"`       // 对应JSON的msg_name
	//	Body string `json:"body,omitempty"` // 如果为空置则忽略字段
	//	Time int64  `json:"-"`              // 直接忽略字段
	//}
	APPID       string       // appID
	TradeNum    string  `json:"tradeNum,omitempty"`
	MoneyFee    float64 `json:"MoneyFee,omitempty"`
	CallbackURL string  `json:"callbackURL,omitempty"`
	Body        string  `json:"body,omitempty"`
	OpenID      string  `json:"openid,omitempty"`
}

// MiniPayCommonResult 基本信息,状态码与状态描述
//统一下单与查询结果通用部分
// https://pay.weixin.qq.com/wiki/doc/api/wxa/wxa_api.php?chapter=9_1
//https://pay.weixin.qq.com/wiki/doc/api/wxa/wxa_api.php?chapter=9_2
//返回结果
type MiniPayCommonResult struct {
	ReturnCode string `xml:"return_code" json:"return_code,omitempty"`
	ReturnMsg  string `xml:"return_msg" json:"return_msg,omitempty"`
}

// MiniPayReturnSuccessData 返回通用数据
//https://pay.weixin.qq.com/wiki/doc/api/wxa/wxa_api.php?chapter=9_1
//统一下单 返回结果 的一部分
// 以下字段在return_code为SUCCESS的时候有返回
//统一下单与查询结果通用部分
type MiniPayReturnSuccessData struct {
	AppID      string `xml:"appid,omitempty" json:"appid,omitempty"`
	MchID      string `xml:"mch_id,omitempty" json:"mch_id,omitempty"`
	//DeviceInfo 统一下单默认就有，查询结果在return_code 、result_code、trade_state都为SUCCESS时有返回，这里统一放在这里
	DeviceInfo string `xml:"device_info,omitempty" json:"device_info,omitempty"`
	NonceStr   string `xml:"nonce_str,omitempty" json:"nonce_str,omitempty"`
	Sign       string `xml:"sign,omitempty" json:"sign,omitempty"`
	ResultCode string `xml:"result_code,omitempty" json:"result_code,omitempty"`
	ErrCode    string `xml:"err_code,omitempty" json:"err_code,omitempty"`
	ErrCodeDes string `xml:"err_code_des,omitempty" json:"err_code_des,omitempty"`
}

// 查询结果或者下单返回公用部分
//https://pay.weixin.qq.com/wiki/doc/api/wxa/wxa_api.php?chapter=9_2
//以下字段在return_code 、result_code、trade_state都为SUCCESS时有返回 ，如trade_state不为 SUCCESS，则只返回out_trade_no（必传）和attach（选传）。
type MiniPayResultData struct {
	OpenID         string `xml:"openid,omitempty" json:"openid,omitempty"`
	IsSubscribe    string `xml:"is_subscribe,omitempty" json:"is_subscribe,omitempty"`
	TradeType      string `xml:"trade_type,omitempty" json:"trade_type,omitempty"`
	TradeState     string `xml:"trade_state,omitempty" json:"trade_state,omitempty"`
	BankType       string `xml:"bank_type,omitempty" json:"bank_type,omitempty"`
	TotalFee       string `xml:"total_fee,omitempty" json:"total_fee,omitempty"`
	SettlementTotalFee  string `xml:"settlement_total_fee,omitempty" json:"settlement_total_fee,omitempty"`
	FeeType        string `xml:"fee_type,omitempty" json:"fee_type,omitempty"`
	CashFee        string `xml:"cash_fee,omitempty" json:"cash_fee,omitempty"`
	CashFeeType    string `xml:"cash_fee_type,omitempty" json:"cash_fee_type,omitempty"`

	/*
	代金券金额	coupon_fee	否	Int	100	“代金券”金额<=订单金额，订单金额-“代金券”金额=现金支付金额，详见支付金额
	代金券使用数量	coupon_count	否	Int	1	代金券使用数量
	代金券类型	coupon_type_$n	否	String	CASH
	CASH--充值代金券
	NO_CASH---非充值优惠券

	开通免充值券功能，并且订单使用了优惠券后有返回（取值：CASH、NO_CASH）。$n为下标,从0开始编号，举例：coupon_type_$0

	代金券ID	coupon_id_$n	否	String(20)	10000 	代金券ID, $n为下标，从0开始编号
	单个代金券支付金额	coupon_fee_$n	否	Int	100	单个代金券支付金额, $n为下标，从0开始编号
	*/


	TransactionID  string `xml:"transaction_id,omitempty" json:"transaction_id,omitempty"`
	OutTradeNO     string `xml:"out_trade_no,omitempty" json:"out_trade_no,omitempty"`
	Attach         string `xml:"attach,omitempty" json:"attach,omitempty"`
	TimeEnd        string `xml:"time_end,omitempty" json:"time_end,omitempty"`
	TradeStateDesc string `xml:"trade_state_desc" json:"trade_state_desc,omitempty"`
}

//支付结果，以下字段在return_code 和result_code都为SUCCESS的时候有返回
type MinipayStateData struct {
	TradeType string `xml:"trade_type" json:"trade_type,omitempty"`
	PrepayID string `xml:"prepay_id" json:"prepay_id,omitempty"`
	CodeURL  string `xml:"code_url" json:"code_url,omitempty"`
}


//异步支付 返回结果
type MiniPayAsyncResult struct {
	//统一下单与查询结果通用部分
	MiniPayCommonResult

	//统一下单与查询结果通用部分
	MiniPayReturnSuccessData

	// 查询结果或者下单返回公用部分
	MiniPayResultData
}

//统一下单请求的响应,即同步的响应
type MiniPaySyncResult struct {
	//统一下单与查询结果通用部分
	MiniPayCommonResult

	//统一下单与查询结果通用部分
	MiniPayReturnSuccessData

	MinipayStateData
}

func init() {
	httpsclient = new(HTTPSClient)
}

// 用户下单支付接口
func Order2Pay(payArg *PayArg) (map[string]interface{}, error) {
	re, err := Minipay().UnifiedPay(payArg)
	return re, err
}

// MiniCallback 微信支付
//response是ResponseWriter接口的实现。
//WriteHeader方法：
//设置响应的状态码，返回错误状态码特别有用。WriteHeader方法在执行完毕之后就不允许对首部进行写入了。
//
//向客户端返回JSON数据：
//首先使用Header方法将内容类型设置成application/json，然后将JSON数据写入ResponseWriter中
//go 处理http response
func MiniPayNotifyCallback(w http.ResponseWriter, body []byte) (*MiniPayAsyncResult, *MiniPayCommonResult, error) {
	var returnCode = "FAIL"
	var returnMsg = ""
	var miniPayCommonResult MiniPayCommonResult
	log.Println(w)
	defer func() {
		//formatStr := `<xml><return_code><![CDATA[%s]]></return_code><return_msg>![CDATA[%s]]</return_msg></xml>`
		//returnBody = fmt.Sprintf(formatStr, returnCode, returnMsg)
		log.Println("log.Println(miniPayCommonResult)---before-----", miniPayCommonResult)
		miniPayCommonResult.ReturnCode = returnCode
		miniPayCommonResult.ReturnMsg = returnMsg
		log.Println("log.Println(miniPayCommonResult)---after-----", miniPayCommonResult)

	}()
	var reXML MiniPayAsyncResult
	log.Println(body)
	log.Println(string(body))
	if string(body) == "" {
		returnCode = "FAIL"
		returnMsg = "Bodyerror"
		return &reXML, &miniPayCommonResult, errors.New(returnCode + ":" + returnMsg)
	}
	err = xml.Unmarshal(body, &reXML)
	if err != nil {
		returnCode = "FAIL"
		returnMsg = "参数错误"
		return &reXML, &miniPayCommonResult, errors.New(returnCode + ":" + returnMsg)
	}

	if reXML.ReturnCode != "SUCCESS" {
		returnCode = "FAIL"
		return &reXML, &miniPayCommonResult, errors.New(reXML.ReturnCode)
	}
	m := XmlToMap(body)

	var signData []string
	for k, v := range m {
		if k == "sign" {
			continue
		}
		signData = append(signData, fmt.Sprintf("%v=%v", k, v))
	}

	log.Println(signData)

	log.Println("minipay()----", &Minipay().Key)
	key := Minipay().Key
	log.Println("key------", key)
	mySign, err := MinipaySign(key, m)
	if err != nil {
		return &reXML, &miniPayCommonResult, err
	}

	if mySign != m["sign"] {
		panic(errors.New("签名交易错误"))
	}

	returnCode = "SUCCESS"
	returnMsg = "SUCCESS"
	return &reXML, &miniPayCommonResult, nil
}

func MinipaySign(key string, m map[string]interface{}) (string, error) {
	var signData []string
	for k, v := range m {

		//签名之前的拼接数据，需要过滤掉sign和key关键字
		if v != "" && k != "sign" && k != "key" {
			signData = append(signData, fmt.Sprintf("%s=%s", k, v))
		}
	}

	//按ascii字母排序
	sort.Strings(signData)

	//将字符串数组按照&拼接起来
	signStr := strings.Join(signData, "&")

	//最后拼接上key，得到签名之前的字符串
	signStr = signStr + "&key=" + key

	log.Println("签名之前的字符串------------", signStr)

	md5Handle := md5.New()
	_, err := md5Handle.Write([]byte(signStr))
	if err != nil {
		return "", errors.New("MinipaySign md5.Write: " + err.Error())
	}
	signByte := md5Handle.Sum(nil)
	if err != nil {
		return "", errors.New("MinipaySign md5.Sum: " + err.Error())
	}

	tosign := strings.ToUpper(fmt.Sprintf("%x", signByte))
	log.Println("签名的结果为-------", tosign)
	return tosign, nil
}

//对微信下订单或者查订单
func PostMiniPay(url string, data map[string]interface{}) (MiniPaySyncResult, error) {
	var xmlRe MiniPaySyncResult
	buf := bytes.NewBufferString("")

	for k, v := range data {
		buf.WriteString(fmt.Sprintf("<%s><![CDATA[%s]]></%s>", k, v, k))
	}
	xmlStr := fmt.Sprintf("<xml>%s</xml>", buf.String())

	log.Println("发起预下单的xml数据：----------", xmlStr)
	log.Println("通过Http客户端发起请求------", httpsclient)
	log.Println("通过Http客户端发起请求，请求的Url------", url)
	log.Println("通过Http客户端发起请求，请求的数据------", xmlStr)
	re, err := httpsclient.PostData(url, "text/xml:charset=UTF-8", xmlStr)
	log.Println("通过Http客户端发起请求，回返的数据是字节，经过string之后的数据------", string(re))
	if err != nil {
		return xmlRe, errors.New("通过Http客户端发起请求出错，错误---------- " + err.Error())
	}

	if err = xml.Unmarshal(re, &xmlRe); err != nil {
		return xmlRe, errors.New("通过Http客户端发起请求，回返的数据是字节，经过string之后的数据，经过xml Unmarshal出错---------- " + err.Error())
	}

	if xmlRe.ReturnCode != "SUCCESS" {
		// 通信失败
		return xmlRe, errors.New("通过Http客户端发起请求，回返的数据是字节，经过string之后的数据 ReturnMsg出错 ----------- " + xmlRe.ReturnMsg)
	}

	if xmlRe.ResultCode != "SUCCESS" {
		// 业务结果失败
		return xmlRe, errors.New("通过Http客户端发起请求，回返的数据是字节，经过string之后的数据 ErrCodeDes 出错 ----------- " + xmlRe.ErrCodeDes)
	}
	return xmlRe, nil
}

// 微信金额浮点转字符串
func Float2String(moneyFee float64) string {
	aDecimal := decimal.NewFromFloat(moneyFee)
	bDecimal := decimal.NewFromFloat(100)
	return aDecimal.Mul(bDecimal).Truncate(0).String()
}

//RandomStr 获取一个随机字符串
func RandomString() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func XmlToMap(xmlData []byte) map[string]interface{} {
	decoder := xml.NewDecoder(bytes.NewReader(xmlData))
	m := make(map[string]interface{})
	var token xml.Token
	var err error
	var k string
	for token, err = decoder.Token(); err == nil; token, err = decoder.Token() {
		if v, ok := token.(xml.StartElement); ok {
			k = v.Name.Local
			continue
		}
		if v, ok := token.(xml.CharData); ok {
			data := string(v.Copy())
			if strings.TrimSpace(data) == "" {
				continue
			}
			m[k] = data
		}
	}

	if err != nil && err != io.EOF {
		panic(err)
	}
	return m
}

// LocalIP 获取机器的IP
func LocalIP() string {
	info, _ := net.InterfaceAddrs()
	for _, addr := range info {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		if !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
			return ipNet.IP.String()
		}
	}
	return ""
}

