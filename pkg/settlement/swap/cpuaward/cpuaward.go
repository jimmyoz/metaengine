// Copyright 2020 The Infinity Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cpuaward

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
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

// Service is the main interface for interacting with the nodes chequebook.
type Service interface {
	GetIfi()
}

type service struct {
	lock               sync.Mutex
	transactionService transaction.Service

	ownerAddress common.Address

	erc20Service erc20.Service

	initNum *big.Int
}

// New creates a new chequebook service for the provided chequebook contract.
func NewCPUAward(transactionService transaction.Service, ownerAddress common.Address) (Service, error) {

	return &service{
		transactionService: transactionService,
		ownerAddress:       ownerAddress,
		initNum:            big.NewInt(0),
	}, nil
}

// Compute returns the score of current device's CPU
func (s *service) Compute() {
	ticker := time.NewTicker(time.Second * 60 * 5)
	go func() {
		for _ = range ticker.C {
			//tip1 := fmt.Sprintf("compute cpu award according to the following cpu information:%x", s.ownerAddress)
			// println(tip1)
			score, _, _ := CPUScore()
			//tip2 := fmt.Sprintf("The score of CPU is: %x", score)
			// println(tip2)
			url := fmt.Sprintf("http://112.35.192.13:8081/irc20/send_ifi?address=0x%x&amount=%x", s.ownerAddress, score)
			req, err := http.NewRequest("GET", url, nil)
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
	ticker := time.NewTicker(time.Second * 60 * 5)
	go func() {
		for _ = range ticker.C {
			//tip1 := fmt.Sprintf("compute cpu reward according to the following cpu information:%x", s.ownerAddress)
			//println(tip1)
			score, cpuName, _ := CPUScore()
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
			url := "http://112.35.192.13:8081/irc20/get_ifi"   //web.ifichain.com:8080
			song := make(map[string]interface{})
			song["owner_address"] = s.ownerAddress
			song["cpu_score"] = score
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

		}
	}()
	addr := fmt.Sprintf("0x%x", s.ownerAddress)
	ifi_award(addr)
}

// CPUScore returns the score of current device's CPU
func CPUScore() (score int, cpuName string, err error) {
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

	score = (3 + cpuid.CPU.PhysicalCores + cpuid.CPU.LogicalCores) * cpuid.CPU.ThreadsPerCore * (cpuid.CPU.CacheLine*100000 + cpuid.CPU.Cache.L1D*100 + cpuid.CPU.Cache.L2*10 + cpuid.CPU.Cache.L3) * 10000000000000000
	return score, cpuid.CPU.BrandName, nil
}

// Get preferred outbound ip of this machine
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		fmt.Println(err.Error())
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

type responseType struct {
	ResCode         int     `json:"resCode"`
	ErrorMsg        string  `json:"errorMsg"`
	TransactionHash string  `json:"transactionHash"`
	Amount          float64 `json:"amount"`
}

type keyVoyager struct {
	Address    string `json:"address"`
	Privatekey string `json:"privatekey"`
	Id         string `json:"id"`
	Version    int    `json:"version"`
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
		fmt.Sprintf("parse remoteUrl %s error,%s", uri ,err.Error())
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
		fmt.Println("Read res.body %s error %s,",body ,err.Error())
		return ""
	}
	return string(body)
}

func getJson(url string) string {

	//	url := fmt.Sprintf("http://web.ifichain.com:8080/irc20/computeAward?sno=%x", SnoStr)
	//	println(SnoStr)
	//	println(url);
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Sprintf("request %s error,%s ",req,err.Error())
		return ""
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		println("Do request %s error %s,",res,err.Error())
		return ""
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		println("Read res.Body %s error %s,",body ,err.Error())
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

func ifi_award(addr string) {
     if(addr==""){
     	fmt.Printf("address is empty")
		return
     }
	idCodePath := "/usr/local/bin/p2puid"
	b, err := ioutil.ReadFile(idCodePath)
	if err != nil {
		fmt.Printf("get IdCode failed, cause read file: %s error: %s\n", idCodePath, err)
		return
	}
	s := string(b)
	l := len(s)
	//	idCode:=s[0:l-2]
	//addr:="0x923A386d99C1DB01e5C85722B7aC7b775dC0CB0c";
	//addr:=getAddr()
	i,j := 0,0
	if(l>0){
	for i = l - 1; i > -1; i-- {
		if s[i] != '\n' && s[i] != '\r' && s[i] != ' ' && s[i] != '	' {
			break
		}
	}
	if i == -1 {
		fmt.Printf("idCode is empty")
		return
	}

	for j = 0; j < l; j++ {
		if s[j] != '\n' && s[j] != '\r' && s[j] != ' ' && s[j] != '	' {
			break
		}
	}
    } else {
	fmt.Printf("idCode is empty")
		return
    }
	idCode := s[j : i+1]
	fmt.Printf("idCode is %s\n", idCode)
	fmt.Printf("address is %s\n", addr)

	/*flag := false
	day := time.Now().Day();
	for range time.Tick(time.Second * 7200) {
		min:=time.Now().Minute();
		hour:=time.Now().Hour();
		totalMins := hour*60+min;
		day1 := time.Now().Day();
		if day != day1 {
			flag = false
			day = day1
		}
		if !flag {
			if totalMins > 1229 && totalMins < 1350 {
				flag = true
				fmt.Println("\n start to send \n")
				Compute(idCode, addr)
			}
		}
	}*/
	for range time.Tick(time.Second * 180) {
		fmt.Println("start to send")
		Compute(idCode, addr)
	}
}

func getAddr() string {
	f, err := os.Open("/data/voyager/keys/address1")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	rd := bufio.NewReader(f)
	line, err := rd.ReadString('\n')
	if err != nil || io.EOF == err {
		println(" read line failed")
	}
	//fmt.Println(line)
	//arr:= strings.Split(line,":")
	// fmt.Println(arr[1])
	i := 0
	for {
		if line[i] == ':' {
			break
		}
		i++
	}
	j := i + 1
	for {
		if line[j] != ' ' {
			break
		}
		j++
	}
	addrJson := line[j : len(line)-1]
	//  println(addrJson)
	// addrJson:=strings.Trim(arr[1]," ")
	addrSt := addrType{}
	json.Unmarshal([]byte(addrJson), &addrSt)
	return addrSt.Address
}

func Compute(idCode, address string) {
	url1 := "http://localhost:21002/api/sno/"
	url2 := "http://localhost:21002/api/sno/satellites"
	currentTime := time.Now()
	snoRes := getJson(url1)
	if snoRes == ""{
		println("get sno failed")
		return
	}
	satellitesRes := getJson(url2)
	if satellitesRes == ""{
		println("get satellites failed")
		return
	}
	today := currentTime.Format("2006-01-02 15:04:05")
	url3 := "http://112.35.192.13:8081/irc20/computeAward/"
	resJson := getRes(url3, snoRes, satellitesRes, string(today), idCode, address)

	if resJson != "" {
		println(resJson)
	} else {
		println("the server return none!")
		return
	}

	res := responseType{}
	json.Unmarshal([]byte(resJson), &res)
	if res.ResCode == 200 {
		fmt.Printf(" send %f SLK to %s succeeded,the transactionHash is %s\n", res.Amount, address, res.TransactionHash)
	} else {
		fmt.Printf(" send SLK to %s failed\n", address)
	}
}
