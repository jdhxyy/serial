// Copyright 2022-2022 The jdh99 Authors. All rights reserved.
// 串口通信模块
// Authors: jdh99 <jdh821@163.com>

package serial

import (
	"errors"
	"github.com/jdhxyy/lagan"
	serial "github.com/tarm/goserial"
	"io"
)

const (
	tag = "serial"

	fifoLen = 8192
	frameMaxLen = 4096
)

type tSerial struct {
	index int
	com io.ReadWriteCloser
	fifo chan []uint8
}

// RxCallback 接收回调函数
type RxCallback func(index int, data []uint8)

var observers []RxCallback
var serials map[int]*tSerial

func init() {
	serials = make(map[int]*tSerial)
}

// Open 打开串口.index是用户定义的串口序号,后续发送和接收使用串口序号来识别对应的串口
func Open(index int, com string, baudRate int) error {
	s, ok := serials[index]
	if ok == true {
		lagan.Error(tag, "index:%d com:%s baud rate:%d already used", index, com, baudRate)
		return errors.New("already used")
	}

	serials[index] = new(tSerial)
	s = serials[index]

	c := &serial.Config{Name: com, Baud: baudRate}
	serialCom, err := serial.OpenPort(c)
	if err != nil {
		lagan.Error(tag, "load failed.open serial failed.com:%s baud rate:%d.err:%s", com, baudRate, err)
		return err
	}
	lagan.Info(tag, "open serial success.com:%s baud rate:%d", com, baudRate)
	s.index = index
	s.com = serialCom
	s.fifo = make(chan []uint8, fifoLen)

	go rx(s)
	go tx(s)

	return nil
}

func rx(s *tSerial) {
	data := make([]uint8, frameMaxLen)
	for {
		num, err := s.com.Read(data)
		if err != nil {
			lagan.Error(tag, "read failed:%v", err)
			continue
		}
		if num <= 0 {
			continue
		}
		notifyObservers(s.index, data[:num])
	}
}

func notifyObservers(index int, data []uint8) {
	n := len(observers)
	for i := 0; i < n; i++ {
		observers[i](index, data)
	}
}

func tx(s *tSerial) {
	for {
		data := <-s.fifo

		lagan.Debug(tag, "serial:%d send len:%d", s.index, len(data))
		lagan.PrintHex(tag, lagan.LevelDebug, data)

		_, err := s.com.Write(data)
		if err != nil {
			lagan.Error(tag, "udp send error:%v", err)
			return
		}
	}
}

// RegisterObserver 注册观察者
func RegisterObserver(callback RxCallback) {
	observers = append(observers, callback)
}

// Send 发送数据
func Send(index int, data []uint8) {
	s, ok := serials[index]
	if ok == false {
		lagan.Error(tag, "index:%d is not open", index)
		return
	}
	s.fifo <- data
}
