package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"net"
)

//定义版本信息， 用于网络节点间的版本查询
type Version struct{
	Version     int32    //版本信息发送方的当前版本
	BestHeight  int32    //版本信息发送方的区块高度
	AddrFrom    string   //命令发送方地址，用于对方应答回来
}

//请求指定高度范围的区块Hash值列表
type GetBlocks struct {
	AddrFrom    string    //命令发送方地址，用于对方应答回来
	LowHeight   int32     //区块高度--低
	HighHeight  int32     //区块高度--高
}

//发送inv命令数据专用结构体
type Inv struct {
	AddrFrom string    //命令发送方地址，用于对方应答回来
	Type string
	Items [][]byte
}

//发送区块Hash数据专用结构体
type GetData struct {
	AddrFrom  string   //命令发送方地址，用于对方应答回来
	Type      string
	ID        []byte
}

//发送区块数据专用结构体
type BlockCMDData struct {
	AddrFrom  string   //命令发送方地址，用于对方应答回来
	Block []byte
}

const nodeversion = 0x00
const cmdLength   = 12    //命令字固定长度10字节，方便接收方解析
var nodeAddress string    //程序使用的本机IP+端口

//定义公共节点地址，在一台电脑上模拟多个节点时，就用不同端口代表不同的网络节点
var  knownNodes = []string{"localhost:3000"}

var blockInTransit [][]byte  //这个保存的是外部公共节点的全部区块Hash值，用于不断的发出下载区块命令的。


//-----------------------------------------------


//启动节点的服务器程序, 参数nodeID就是端口号
func StartServer(nodeID, minerAddrerss string, bc *BlockChain){
	nodeAddress = fmt.Sprintf("localhost:%s",nodeID)

	listen,err := net.Listen("tcp",nodeAddress)
	checkErr(err)
	defer listen.Close()

	//如果本程序监听的IP:port不是公共节点，就向公共节点发送自己的版本信息
	if nodeAddress != knownNodes[0]{
		sendVersion(knownNodes[0],bc)
	}

	for{
		conn, err := listen.Accept()
		if err != nil{
			fmt.Println(err)
			continue
		}
		go HandleConnection(conn,bc)  //没收到连接就启动一个协程进行处理
	}
}

//处理其他节点的请求命令
func HandleConnection(conn net.Conn, bc *BlockChain) {
	request,err := ioutil.ReadAll(conn) //读取全部数据
	checkErr(err)

	//解析命令
	cmd := bytesToCmd(request[:cmdLength])
	fmt.Println("HandleConnection(): cmd=",cmd)
	switch cmd{
	case "version":
		handleVersion(request,bc)
	case "getblocks":
		handleGetBlocks(request,bc)
	case "inv":
		handleInv(request,bc)
	case "getdata":
		handleGetData(request,bc)
	case "blockdata":
		handleBlockData(request,bc)
	}
}

//处理收到的version版本命令
func handleVersion(request []byte, bc *BlockChain) {
	var buff bytes.Buffer
	var payload Version

	buff.Write(request[cmdLength:]) //提取命令数据的内容
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	checkErr(err)
	fmt.Printf("handleVersion(), receive ‘version’ \n")
	payload.toString() //显示收到的Version数据

	myBestHeight := bc.GetBestHeight()
	foreignerBestHeight := payload.BestHeight

	fmt.Printf("myBestHeight=%d, foreignerBestHeight=%d\n",myBestHeight,foreignerBestHeight)
	if myBestHeight < foreignerBestHeight{
		//说明本节点的区块高度小，需要从外部节点获取新的区块
		sendGetBlocks(payload.AddrFrom, myBestHeight+1,foreignerBestHeight)

	}else{
		//说明本节点的区块高度大，把自己的版本信息发给外部节点
		fmt.Printf("发送Version to:%s\n", payload.AddrFrom)
		sendVersion(payload.AddrFrom,bc)
	}

	//无论区块高度大小，都说明这个外部节点是一个可用的节点，添加到公共节点列表中
	if !nodeIsKnow(payload.AddrFrom){
		knownNodes = append(knownNodes, payload.AddrFrom)
	}

}

//处理收到的getblocks命令， 发送inv命令，携带全部区块hash值
func handleGetBlocks(request []byte, bc *BlockChain) {
	var buff bytes.Buffer
	var payload GetBlocks

	buff.Write(request[cmdLength:]) //提取命令数据的内容
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	checkErr(err)
	fmt.Printf("handleGetBlocks(), receive ‘getblocks’ \n")
	fmt.Printf("     low=%d, high=%d\n",payload.LowHeight,payload.HighHeight)


	blockhash:=bc.GetBlockHashScope(payload.LowHeight,payload.HighHeight)
	sendInv(payload.AddrFrom,"block",blockhash)
}

