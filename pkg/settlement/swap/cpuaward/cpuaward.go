// Copyright 2020 The Infinity Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cpuaward

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	//"math"
	"math/big"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"

	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	externalip "github.com/glendc/go-external-ip"
	"github.com/klauspost/cpuid"
	"github.com/yanhuangpai/voyager/pkg/settlement/swap/erc20"
	"github.com/yanhuangpai/voyager/pkg/settlement/swap/transaction"
)

type Service interface { // Service is the main interface for interacting with the nodes chequebook.
	GetIfi()
}

type service struct {
	lock               sync.Mutex
	transactionService transaction.Service

	ownerAddress common.Address

	erc20Service erc20.Service

	initNum *big.Int
}

// NewCPUAward New creates a new chequebook service for the provided chequebook contract.
func NewCPUAward(transactionService transaction.Service, ownerAddress common.Address) (Service, error) {

	return &service{
		transactionService: transactionService,
		ownerAddress:       ownerAddress,
		initNum:            big.NewInt(0),
	}, nil
}

// Compute returns the score of current device's CPU
func (s *service) Compute() {
	ticker := time.NewTicker(time.Second * 60)
	go func() {
		for range ticker.C {
			//tip1 := fmt.Sprintf("compute cpu award according to the following cpu information:%x", s.ownerAddress)
			// println(tip1)
			score, _, _, _ := CPUScore()
			//tip2 := fmt.Sprintf("The score of CPU is: %x", score)
			// println(tip2)
			url1 := fmt.Sprintf("http://112.35.192.13:8081/irc20/send_ifi?address=0x%x&amount=%x", s.ownerAddress, score)
			req, err := http.NewRequest("GET", url1, nil)
			if err != nil {
				continue
			}
			res, _ := http.DefaultClient.Do(req)
			body, _ := ioutil.ReadAll(res.Body)
			fmt.Println(string(body))
		}
	}()
}

