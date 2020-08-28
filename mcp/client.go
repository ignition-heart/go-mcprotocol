package mcp

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net"
)

type Client interface {
	Read(deviceName string, offset, numPoints int64) ([]byte, error)
	Write(deviceName string, offset int64, writeData []byte) error
	HealthCheck() error
}

// client3E is 3E frame mcp client
type client3E struct {
	// PLC address
	tcpAddr *net.TCPAddr
	// PLC station
	stn *station
}

func New3EClient(host string, port int, stn *station) (Client, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%v:%v", host, port))
	if err != nil {
		return nil, err
	}
	return &client3E{tcpAddr: tcpAddr, stn: stn}, nil
}

// MELSECコミュニケーションプロトコル p180
// 11.4折返しテスト
func (c *client3E) HealthCheck() error {
	requestStr := c.stn.BuildHealthCheckRequest()

	// binary protocol
	payload, err := hex.DecodeString(requestStr)
	if err != nil {
		return err
	}

	// TODO Keep-Alive
	conn, err := net.DialTCP("tcp", nil, c.tcpAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Send message
	if _, err = conn.Write(payload); err != nil {
		return err
	}

	// Receive message
	readBuff := make([]byte, 30)
	readLen, err := conn.Read(readBuff)
	if err != nil {
		return err
	}

	resp := readBuff[:readLen]

	if readLen != 18 {
		return errors.New("plc connect test is fail: return length is [" + fmt.Sprintf("%X", resp) + "]")
	}

	// decodeString is 折返しデータ数ヘッダ[1byte]
	if "0500" != fmt.Sprintf("%X", resp[11:13]) {
		return errors.New("plc connect test is fail: return header is [" + fmt.Sprintf("%X", resp[11:13]) + "]")
	}

	//  折返しデータ[5byte]=ABCDE
	if "4142434445" != fmt.Sprintf("%X", resp[13:18]) {
		return errors.New("plc connect test is fail: return body is [" + fmt.Sprintf("%X", resp[13:18]) + "]")
	}

	return nil
}

// Read is send read command to remote plc by mc protocol
// deviceName is device code name like 'D' register.
// offset is device offset addr.
// numPoints is number of read device points.
func (c *client3E) Read(deviceName string, offset, numPoints int64) ([]byte, error) {
	requestStr := c.stn.BuildReadRequest(deviceName, offset, numPoints)

	// TODO binary protocol
	payload, err := hex.DecodeString(requestStr)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTCP("tcp", nil, c.tcpAddr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Send message
	if _, err = conn.Write(payload); err != nil {
		return nil, err
	}

	// Receive message
	readBuff := make([]byte, 22+2*numPoints) // 22 is response header size. [sub header + network num + unit i/o num + unit station num + response length + response code]
	readLen, err := conn.Read(readBuff)
	if err != nil {
		return nil, err
	}

	return readBuff[:readLen], nil
}

// Write is send write command to remote plc by mc protocol
// deviceName is device code name like 'D' register.
// offset is device offset addr.
// writeData is data to write.
// If writeData is larger than 4 bytes, the fifth and subsequent bytes are ignored.
func (c *client3E) Write(deviceName string, offset int64, writeData []byte) error {
	requestStr := c.stn.BuildWriteRequest(deviceName, offset, writeData)
	payload, err := hex.DecodeString(requestStr)
	if err != nil {
		return err
	}
	conn, err := net.DialTCP("tcp", nil, c.tcpAddr)
	if err != nil {
		return err
	}
	defer conn.Close()
	// Send message
	if _, err = conn.Write(payload); err != nil {
		return err
	}
	// FIX_ME: Receive return message
	/*
	   readBuff := make([]byte, 30)
	   readLen, err := conn.Read(readBuff)
	   if err != nil {
	       return err
	   }
	*/
	return nil
}
