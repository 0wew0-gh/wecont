package wecont

import (
	"bufio"
	"log"
	"net"
	"os"
)

func Children_init(infoLog *log.Logger, debugLog *log.Logger, errLog *log.Logger, ping func(net.Conn), exit func(net.Conn)) {
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
		go listenCommand(conn, ping, exit)
	}
}

func listenCommand(conn net.Conn, ping func(net.Conn), exit func(net.Conn)) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		msg := scanner.Text()
		l.Info.Println("get msg:", msg)
		switch msg {
		case "PING":
			ping(conn)
		case "STOP":
			exit(conn)
			os.Remove(SocketAddr)
			os.Exit(0)
		}
	}
}