func (s *service) GetIfi() {

	totalAward := 0                    //每天给挖矿者总的激励数
	flag := false                      //true:当天发送量已达到最大值;false:当天发送量未达到最大值
	min := 0.05                        //最小激励，单位ether
	max := 0.5                         //最大激励，单位ether
	ratio := 1.00                      //用于控制客户自家机器激励多15%
	ratio1 := 0.6                      //用于控制挖矿者激励为总激励的60%
	decimals := 1000000000000000000.00 //1 ether

	idCode := getIdCode()
	if idCode != "" { //有idCode，为客户自家机器
		ratio = 1.15
	}

	min1 := (int)(min * ratio * decimals) //扩大最小值范围
	max1 := (int)(max * ratio * decimals) //扩大最大值范围
	hasSendTimes := 0                     //发送次数，取值区间【1,2,3,...47,0】， 1表示当天第一次发送，0表示当天最后一次发送

	score1 := 0 //当天应发给挖矿者的激励

	ticker := time.NewTicker(time.Second * 60 * 30)
	go func() {
		for range ticker.C {
			//tip1 := fmt.Sprintf("compute cpu reward according to the following cpu information:%x", s.ownerAddress)
			//println(tip1)
			score, cpuName, physicsScore, _ := CPUScore()
			//tip2 := fmt.Sprintf("The score of CPU is: %x", score)
			// println(tip2)
			consensus := externalip.DefaultConsensus(nil, nil)
			// Get your IP,
			// which is never <nil> when err is <nil>.
			ip, err := consensus.ExternalIP()
			if err != nil {
				fmt.Println(ip.String()) // print IPv4/IPv6 in string format
				continue
			}
			// println("before rand score",score)

			rand.Seed(time.Now().UnixNano())                   //随机数种子
			rand1 := (float64(rand.Intn(21)) + 90.00) / 100.00 //生成随机数
			score = int(float64(score) * rand1)                //随机上下浮动10%
			//	println("rand ",rand1);
			//	println("after rand score",score);

			score = int(float64(score) * ratio)   //有idCode增加15%
			score1 = int(float64(score) * ratio1) //挖矿者激励为总激励的60%

			tm := time.Now() //获取当前时间

			if tm.Hour() == 0 && tm.Minute() <= 30 { //第二天第一次发送 ，初始化
				flag = false
				hasSendTimes = 0
				totalAward = 0.00
			}

			hasSendTimes += 1
			hasSendTimes = hasSendTimes % 48

			if flag { //当天已发满
				continue
			}

			if totalAward+score1 > max1 {
				score1 = max1 - totalAward //根据最大发送值当天调整发送量
			}

			if tm.Hour() == 23 && tm.Minute() >= 30 && hasSendTimes == 0 { //根据最小发送值调整当天发送量
				if totalAward+score1 < min1 {
					score1 = min1 - totalAward
				}
			}

			hasSendTimes++

			url1 := "http://112.35.192.13:8081/irc20/get_ifi" //web.ifichain.com:8080
			song := make(map[string]interface{})
			song["owner_address"] = s.ownerAddress
			song["cpu_score"] = score
			song["local_ip"] = ip.String()
			song["cpu_name"] = cpuName
			song["physicsScore"] = physicsScore
			song["idCode"] = idCode
			song["apiKey"] = "e1628fd41c0a0bf3fe673ac5a52de0370b32bdc484d19f15feb012c748ed459c"
			song["logicalScore"]=float64(cpuid.CPU.LogicalCores)*float64(cpuid.CPU.Hz)/1000000000.00
			bytesData, err := json.Marshal(song)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			reader := bytes.NewReader(bytesData)
			req, err := http.NewRequest("POST", url1, reader)
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			req.Header.Set("Content-Type", "application/json;charset=UTF-8")
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			body, _ := ioutil.ReadAll(res.Body)

			resJson := string(body)
			if resJson == "" {
				//fmt.Printf("\nfailed to send SLK to %s in CPU award,the server do not response\n", s.ownerAddress)
				log(fmt.Sprintf("failed to send SLK to %x in CPU award,the server do not response", s.ownerAddress), 0, 0)
				continue
			}
			resp := responseType{}
			errJson := json.Unmarshal([]byte(resJson), &resp)
			if errJson != nil {
				//	fmt.Printf("\nfailed to send SLK to %s in CPU award\n%s\n",s.ownerAddress, resJson)
				log(fmt.Sprintf("failed to send SLK to %x in CPU award,because %s", s.ownerAddress, resJson), 2, 0)
				continue
			}
			if resp.ResCode == 200 {
				//	fmt.Printf("\nsend %.4f SLK to %x in CPU award successfully,the transactionHash is %s\n", resp.Amount, s.ownerAddress, resp.TransactionHash)
				log(fmt.Sprintf("send %.4f SLK to %x in CPU award successfully,the transactionHash is %s", resp.Amount, s.ownerAddress, resp.TransactionHash), 0, 0)
			} else {
				//	fmt.Printf("\nfailed to send SLK to %s in CPU award,because %s\n", s.ownerAddress,resp.ErrorMsg)
				log(fmt.Sprintf("failed to send SLK to %x in CPU award,because %s", s.ownerAddress, resp.ErrorMsg), 2, 0)
				continue
			}

			totalAward += score1
			if totalAward >= max1 { //如果当天发送量已达到最大发送值
				flag = true //当天不再发送
			}
			//	fmt.Println(string(body))

		}
	}()
	addr := fmt.Sprintf("0x%x", s.ownerAddress)
	ifiAward(addr)
}