//处理收到的inv命令， 发送getdata命令，携带指定的区块hash，表示要下载这个区块的数据
func handleInv(request []byte, bc *BlockChain) {
	var buff bytes.Buffer
	var payload Inv

	buff.Write(request[cmdLength:]) //提取命令数据的内容
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	checkErr(err)
	fmt.Printf("handleInv(), receive inventory %d, %s \n",len(payload.Items),payload.Type)

	if payload.Type == "block"{
		blockInTransit = payload.Items
		blockHash := payload.Items[0]  //这是最新区块的hash
		sendGetData(payload.AddrFrom,"block",blockHash)   //请求下载这个最新区块


		//发出请求最新区块命令后， 这个最新的blockHash就没用了，可以从blockInTransit删除了。
		//删除的方法是保存剩余的hash值，然后替换掉blockInTransit
		newInTransit := [][]byte{}
		for _,b:= range blockInTransit{
			if bytes.Compare(b,blockHash) !=0{
				newInTransit = append(newInTransit,b)
			}
		}
		blockInTransit = newInTransit   //替换
	}
}

//处理收到的getdata命令， 发出命令，携带这个区块的具体内容数据
func handleGetData(request []byte, bc *BlockChain) {
	var buff bytes.Buffer
	var payload GetData

	buff.Write(request[cmdLength:]) //提取命令数据的内容
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	checkErr(err)
	fmt.Printf("handleGetData(), receive ‘getdata’ \n")

	if payload.Type == "block"{
		block,err := bc.GetBlock(payload.ID)
		checkErr(err)
		sendBlock(payload.AddrFrom,&block)   //这里才真正的发送这个区块数据
	}
}

//处理收到的blockdata版本命令
func handleBlockData(request []byte, bc *BlockChain) {
	var buff bytes.Buffer
	var payload BlockCMDData

	buff.Write(request[cmdLength:]) //提取命令数据的内容
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	checkErr(err)

	//保存接收到的block区块数据
	blockdata := payload.Block
	block := DeserializeBlock(blockdata)
	fmt.Printf("handleBlockData(): receive a new Block, hash=%x\n",block.Hash)
	bc.AddBlock(block)

	if len(blockInTransit)>0{
		blockHash := blockInTransit[0]
		sendGetData(payload.AddrFrom,"block",blockHash)

		blockInTransit = blockInTransit[1:]  //更新hash列表
	}else{
		//全部区块都更新完毕，准备更新UTXO
		set := UTXOSet{bc}
		set.Reindex()
	}

}

//-----------------------------------------------------


//发送区块具体内容
func sendBlock(addr string, block *Block) {
	data := BlockCMDData{nodeAddress,block.Serialize()}
	payload := gobEncode(data)
	request := append(cmdToBytes("blockdata"),payload...)
	sendData(addr,request)
}


//发送获取区块数据命令getdata
func sendGetData(addr string, kind string, id []byte) {
	payload := gobEncode(GetData{nodeAddress,kind,id})
	request := append(cmdToBytes("getdata"),payload...)
	sendData(addr,request)
}

//发送本节点的全部区块Hash值的命令inv
func sendInv(addr string, kind string, items [][]byte) {
	inventory := Inv{nodeAddress,kind,items}
	payload := gobEncode(inventory)
	request := append(cmdToBytes("inv"),payload...)
	sendData(addr,request)
}

//发送下载区块Hash列表命令getblocks
func sendGetBlocks(addr string, low int32, high int32) {
	fmt.Printf("sendGetBlocks(): nodeAddress=%s\n",nodeAddress)
	payload := gobEncode(GetBlocks{nodeAddress, low, high})
	request := append(cmdToBytes("getblocks"),payload...)
	sendData(addr,request)
}

//发送自己的版本信息
func sendVersion(addr string, bc *BlockChain) {
	bestHeight := bc.GetBestHeight()

	payload := gobEncode(Version{nodeversion,bestHeight,nodeAddress})
	request := append(cmdToBytes("version"),payload...)
	sendData(addr,request)
}

//发送命令请求
func sendData(addr string, data []byte) {
	con,err := net.Dial("tcp",addr)
	if err != nil{
		//连接失败后要把这个地址从公共节点中删除
		fmt.Printf("%s is not available\n", addr)
		var updateNodes []string
		for _,node := range knownNodes{
			if node != addr{   //addr这个地址连接不通了，故意丢弃addr，只保留其它地址
				updateNodes = append(updateNodes,node)
			}
		}
		knownNodes = updateNodes   //更新公共节点
	}
	defer con.Close()

	_,err = io.Copy(con,bytes.NewReader(data))  //发送命令
	checkErr(err)
}

//命令字转成字节切片
func cmdToBytes(command string) []byte{
	var bytes [cmdLength]byte
	for i,c:= range command{
		bytes[i]=byte(c)
	}
	return bytes[:]
}

//解析出命令字
func bytesToCmd(bytes []byte) string{
	var cmd []byte
	for _,b:=range bytes{
		if b!=0x00{
			cmd = append(cmd,b)
		}
	}
	return fmt.Sprintf("%s",cmd)
}

//序列化任意类型数据,注意用的是接口类型
func gobEncode(data interface{}) []byte{
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(data)
	checkErr(err)

	return buff.Bytes()
}

//检查指定地址是否在公共节点列表中
func nodeIsKnow(addr string) bool {
	for  _,node := range knownNodes{
		if node == addr{
			return true
		}
	}
	return false
}

//对象输出字符串形式
func (ver *Version) toString(){
	fmt.Printf("Version:%x\n",ver.Version)
	fmt.Printf("BestHeight:%d\n",ver.BestHeight)
	fmt.Printf("AddrFrom:%s\n",ver.AddrFrom)
}


