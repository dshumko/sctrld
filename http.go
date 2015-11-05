package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"strings"
)

type TAnswerRuntime struct {
	Memheap    uint64 `json:"memheap"`
	Memidle    uint64 `json:"memidle"`
	Meminuse   uint64 `json:"meminuse"`
	Goroutines int    `json:"goroutines"`
	NextGC     uint64 `json:"nextgc"`
}

type TAnswerStat struct {
	Ip            string     `json:"ip"`
	Traffic       TIpTraffic `json:"stat"`
	Limit         TIpTraffic `json:"limit"`
	OffCountStart int        `json:"offstart"`
	OffCountStop  int        `json:"offstop"`
}

type TAnswerLimitAdd struct {
	Ip    string     `json:"ip"`
	Limit TIpTraffic `json:"limit_add"`
}

type TAnswerError struct {
	Error string `json:"error"`
}

func IPV4AddrToInt(addr string) (uint32, error) {
	parts := strings.Split(addr, ".")
	var ip uint32
	part, err := strconv.Atoi(parts[0])
	ip |= uint32(part) << 24
	part, err = strconv.Atoi(parts[1])
	ip |= uint32(part) << 16
	part, err = strconv.Atoi(parts[2])
	ip |= uint32(part) << 8
	part, err = strconv.Atoi(parts[3])
	ip |= uint32(part)
	return ip, err
}

func httpGetRuntime(w http.ResponseWriter, r *http.Request) {
	var vAnswer TAnswerRuntime

	memStats := runtime.MemStats{}
	runtime.ReadMemStats(&memStats)

	vAnswer.Memheap = memStats.HeapSys
	vAnswer.Memidle = memStats.HeapIdle
	vAnswer.Meminuse = memStats.HeapInuse
	vAnswer.Goroutines = runtime.NumGoroutine()
	vAnswer.NextGC = memStats.NextGC

	js, _ := json.Marshal(vAnswer)

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func httpGetStat(w http.ResponseWriter, r *http.Request) {
	var vAnswer TAnswerStat
	var vError TAnswerError
	var js []byte
	w.Header().Set("Content-Type", "application/json")
	v := r.URL.Query()
	ip_adr := v.Get("ip")
	ip, err := IPV4AddrToInt(ip_adr)
	if err != nil {
		vError.Error = fmt.Sprintf("ip format error %s", ip_adr)
		js, _ = json.Marshal(vError)
	} else {
		response := GetIpInfo(uint32(ip))
		if response.Found {
			vAnswer.Ip = ip_adr
			vAnswer.Limit = response.IpRec.Limit
			vAnswer.OffCountStart = response.IpRec.OffCountStart
			vAnswer.OffCountStop = response.IpRec.OffCountStop
			vAnswer.Traffic = response.IpRec.Traffic
			js, _ = json.Marshal(vAnswer)

		} else {
			vError.Error = fmt.Sprintf("%s not found", ip_adr)
			js, _ = json.Marshal(vError)
		}
	}

	w.Write(js)
}

func httpAddLimit(w http.ResponseWriter, r *http.Request) {
	var vAnswer TAnswerLimitAdd
	var vError TAnswerError
	var js []byte
	w.Header().Set("Content-Type", "application/json")
	v := r.URL.Query()
	ip_adr := v.Get("ip")
	ip, err := IPV4AddrToInt(ip_adr)
	if err != nil {
		vError.Error = fmt.Sprintf("ip format error %s", ip_adr)
		js, _ = json.Marshal(vError)
	} else {
		traf, err := strconv.Atoi(v.Get("limit"))
		if err != nil {
			vError.Error = fmt.Sprintf("limit error %s", v.Get("limit"))
			js, _ = json.Marshal(vError)
		} else {
			AddLimitToIp(ip, TIpTraffic(traf))
			vAnswer.Ip = ip_adr
			vAnswer.Limit = TIpTraffic(traf)
			js, _ = json.Marshal(vAnswer)
		}
	}
	w.Write(js)
}

func httpSetLimit(w http.ResponseWriter, r *http.Request) {
	var vAnswer TAnswerStat
	var vError TAnswerError
	var js []byte
	var rec TipRecord
	w.Header().Set("Content-Type", "application/json")
	v := r.URL.Query()
	ip_adr := v.Get("ip")
	IPAddress := net.ParseIP(ip_adr)

	if IPAddress == nil {
		vError.Error = fmt.Sprintf("ip format error %s", ip_adr)
		js, _ = json.Marshal(vError)
	} else {
		traf, err := strconv.Atoi(v.Get("limit"))
		if err != nil {
			vError.Error = fmt.Sprintf("limit error %s", v.Get("limit"))
			js, _ = json.Marshal(vError)
		} else {
			offstart, err := strconv.Atoi(v.Get("offstart"))
			if err != nil {
				vError.Error = fmt.Sprintf("offstart error %s", v.Get("offstart"))
				js, _ = json.Marshal(vError)
			} else {
				offstop, err := strconv.Atoi(v.Get("offstop"))
				if err != nil {
					vError.Error = fmt.Sprintf("offstop error %s", v.Get("offstop"))
					js, _ = json.Marshal(vError)
				} else {
					rec.Limit = TIpTraffic(traf)
					rec.OffCountStart = offstart
					rec.OffCountStop = offstop
					SetLimitToIp(netIpToInt(IPAddress), rec)
					vAnswer.Ip = ip_adr
					vAnswer.Traffic = rec.Traffic
					vAnswer.Limit = rec.Limit
					vAnswer.OffCountStart = rec.OffCountStart
					vAnswer.OffCountStop = rec.OffCountStop
					js, _ = json.Marshal(vAnswer)
				}
			}
		}
	}
	w.Write(js)
}