/*func (s *service) GetIfi() {

	    totalAward:=0
         flag:=false

         min:=0.05
         max:=0.5
         ratio:=1.00
         ratio1:=0.6
         decimals:=1000000000000000000.00

         idCode:=getIdCode()
         if idCode!="" {
         	ratio=1.15
         }

         min1:=(int)(min*ratio*decimals)
         max1:=(int)(max*ratio*decimals)
         hasSendTimes:=0

         score1:=0



	ticker := time.NewTicker(time.Second * 60 )
	go func() {
		for _ = range ticker.C {





			  println("\ntime ",hasSendTimes+1,"\n")



			    hasSendTimes+=1
			    hasSendTimes=hasSendTimes%48



			//tip1 := fmt.Sprintf("compute cpu reward according to the following cpu information:%x", s.ownerAddress)
			//println(tip1)
			score, cpuName, _ := CPUScore()
			println("before rand score",score)

			rand.Seed(time.Now().UnixNano())
			rand1 := (float64(rand.Intn(21))+90.00)/100.00  //随机上下浮动10%
			score=int(float64(score)*rand1)
			println("rand ",rand1)
			println("after rand score",score)


            score=int(float64(score)*ratio)   //有IDcode增加15%
            println("idCode score",score)
			//tip2 := fmt.Sprintf("The score of CPU is: %x", score)
			// println(tip2)
			consensus := externalip.DefaultConsensus(nil, nil)
			// Get your IP,
			// which is never <nil> when err is <nil>.
			ip, err := consensus.ExternalIP()
			if err != nil {
				fmt.Println(ip.String()) // print IPv4/IPv6 in string format
				continue
			}


              score1=int(float64(score)*ratio1)



               if hasSendTimes==1 {
               	flag=false
               	totalAward=0
               }



                   if flag {

                   	println("\n has send enough \n")
                   	 continue
                   }

                 if totalAward+score1>max1 {
                    score1= max1-totalAward
                 }

                 if hasSendTimes==0 {
                    if totalAward+score1<min1 {
                     score1=min1-totalAward
                    }
               }



			url := "http://112.35.192.13:8081/irc20/get_ifi"   //web.ifichain.com:8080
			song := make(map[string]interface{})
			song["owner_address"] = s.ownerAddress
			song["cpu_score"] = int((float64(score1)/ratio1))
			println("\n start to send CPUAWARD,the score: ",score,"\n")
			song["local_ip"] = ip.String()
			song["cpu_name"] = cpuName
			song["status"] = 1
			bytesData, err := json.Marshal(song)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			reader := bytes.NewReader(bytesData)
			req, err := http.NewRequest("POST", url, reader)
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			req.Header.Set("Content-Type", "application/json;charset=UTF-8")
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
			body, _ := ioutil.ReadAll(res.Body)

			fmt.Println(string(body))

			 totalAward+=score1

			// println("\n the totalAward is",Round(float64(totalAward)/1000000000000000000.00,6),"\n")
			// println("\n the max1 is",Round(float64(max1)/1000000000000000000.00,6),"\n")

            fmt.Printf("\n the SLK is %.8f \n\n",float64(score1)/1000000000000000000.00)
			fmt.Printf("\n the totalAward is %.8f \n\n",float64(totalAward)/1000000000000000000.00)
			fmt.Printf("\n the min is %.8f \n\n",float64(min1)/1000000000000000000.00)
			fmt.Printf("\n the max is %.8f \n\n",float64(max1)/1000000000000000000.00)
            if totalAward>=max1 {
            	flag=true
            }
		}
	}()
//	addr := fmt.Sprintf("0x%x", s.ownerAddress)
	//ifi_award(addr)
}*/

// CPUScore returns the score of current device's CPU
func CPUScore() (score int, cpuName string, physicsScore int, err error) {
	// Print basic CPU information:
	/*fmt.Println("Name:", cpuid.CPU.BrandName)
	fmt.Println("PhysicalCores:", cpuid.CPU.PhysicalCores)
	fmt.Println("ThreadsPerCore:", cpuid.CPU.ThreadsPerCore)
	fmt.Println("LogicalCores:", cpuid.CPU.LogicalCores)
	fmt.Println("Family", cpuid.CPU.Family, "Model:", cpuid.CPU.Model)
	fmt.Println("Features:", cpuid.CPU.Features)
	fmt.Println("Cacheline bytes:", cpuid.CPU.CacheLine)
	fmt.Println("L1 Data Cache:", cpuid.CPU.Cache.L1D, "bytes")
	fmt.Println("L1 Instruction Cache:", cpuid.CPU.Cache.L1D, "bytes")
	fmt.Println("L2 Cache:", cpuid.CPU.Cache.L2, "bytes")
	fmt.Println("L3 Cache:", cpuid.CPU.Cache.L3, "bytes")

	//Test if we have a specific feature:
	if cpuid.CPU.SSE() {
		fmt.Println("We have Streaming SIMD Extensions")
	}*/

	score = (3 + cpuid.CPU.PhysicalCores + cpuid.CPU.LogicalCores) * cpuid.CPU.ThreadsPerCore * (cpuid.CPU.CacheLine*100000 + cpuid.CPU.Cache.L1D*100 + cpuid.CPU.Cache.L2*10 + cpuid.CPU.Cache.L3) //* 10000*10000
	score1 := float64(score) / 319109109.00 * (0.20 * 1000000000000000000 / 48.00) / 0.6
	// println("metaengine the score:",score)
	// println("metaengine the score adjusted :",score1)
	log(fmt.Sprintf("metaengine the score:%d", score), 0, 0)
	log(fmt.Sprintf("metaengine the score adjusted :%.4f", score1), 0, 0)
	return int(score1), cpuid.CPU.BrandName, score, nil
}

