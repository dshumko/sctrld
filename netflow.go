package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"net"
)

type TflowHeader struct {
	Version          uint16
	FlowRecords      uint16
	Uptime           uint32
	UnixSec          uint32
	UnixNsec         uint32
	FlowSeqNum       uint32
	EngineType       uint8
	EngineId         uint8
	SamplingInterval uint16
}

type TflowRecord struct {
	Ipv4SrcAddr   uint32
	Ipv4DstAddr   uint32
	Ipv4NextHop   uint32
	InputSnmp     uint16
	OutputSnmp    uint16
	InPkts        uint32
	InBytes       uint32
	FirstSwitched uint32
	LastSwitched  uint32
	L4SrcPort     uint16
	L4DstPort     uint16
	_             uint8
	TcpFlags      uint8
	Protocol      uint8
	SrcTos        uint8
	SrcAs         uint16
	DstAs         uint16
	SrcMask       uint8
	DstMask       uint8
	_             uint16
}

func intToIPv4Addr(intAddr uint32) net.IP {
	return net.IPv4(
		byte(intAddr>>24),
		byte(intAddr>>16),
		byte(intAddr>>8),
		byte(intAddr))
}

func intToStrIP(intAddr uint32) string {
	return intToIPv4Addr(intAddr).String()
}

func handleNetFlowPacket(buf *bytes.Buffer, remoteAddr *net.UDPAddr) {
	var id uint32

	header := TflowHeader{}
	err := binary.Read(buf, binary.BigEndian, &header)
	if err != nil {
		log.Fatalf("Error:", err)
	}

	for i := 0; i < int(header.FlowRecords); i++ {
		record := TflowRecord{}
		err := binary.Read(buf, binary.BigEndian, &record)
		if err != nil {
			log.Fatalf("binary.Read failed: %v\n", err)
		}

		// 172.16.0.0     - 2886729728
		// 172.26.255.255 - 2887450623
		if (record.Ipv4SrcAddr < 2887450623) && (record.Ipv4SrcAddr > 2886729728) {
			id = record.Ipv4SrcAddr
		} else {
			if (record.Ipv4DstAddr < 2887450623) && (record.Ipv4DstAddr > 2886729728) {
				id = record.Ipv4DstAddr
			}
		}
		// record.Ipv4SrcAddrInt
		// record.Ipv4DstAddrInt
		// record.InBytes
		if id > 0 {
			//log.Printf("Start web listening on %v\n", SccCfg.WebPort)
			//buf, err := json.Marshal(record)
			//if err != nil {
			//	log.Fatalf("json.Marshal failed: %v\n", err)
			//}

			//fmt.Printf("%v\n", string(buf))
			AddTraffic(id, TIpTraffic(record.InBytes))
		}

	}
}

func ListenNetflow(inSource string, receiveBufferSizeBytes int) {
	/* Start listerning on the specified port */
	// log.Printf("Start netflow listening on %v\n", inSource)
	addr, err := net.ResolveUDPAddr("udp", inSource)
	if err != nil {
		log.Fatalf("Error: %v\n", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalln(err)
	}

	err = conn.SetReadBuffer(receiveBufferSizeBytes)
	if err != nil {
		log.Fatalln(err)
	}

	defer conn.Close()

	for {
		buf := make([]byte, 4096)
		rlen, remote, err := conn.ReadFromUDP(buf)

		if err != nil {
			log.Fatalf("Error: %v\n", err)
		}

		stream := bytes.NewBuffer(buf[:rlen])

		go handleNetFlowPacket(stream, remote)
	}
}
