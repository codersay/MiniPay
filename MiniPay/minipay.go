package MiniPay

import (
	"errors"
	"fmt"
	"github.com/astaxie/beego/logs"
	//"strconv"
	"time"
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
	xmlRe         MiniPaySyncResult
)

//初始化参数
func init() {
	payHandle = make(map[string]interface{})
	signType = "MD5"
	tradeType = "JSAPI"
}

//初始化小程序参数
func InitMinipay(pay *MiniPayParams) {
	miniPayParams = pay
}

func Minipay() *MiniPayParams {
	return miniPayParams
}

// Pay 支付
func (this *MiniPayParams) UnifiedPay(charge *PayArg) (map[string]interface{}, error) {
	appId = this.AppID
	if charge.APPID != "" {
		appId = charge.APPID
	}
	payHandle = make(map[string]interface{})
	payHandle["appid"] = appId
	payHandle["mch_id"] = this.MchID
	payHandle["nonce_str"] = RandomString()
	//payHandle["body"] = PreDealData(charge.Body, 32)
	payHandle["body"] = (charge.Body)
	payHandle["out_trade_no"] = charge.TradeNum
	payHandle["total_fee"] = Float2String(charge.MoneyFee)

	//payHandle["total_fee"] = strconv.FormatFloat(charge.MoneyFee,'E',-1,32)
 	//payHandle["spbill_create_ip"] = common.LocalIP()
	payHandle["spbill_create_ip"] = "49.234.14.102"
	payHandle["notify_url"] = charge.CallbackURL
	payHandle["trade_type"] = tradeType
	payHandle["openid"] = charge.OpenID
	payHandle["sign_type"] = signType
	logs.Debug("签名之前的数据：", payHandle)
	if sign, err = MinipaySign(this.Key, payHandle); err != nil {
		logs.Debug("签名失败：", err.Error())
		return payHandle, err
	}
	payHandle["sign"] = sign

	logs.Debug("发起支付的参数：------", payHandle)

	// 预下单
	if xmlRe, err = PostMiniPay("https://api.mch.weixin.qq.com/pay/unifiedorder", payHandle); err != nil {
		logs.Debug("预下单失败：", err.Error())
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