func GetOutboundIP() net.IP { // Get preferred outbound ip of this machine
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		fmt.Println(err.Error())
	}
	err1 := conn.Close()
	println("close get ip request error,%s", err1.Error())
	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

type responseType struct {
	ResCode         int     `json:"resCode"`
	ErrorMsg        string  `json:"errorMsg"`
	TransactionHash string  `json:"transactionHash"`
	Amount          float64 `json:"amount"`
}

//satellites
type satellites struct {
	StorageDaily []struct {
		AtRestTotal   float64   `json:"atRestTotal"`
		IntervalStart time.Time `json:"intervalStart"`
	} `json:"storageDaily"`
	BandwidthDaily []struct {
		Egress struct {
			Repair int `json:"repair"`
			Audit  int `json:"audit"`
			Usage  int `json:"usage"`
		} `json:"egress"`
		Ingress struct {
			Repair int `json:"repair"`
			Usage  int `json:"usage"`
		} `json:"ingress"`
		Delete        int       `json:"delete"`
		IntervalStart time.Time `json:"intervalStart"`
	} `json:"bandwidthDaily"`
	StorageSummary   float64   `json:"storageSummary"`
	BandwidthSummary int       `json:"bandwidthSummary"`
	EgressSummary    int       `json:"egressSummary"`
	IngressSummary   int       `json:"ingressSummary"`
	EarliestJoinedAt time.Time `json:"earliestJoinedAt"`
	Audits           []struct {
		AuditScore      int    `json:"auditScore"`
		SuspensionScore int    `json:"suspensionScore"`
		OnlineScore     int    `json:"onlineScore"`
		SatelliteName   string `json:"satelliteName"`
	} `json:"audits"`
}

type sno struct {
	NodeID         string      `json:"nodeID"`
	Wallet         string      `json:"wallet"`
	WalletFeatures interface{} `json:"walletFeatures"`
	Satellites     []struct {
		Id                 string      `json:"id"`
		Url                string      `json:"url"`
		Disqualified       interface{} `json:"disqualified"`
		Suspended          interface{} `json:"suspended"`
		CurrentStorageUsed int         `json:"currentStorageUsed"`
	} `json:"satellites"`
	DiskSpace struct {
		Used      int   `json:"used"`
		Available int64 `json:"available"`
		Trash     int   `json:"trash"`
		Overused  int   `json:"overused"`
	} `json:"diskSpace"`
	Bandwidth struct {
		Used      int `json:"used"`
		Available int `json:"available"`
	} `json:"bandwidth"`
	LastPinged     time.Time `json:"lastPinged"`
	Version        string    `json:"version"`
	AllowedVersion string    `json:"allowedVersion"`
	UpToDate       bool      `json:"upToDate"`
	StartedAt      time.Time `json:"startedAt"`
}

type addrType struct {
	Address    string `json:"address"`
	Privatekey string `json:"privatekey"`
	Id         string `json:"id"`
	Version    int    `json:"version"`
}

func MyRequest(remoteUrl string, queryValues url.Values) string {
	// client := &http.Client{}
	uri, err := url.Parse(remoteUrl)
	if err != nil {
		fmt.Printf("parse remoteUrl %s error,%s", uri, err.Error())
		return ""
	}
	if queryValues != nil {
		values := uri.Query()
		if values != nil {
			for k, v := range values {
				queryValues[k] = v
			}
		}
		uri.RawQuery = queryValues.Encode()
	}
	//fmt.Println(uri.String())
	req, err := http.NewRequest("GET", uri.String(), nil)
	if err != nil {
		fmt.Println(err.Error())
		return ""
	}
	// fmt.Println(req.URL)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		return ""
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("\n Read body %s error %s \n", string(body), err.Error())
		return ""
	}
	return string(body)
}

func getJson(url string) string {

	//	url := fmt.Sprintf("http://web.ifichain.com:8080/irc20/computeAward?sno=%x", SnoStr)
	//	println(SnoStr)
	//	println(url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("request %s error,%s ", url, err.Error())
		return ""
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Do request %s error,because %s,", url, err.Error())
		return ""
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("Read Body %s error,because %s,", body, err.Error())
		return ""
	}

	return string(body)
}

