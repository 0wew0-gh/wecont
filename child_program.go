package wecont

import (
	"bufio"
	"log"
	"net"
	"os"
)

func Children_init(infoLog *log.Logger, debugLog *log.Logger, errLog *log.Logger, message func(net.Conn, string)) {
	l = Logger{Info: infoLog, Debug: debugLog, Error: errLog}
	// 先删除旧的 sock 文件
	os.Remove(SocketAddr)
	// 1. 监听本地端口进行通讯
	ln, err := net.Listen(NetType, SocketAddr)
	if err != nil {
		l.Error.Println("listen error:", err)
		os.Exit(1)
	}
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			l.Error.Println("listen accept error:", err)
			continue
		}
		go listenCommand(conn, message)
	}
}

func Dispose() {
	os.Remove(SocketAddr)
}

func listenCommand(conn net.Conn, message func(net.Conn, string)) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		msg := scanner.Text()
		l.Info.Println("get msg:", msg)
		message(conn, msg)
	}
}
