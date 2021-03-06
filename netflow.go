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

func intToNetIp(intAddr uint32) net.IP {
	return net.IPv4(
		byte(intAddr>>24),
		byte(intAddr>>16),
		byte(intAddr>>8),
		byte(intAddr))
}

func netIpToInt(ip net.IP) uint32 {
	p := ip.To4()
	vip := uint32(p[0]) << 24
	vip |= uint32(p[1]) << 16
	vip |= uint32(p[2]) << 8
	vip |= uint32(p[3])
	return vip
}

func intToStrIP(intAddr uint32) string {
	return intToNetIp(intAddr).String()
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
		id = 0
		err := binary.Read(buf, binary.BigEndian, &record)
		if err != nil {
			log.Fatalf("binary.Read failed: %v\n", err)
		}

		if (record.Ipv4SrcAddr >= minCheckIP) && (record.Ipv4SrcAddr <= maxCheckIP) {
			id = record.Ipv4SrcAddr
		} else {
			if (record.Ipv4DstAddr >= minCheckIP) && (record.Ipv4DstAddr <= maxCheckIP) {
				id = record.Ipv4DstAddr
			}
		}

		if id > 0 {
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