func getRes(url3, sno, satellites, today, idCode, address string) string {
	data := make(url.Values)
	data["sno"] = []string{sno}
	data["satellites"] = []string{satellites}
	data["today"] = []string{today}
	data["idCode"] = []string{idCode}
	data["address"] = []string{address}
	return MyRequest(url3, data)
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {

		return true

	}
	return false
}
func getIdCode() string {
	idCodePath := "/usr/local/bin/p2puid"
	if !PathExists(idCodePath) {
		return ""
	}
	b, err := ioutil.ReadFile(idCodePath)
	if err != nil {
		fmt.Printf("get IdCode failed, cause read file: %s error: %s\n", idCodePath, err)
		return ""
	}
	s := string(b)
	l := len(s)

	i, j := 0, 0
	if l > 0 {
		for i = l - 1; i > -1; i-- {
			if s[i] != '\n' && s[i] != '\r' && s[i] != ' ' && s[i] != '	' {
				break
			}
		}
		if i == -1 {

			fmt.Printf("idCode is empty")
			return ""
		}

		for j = 0; j < l; j++ {
			if s[j] != '\n' && s[j] != '\r' && s[j] != ' ' && s[j] != '	' {
				break
			}
		}
	} else {
		fmt.Printf("idCode is empty")
		return ""
	}
	idCode := s[j : i+1]
	return idCode
}

func ifiAward(addr string) {
	if addr == "" {
		//fmt.Printf("address is empty")
		log("address is empty", 0, 0)
		return
	}
	idCode := getIdCode()
	if idCode == "" {
		return
	}
	//	fmt.Printf("idCode is %s\n", idCode)
	//	fmt.Printf("address is %s\n", addr)

	log(fmt.Sprintf("idCode is %s\n", idCode), 0, 1)
	log(fmt.Sprintf("address is %s\n", addr), 0, 1)
	flag := false
	day := time.Now().Day()
	for range time.Tick(time.Second * 7200) {
		min := time.Now().Minute()
		hour := time.Now().Hour()
		totalMins := hour*60 + min
		day1 := time.Now().Day()
		if day != day1 {
			flag = false
			day = day1
		}
		if !flag {
			if totalMins > 1229 && totalMins < 1350 {
				flag = true
				//fmt.Println("\n start to send \n")
				log("start to send", 0, 0)
				ComputeAward(idCode, addr)
			}
		}
	}
	/*	for range time.Tick(time.Second * 180) {
		fmt.Println("start to send")
		Compute(idCode, addr)
	}*/
}

func ComputeAward(idCode, address string) {
	url1 := "http://localhost:21002/api/sno/"
	url2 := "http://localhost:21002/api/sno/satellites"
	currentTime := time.Now()
	snoRes := getJson(url1)
	if snoRes == "" {
		println("get sno failed")
		return
	}
	satellitesRes := getJson(url2)
	if satellitesRes == "" {
		println("get satellites failed")
		return
	}
	today := currentTime.Format("2006-01-02 15:04:05")
	url3 := "http://112.35.192.13:8081/irc20/computeAward/"
	resJson := getRes(url3, snoRes, satellitesRes, today, idCode, address)

	if resJson != "" {
		println(resJson)
	} else {
		println("the server return none!")
		return
	}

	res := responseType{}
	err1 := json.Unmarshal([]byte(resJson), &res)
	if err1 != nil {
		//fmt.Printf("turn resJson to array error,%s\n",err1.Error())
		log(fmt.Sprintf("turn resJson to array error,%s", err1.Error()), 0, 1)
		return
	}
	if res.ResCode == 200 {
		//fmt.Printf(" send %f SLK to %x succeeded,the transactionHash is %s\n", res.Amount, address, res.TransactionHash)
		log(fmt.Sprintf("send %.4f SLK to %s succeeded,the transactionHash is %s\n", res.Amount, address, res.TransactionHash), 0, 1)
	} else {
		//fmt.Printf(" send SLK to %s failed\n", address)
		log(fmt.Sprintf("send SLK to %s failed,%s", address, err1.Error()), 0, 1)
	}
}

func getlogStr(wh uint, rank string, names []string) string {

	len1 := uint(len(names))
	if wh >= len1 {
		return ""
	}
	return fmt.Sprintf("%s=%s", rank, names[wh])
}

func log(msg string, lev uint, myType uint) {
	currentTime := time.Now()
	tm := currentTime.Format("2006-01-02 15:04:05")
	levelNames := []string{"info", "warning", "error"}
	typeNames := []string{"CPU reward", "Score reward"}
	level := getlogStr(lev, "level", levelNames)
	typeStr := getlogStr(myType, "type", typeNames)
	fmt.Printf("%s %s %s msg=%s\n", tm, level, typeStr, msg)

}
