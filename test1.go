package main

import (
	"fmt"
	"net"
	"time"
)
func main(){
		var who string
		//func Dial(network, address string) (Conn, error)
		conn,err:=net.Dial("tcp",":8080")
		if err!=nil {
			fmt.Println("net.Dial err:",err)
			return
		}
		fmt.Println("client与server链接建立成功！")
		for {
				fmt.Scanln(&who)
				sendData:=[]byte(who)
				cnt, err := conn.Write(sendData)
				if err != nil {
					fmt.Println("conn.write err:", err)
					return
				}
				fmt.Println("Client===》server cnt:", cnt, ",data:", string(sendData))

				//接收服务器返回的数据
				buf := make([]byte, 2048)
				//Read(b []byte) (n int, err error)
				cnt, err = conn.Read(buf)
				if err != nil {
					fmt.Println("conn.read err:", err)
					return
				}
				fmt.Println("server ===>client,cnt:", cnt, ",data:", string(buf[0:cnt]))
				time.Sleep(1*time.Second)


		}
	    conn.Close()

}



