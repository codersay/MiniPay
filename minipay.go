// Copyright 2019 全栈编程@luboke.com  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// MiniPay.

package MiniPay

import (
	"errors"
	"fmt"
	//"strconv"
	"time"
	"log"
)

//定义变量
var (
	miniPayParams *MiniPayParams
	sign          string
	err           error
	payHandle     map[string]interface{}
	appId         string
	signType      string
	tradeType     string
	xmlRe         MiniPaySyncResult //支付同步结果
)

//初始化参数
func init() {
	payHandle = make(map[string]interface{})
	signType = "MD5"
	tradeType = "JSAPI"
}

//初始化支付公用参数
func InitMiniPay(pay *MiniPayParams){
	miniPayParams = pay
}

//返回支付公用参数
func Minipay() *MiniPayParams {
	return miniPayParams
}

// Pay 支付
func (this *MiniPayParams) UnifiedPay(payArg *PayArg) (map[string]interface{}, error) {
	appId = this.AppID
	if payArg.APPID != "" {
		appId = payArg.APPID
	}
	payHandle = make(map[string]interface{})
	payHandle["appid"] = appId
	payHandle["mch_id"] = this.MchID
	payHandle["nonce_str"] = RandomString()
	payHandle["body"] = payArg.Body
	payHandle["out_trade_no"] = payArg.TradeNum
	payHandle["total_fee"] = fmt.Sprintf("%f",payArg.MoneyFee)

	//payHandle["total_fee"] = strconv.FormatFloat(payArg.MoneyFee,'E',-1,32)
 	//payHandle["spbill_create_ip"] = common.LocalIP()
	payHandle["spbill_create_ip"] = "49.234.14.102"
	payHandle["notify_url"] = payArg.CallbackURL
	payHandle["trade_type"] = tradeType
	payHandle["openid"] = payArg.OpenID
	payHandle["sign_type"] = signType
	log.Println("签名之前的数据：", payHandle)
	if sign, err = MinipaySign(this.Key, payHandle); err != nil {
		log.Println("签名失败：", err.Error())
		return payHandle, err
	}
	payHandle["sign"] = sign

	log.Println("发起支付的参数：------", payHandle)

	// 预下单
	if xmlRe, err = PostMiniPay("https://api.mch.weixin.qq.com/pay/unifiedorder", payHandle); err != nil {
		log.Println("预下单失败：", err.Error())
		return payHandle, err
	}

	//再次使用，请重新 make,避免数据污染，因为有之前的数据
	payHandle = make(map[string]interface{})
	payHandle["appId"] = appId
	payHandle["timeStamp"] = fmt.Sprintf("%d", time.Now().Unix())
	payHandle["nonceStr"] = RandomString()
	payHandle["package"] = fmt.Sprintf("prepay_id=%s", xmlRe.PrepayID)
	payHandle["signType"] = "MD5"
	if sign, err = MinipaySign(this.Key, payHandle); err != nil {
		return payHandle, errors.New("MiniWeb: " + err.Error())
	}

	payHandle["paySign"] = sign
	return payHandle, nil
}
