package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
)

type TConfiguration struct {
	NetflowAdress     string
	NetflowBufferSize int
	WebPort           string
	CmdDown           string
	CmdUp             string
}

type TIpTraffic uint64

type TipRecord struct {
	Traffic       TIpTraffic
	Limit         TIpTraffic
	InFullSpeed   bool
	OffCountStart int
	OffCountStop  int
}

type TAddTraffic struct {
	Id      uint32
	Traffic TIpTraffic
}

type setLimit struct {
	Id    uint32
	Value TipRecord
}

type getRequest struct {
	Id       uint32
	Response chan getResponse
}

type getResponse struct {
	Found bool
	IpRec TipRecord
}

var (
	puts       chan TAddTraffic
	gets       chan getRequest
	sets       chan setLimit
	maxCheckIP uint32
	minCheckIP uint32
)

func AddLimitToIp(ip uint32, value TIpTraffic) {
	response := GetIpInfo(ip)
	response.IpRec.Limit += value

	sets <- setLimit{
		Id:    ip,
		Value: response.IpRec,
	}
}

func SetLimitToIp(ip uint32, value TipRecord) {
	sets <- setLimit{
		Id:    ip,
		Value: value,
	}
}

func AddTraffic(id uint32, value TIpTraffic) {
	puts <- TAddTraffic{
		Id:      id,
		Traffic: value,
	}
}

func GetIpInfo(id uint32) getResponse {
	getResponses := make(chan getResponse)
	gets <- getRequest{
		Id:       id,
		Response: getResponses,
	}
	response := <-getResponses
	return response
}

func RunCMD(Id uint32, ipRec TipRecord, cmd string) {
	// Тут блокируем IP
	ips := intToStrIP(Id)
	exec.Command(cmd, ips).Run()
	log.Printf("ip:%s\n%v\ncmd: %s %s\n", ips, ipRec, cmd, ips)
}

func main() {
	var sctrldCfg TConfiguration
	var cur_hour int
	var old_limit TIpTraffic

	// чтоб не проверять все ip установим границы
	maxCheckIP = 0
	minCheckIP = 4294967295

	file, _ := os.Open("sctrld.config")
	decoder := json.NewDecoder(file)
	err := decoder.Decode(&sctrldCfg)
	if err != nil {
		sctrldCfg.NetflowAdress = "0.0.0.0:2055"
		sctrldCfg.WebPort = "8080"
		sctrldCfg.NetflowBufferSize = 212992
	}
	file.Close()
	log.Printf("Start netflow listening on %v\n", sctrldCfg.NetflowAdress)
	go ListenNetflow(sctrldCfg.NetflowAdress, sctrldCfg.NetflowBufferSize)

	storage := map[uint32]TipRecord{}

	puts = make(chan TAddTraffic)
	gets = make(chan getRequest)
	sets = make(chan setLimit)

	http.HandleFunc("/runtime/", httpGetRuntime) // /v1/get/?ip=127.0.0.1
	http.HandleFunc("/v1/get/", httpGetStat)     // /v1/get/?ip=127.0.0.1
	http.HandleFunc("/v1/add/", httpAddLimit)    // /v1/add/?ip=127.0.0.1&limit=100
	http.HandleFunc("/v1/set/", httpSetLimit)    // /v1/set/?ip=127.0.0.1&limit=200
	log.Printf("Start web listening on %v\n", sctrldCfg.WebPort)
	go http.ListenAndServe(":"+sctrldCfg.WebPort, nil)

	//go AddLimitToIp(2130706433, 100) // 127.0.0.1
	//storage[2130706433] = TipRecord{0, 100}
	/*
		создаем 10 канадов для работы со своей зоной IP
		и каждую зону передавать по своим каналам
		а также для каждой зоны будет свой map
		что ускорит обработку и запаралелит еще больше

		проверку IP нужно сделать через AND 53EF FFFF и SHR 16

		53EF FFFF = not AC100000

		172.16.0.0	AC100000	0
		172.17.0.0	AC110000	1
		172.18.0.0	AC120000	2
		172.19.0.0	AC130000	3
		172.20.0.0	AC140000	4
		172.21.0.0	AC150000	5
		172.22.0.0	AC160000	6
		172.23.0.0	AC170000	7
		172.24.0.0	AC180000	8
		172.25.0.0	AC190000	9
		172.26.0.0	AC1A0000	10
		172.27.0.0	AC1B0000	11
		172.28.0.0	AC1C0000	12
	*/

	for {
		select {
		case put := <-puts:
			value, found := storage[put.Id]
			if found {
				cur_hour = time.Now().Hour()
				if !((cur_hour >= value.OffCountStart) && (cur_hour < value.OffCountStop)) {
					old_limit = value.Traffic
					value.Traffic += put.Traffic
					storage[put.Id] = value
					if (value.Traffic > value.Limit) && (old_limit < value.Limit) {
						go RunCMD(put.Id, value, sctrldCfg.CmdDown)
					}
				} else {
					if !(value.InFullSpeed) {
						value.InFullSpeed = true
						storage[put.Id] = value
						go RunCMD(put.Id, value, sctrldCfg.CmdUp)
					}
				}
			}
		case get := <-gets:
			value, found := storage[get.Id]
			get.Response <- getResponse{
				IpRec: value,
				Found: found,
			}
		case set := <-sets:
			if set.Id > maxCheckIP {
				maxCheckIP = set.Id
			}
			if set.Id < minCheckIP {
				minCheckIP = set.Id
			}
			value, _ := storage[set.Id]
			old_limit = set.Value.Limit
			value.Limit = set.Value.Limit
			value.OffCountStart = set.Value.OffCountStart
			value.OffCountStop = set.Value.OffCountStop
			storage[set.Id] = value
			if (old_limit < value.Traffic) && (value.Limit > value.Traffic) {
				go RunCMD(set.Id, value, sctrldCfg.CmdUp)
			}
		}
	}
}
