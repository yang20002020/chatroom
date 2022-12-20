package main

import (
	"fmt"
	"net"
)

type User struct {
	//名字
	name string
	//唯一的id
	id string
	//管道
	msg chan string
}

func main() {
	//1. 创建服务器
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("net.Listen err:", err)
		return
	}

	fmt.Println("服务器启动成功!")
	//for循环 实时监听 建立多个连接
	for {

		fmt.Println("主go程监听中.....")
		//2. 监听
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("listener.Accept err:", err)
			return
		}
		//建立链接
		fmt.Println("建立链接成功！")

		// 启动处理业务的go程
		go handler(conn)
	}

}

// 处理具体业务
func handler(conn net.Conn) {

	fmt.Println("启动业务...")

	// for 循环 处理客户端发送过来的数据 多次处理数据
	for {
		//具体业务逻辑
		buf := make([]byte, 1024)
		//读取客户端发送过来的请求数据)
		cnt, err := conn.Read(buf)
		if err != nil {
			fmt.Println("conn.Read err:", err)
			return
		}

		fmt.Println("服务器接收客户端发送过来的数据为：", string(buf[:cnt]), ",cnt:", cnt)
	}
}
