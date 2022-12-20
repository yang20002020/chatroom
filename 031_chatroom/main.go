package main

import (
	"fmt"
	"net"
	"strings"
	"time"
)

type User struct {
	//名字
	name string
	//唯一的id
	id string
	//管道
	msg chan string
}

// 创建一个全局的map结构，用于保存所有的用户 //allUsers[newUser.id] = newUser
var allUsers = make(map[string]User)

// 定义一个message 全局通道，用于接收任何人发送过来的消息
var message = make(chan string, 10)

func main() {
	//1. 创建服务器
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("net.Listen err:", err)
		return
	}
	//启动全局唯一的go程，负责监听message通道，写给所有的用户
	go broadcast()
	fmt.Println("服务器启动成功!")
	for {
		fmt.Println("=====>主go程监听中.....")
		//2. 监听
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("listener.Accept err:", err)
			return
		}
		//建立链接
		fmt.Println("建立链接成功！")

		// 启动处理业务的go程 一个用户对应一个handler
		go handler(conn)
	}

}

// 处理具体业务  向管道写入数据
func handler(conn net.Conn) {
	fmt.Println("启动业务...")

	//客户端与服务器建立链接的时候,会有ip和port==》当成user的id
	clientAddr := conn.RemoteAddr().String()
	fmt.Println("clientAddr:", clientAddr)
	//创建user 一个handler 对应一个user
	newUser := User{
		name: clientAddr,            //id,我们不会修改，这个作为在map中的key
		id:   clientAddr,            //name 可以修改，会提供rename命令修改；建立链接时，初始值与id相同
		msg:  make(chan string, 10), //注意需要make空间，否则无法写入数据
	}
	//添加user到map结构
	allUsers[newUser.id] = newUser
	//定义一个退出信号，用于监听client退出
	var isQuit = make(chan bool)

	//创建一个用于重置计数器的管道，用于告知watch函数，当前用户正在输入
	var restTimer = make(chan bool)

	//启动go程，负责监听退出信号 每一个用户都有watch编程
	go watch(&newUser, conn, isQuit, restTimer)

	//每一个用户应该还有一个监听自己msg管道的go程，负责将msg信息返回给客户端
	//启动go程,负责将msg信息返回给客户端
	go wrtieBackToClient(&newUser, conn)

	//向通道message写入数据，当前用户上线的消息，用于通知所有人（广播）
	loginInfo := fmt.Sprintf("[%s]:[%s]===>上线了login！！！", newUser.id, newUser.name)
	message <- loginInfo //向通道写入数据
	//for 循环 处理客户端发送过来的数据 多次处理数据
	for {
		//具体业务逻辑
		buf := make([]byte, 1024)
		//读取客户端发送过来的请求数据，
		cnt, err := conn.Read(buf)
		if cnt == 0 {
			fmt.Println("客户端主动关闭ctrl+c,准备退出！err；", err)
			//map删除，用户，conn，close掉
			//服务器还可以主动退出
			//在这里不进行真正的退出动作，而是发送一个退出信号，统一做出退出动作，可以使用新的管道来做信号传递
			isQuit <- true
		}

		if err != nil {
			fmt.Println("conn.Read err:", err, "cnt:", cnt)
			return
		}

		fmt.Println("服务器接收客户端发送过来的数据为：", string(buf[:cnt]), ",cnt:", cnt)
		//---------------业务逻辑处理 开始----------------------------
		//1、查询当前所有的用户 who
		//a.先判断接收的数据是不是who==》长度&&字符串
		userInput := string(buf[:cnt]) //这是用户输入的数据，最后一个是回车，我们去掉它
		//规则： rename|Duke
		//1.读取数据判断长度7，判断字符是rename
		//2.使用|进行分割，获取|后面的部分，作为名字
		//arry:=strings.Split(userInput,"")
		//name:=array[1]
		//3.更新用户名字newUser.name=duke
		if len(userInput) == 4 && userInput == "\\who" {
			//b.遍历allUsers这个map:key:=userid value: user本身。将id和name拼接成一个字符串，返回给客户端
			fmt.Println("用户即将查询所有用户信息！")
			//这个切片包含所有的用户信息
			var userInfos []string
			//[]string{userid:z3,username:z3,userid:l4,username:l4,userid:w5,username:l4}
			for _, user := range allUsers {
				userInfo := fmt.Sprintf("userid:%s,username:%s", user.id, user.name)
				userInfos = append(userInfos, userInfo)
			}

			//最终写到管道中， 一定是一个字符串
			r := strings.Join(userInfos, "\n") //连接数字切片，生成字符串
			// "ueridid:z3",username：z3“
			//
			//将数据返回给查询的客户端
			newUser.msg <- r

		} else if len(userInput) > 9 && userInput[:7] == "\\rename" {
			newUser.name = strings.Split(userInput, "|")[1]
			allUsers[newUser.id] = newUser //更新map中的user
			//4.通知客户端，更新成功
			newUser.msg <- "rename successfully!"

		} else {
			//如果用户输入的不是 命令，只是普通的聊天信息，那么只需要写到广播通道中即可，由其他的go程进行常规转发
			message <- userInput
		}

		restTimer <- true
		//---------------业务逻辑处理 结束-----------------------------
	}
}

// 向所有的用户广播消息，启动一个全局唯一go程 向管道读取数据
func broadcast() {
	fmt.Println("广播go程启动成功...")
	defer fmt.Println("broadcast程序退出")
	//for  循环的目的  不让该函数broadcast退出 可以一直读取数据
	for {

		fmt.Println("broadcast监听message中...")
		//1.从message中读取数据
		info := <-message
		fmt.Println("message 接收到消息:", info)
		//2.将数据写入到每一个用户的msg通道之中
		for _, user := range allUsers {
			//如果msg是非缓冲的，那么会在这里阻塞了，
			user.msg <- info
		}

	}
}

// 每个用户应该还有一个用来监听自己msg管道的go程，负责将数据返回给客户端
func wrtieBackToClient(user *User, conn net.Conn) {
	fmt.Println("1111111111111user:%s 的go程 正在监听自己的msg管道:\n", user.name)
	// msg 管道
	for data := range user.msg {
		fmt.Printf("user: %s 写回给客户端的数据为%s\n", user.name, data)
		//Write(b []byte) (n int, err error)
		_, _ = conn.Write([]byte(data))
	}
}

// 启动一个go程，负责监听退出信号；触发后，进行清零工作：delete map，close conn都在这里
func watch(user *User, conn net.Conn, isQuit, restTimer <-chan bool) {
	fmt.Println("22222222222222启动监听退出信号的go程...")
	defer fmt.Println("watch go程退出!")
	for {
		select {
		case <-isQuit:
			logoutInfo := fmt.Sprintf("%s exit already!\n", user.name)
			fmt.Println("删除当前用户:", user.name)
			delete(allUsers, user.id)
			message <- logoutInfo
			conn.Close()
			return
		case <-time.After(10 * time.Second):
			logoutInfo := fmt.Sprintf("%s timeout exit already!", user.name)
			fmt.Println("删除当前用户:", user.name)
			delete(allUsers, user.id)
			message <- logoutInfo
			conn.Close()
			return
		case <-restTimer:
			fmt.Println("链接%s 重置计数器！\n", user.name)

		}
	}
}
