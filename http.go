package main

import (
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
)

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
	memStats := runtime.MemStats{}
	runtime.ReadMemStats(&memStats)
	fmt.Fprintf(w, "{\"Memheap\": %d, \"Memidle\": %d, \"Meminuse\": %d, \"goroutines\": %d, \"NextGC:\": %d, \"PauseTotalNs\": %d}",
		memStats.HeapSys,
		memStats.HeapIdle,
		memStats.HeapInuse,
		runtime.NumGoroutine(),
		memStats.NextGC,
		memStats.PauseTotalNs,
	)
}

func httpGetStat(w http.ResponseWriter, r *http.Request) {
	v := r.URL.Query()
	ip_adr := v.Get("ip")
	ip, err := IPV4AddrToInt(ip_adr)
	if err != nil {
		fmt.Fprintf(w, "{\"error\": \"ip format error %s\"}", ip_adr)
	} else {
		response := GetIpInfo(uint32(ip))
		if response.Found {
			fmt.Fprintf(w, "{\"ip\": \"%s\", \"stat\": %d, \"limit\": %d, \"offstart\": %d, \"offstop\": %d}", ip_adr, response.IpRec.Traffic, response.IpRec.Limit, response.IpRec.OffCountStart, response.IpRec.OffCountStop)
		} else {
			fmt.Fprintf(w, "{\"errorip\": \"%s not found\"}", ip_adr)
		}
	}
}

func httpAddLimit(w http.ResponseWriter, r *http.Request) {
	v := r.URL.Query()
	ip, err := IPV4AddrToInt(v.Get("ip"))
	if err != nil {
		fmt.Fprintf(w, "{\"error\": \"ip format error %s\"}", v.Get("ip"))
	} else {
		traf, err := strconv.Atoi(v.Get("limit"))
		if err != nil {
			fmt.Fprintf(w, "{\"error\": \"limit error %s\"}", v.Get("limit"))
		} else {
			AddLimitToIp(ip, TIpTraffic(traf))
			fmt.Fprintf(w, "{\"ip\": \"%s\", \"limit_add\": %d}", v.Get("ip"), traf)
		}
	}

}

func httpSetLimit(w http.ResponseWriter, r *http.Request) {
	var rec TipRecord
	v := r.URL.Query()
	ip, err := IPV4AddrToInt(v.Get("ip"))
	if err != nil {
		fmt.Fprintf(w, "{\"error\": \"ip format error %s\"}", v.Get("ip"))
	} else {
		traf, err := strconv.Atoi(v.Get("limit"))
		if err != nil {
			fmt.Fprintf(w, "{\"error\": \"limit error %s\"}", v.Get("limit"))
		} else {
			offstart, err := strconv.Atoi(v.Get("offstart"))
			if err != nil {
				fmt.Fprintf(w, "{\"error\": \"offstart error %s\"}", v.Get("offstart"))
			} else {
				offstop, err := strconv.Atoi(v.Get("offstop"))
				if err != nil {
					fmt.Fprintf(w, "{\"error\": \"offstop error %s\"}", v.Get("offstop"))
				} else {
					rec.Limit = TIpTraffic(traf)
					rec.OffCountStart = offstart
					rec.OffCountStop = offstop
					SetLimitToIp(ip, rec)
					fmt.Fprintf(w, "{\"ip\": \"%s\", \"stat\": %d, \"limit\": %d, \"offstart\": %d, \"offstop\": %d}", v.Get("ip"), rec.Traffic, rec.Limit, rec.OffCountStart, rec.OffCountStop)
				}
			}
		}
	}
}
