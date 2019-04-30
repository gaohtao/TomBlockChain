package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

type CLI struct{
	bc * BlockChain
}

//检查命令行参数个数
func (cli *CLI) validateArgs(){
	if len(os.Args) <=1 {
		println("命令行没有参数，请重新输入！")
		os.Exit(1)
	}
	//println(os.Args)
}

func (cli *CLI) printUsage(){
	fmt.Println("Usage:")
	fmt.Println("	addBlock: 增加区块")
	fmt.Println("	printChain:打印所有区块")
	fmt.Println("	getBalance -address Tom: 查询Tom的账户余额")
	fmt.Println("	send -from  Tom  -to Jerry -amount 20: Tom转账给Jerry 20")
	fmt.Println("	createWallet :创建一个钱包地址")
	fmt.Println("	listAddress :显示所有钱包地址")
	fmt.Println("	getBestHeight :显示区块高度")
	fmt.Println("	startNode -minner Tom: 启动节点，设置矿工钱包地址")

}

//运行命令行程序，解析参数，执行各个功能
func (cli *CLI) Run(){

	//获取系统环境变量
	nodeID := os.Getenv("NODE_ID")
	if nodeID==""{
		fmt.Printf("NODE_ID is not set， please set system ENV ： NODE_ID=3000")
		os.Exit(1)
	}

	cli.validateArgs()

	addBlockCmd  := flag.NewFlagSet("addBlock"  ,flag.ExitOnError)
	printChainCmd:= flag.NewFlagSet("printChain",flag.ExitOnError)
	getBalanceCmd:= flag.NewFlagSet("getBalance",flag.ExitOnError)
	getBalanceAddress := getBalanceCmd.String("address","","getBalance --address Tom")

	//转账命令参数解析
	sendCmd := flag.NewFlagSet("send",flag.ExitOnError)
	send_From   := sendCmd.String("from","","Source wallet address")
	send_To     := sendCmd.String("to","","Destination wallet address")
	send_Amount   := sendCmd.Int("amount",0,"Amount to send")

	//创建钱包，查看钱包地址
	createWalletCmd := flag.NewFlagSet("createWallet",flag.ExitOnError)
	listAddressCmd := flag.NewFlagSet("listAddress",flag.ExitOnError)

	getBestHeightCmd:= flag.NewFlagSet("getBestHeight",flag.ExitOnError)

	startNodeCmd:= flag.NewFlagSet("startNode",flag.ExitOnError)
	startNodeMinner := startNodeCmd.String("minner","","startNode --minner Tom")


	switch os.Args[1]{
	case "addBlock":
		err :=addBlockCmd.Parse(os.Args[2:])
		if err != nil{
			log.Panic(err)
		}
	case "printChain":
		err :=printChainCmd.Parse(os.Args[2:])
		if err != nil{
			log.Panic(err)
		}
	case "getBalance":
		err :=getBalanceCmd.Parse(os.Args[2:])
		if err != nil{
			log.Panic(err)
		}
	case "send":
		err :=sendCmd.Parse(os.Args[2:])
		if err != nil{
			log.Panic(err)
		}
	case "createWallet":
		err :=createWalletCmd.Parse(os.Args[2:])
		if err != nil{
			log.Panic(err)
		}
	case "listAddress":
		err :=listAddressCmd.Parse(os.Args[2:])
		if err != nil{
			log.Panic(err)
		}
	case "getBestHeight":
		err :=getBestHeightCmd.Parse(os.Args[2:])
		if err != nil{
			log.Panic(err)
		}
	case "startNode":
		err :=startNodeCmd.Parse(os.Args[2:])
		if err != nil{
			log.Panic(err)
		}

	default:
		cli.printUsage()
		os.Exit(1)
	}

	//addBlockCmd参数解析成功，该执行相关处理了
	if addBlockCmd.Parsed(){
		cli.addBlock()
	}
	if printChainCmd.Parsed(){
		cli.printChain()
	}
	if getBalanceCmd.Parsed(){
		//检查地址参数是否正确，如果为空表示错误，强制停止运行
		if *getBalanceAddress == ""{
			os.Exit(1)
		}
		account := cli.GetBalance(*getBalanceAddress)
		fmt.Printf("钱包地址:%s， 余额:%d\n",*getBalanceAddress, account)
	}
	if sendCmd.Parsed(){
		//检查from/to/amount参数是否正确，如果为空表示错误，强制停止运行
		if *send_From=="" || *send_To=="" || *send_Amount<=0 {
			os.Exit(1)
		}
		cli.send(*send_From, *send_To, *send_Amount)

		fmt.Printf("转账完成。。。\n")
	}

	if createWalletCmd.Parsed(){
		cli.createWallet()
	}
	if listAddressCmd.Parsed(){
		cli.listAddress()
	}

	if getBestHeightCmd.Parsed(){
		cli.getBestHeight()
	}

	if startNodeCmd.Parsed(){
		nodeID := os.Getenv("NODE_ID")
		if nodeID==""{
			startNodeCmd.Usage()
			os.Exit(1)
		}
		//检查矿工钱包地址参数是否正确，如果为空表示错误，强制停止运行
		if *startNodeMinner == ""{
			fmt.Printf("Error: minner address is null! \n")
			os.Exit(1)
		}
		cli.startNode(nodeID, *startNodeMinner)
	}
}

//根据命令行参数添加区块
func (cli *CLI) addBlock(){
	cli.bc.MineBlock([]*Transation{})   //先添加一个空的交易列表
}

//打印链上的区块信息
func (cli *CLI) printChain(){
	cli.bc.PrintBlockChain()
}

//计算指定账户的余额,不再是遍历链上的交易，而是从数据桶中找出指定用户的余额。
func (cli *CLI) GetBalance(address string) int{
	balance :=0
	pubkeyhash := GetPubKeyHash(address)

	//UTXOs := cli.bc.FindUTXO2(pubkeyhash)
	set := UTXOSet{cli.bc}
	UTXOs := set.FindUTXObyPubkeyHash(pubkeyhash)

	for _,out :=range UTXOs{
		balance += out.Value
	}

	return balance
}

//转账操作，先生成一笔新交易， 存入挖矿所得的新区块中
func (cli *CLI) send(from, to string, amount int){
	tx := NewUTXOTransation(from,to,amount,cli.bc)  //会进行交易签名
	newblock := cli.bc.MineBlock([]*Transation{tx}) //会验证交易签名

	//把新区块的交易数据更新到数据桶中
	set := UTXOSet{cli.bc}
	set.update(newblock)

	fmt.Printf("send success!\n")
}

// 新建钱包
func (cli *CLI) createWallet(){
	wallets,_ :=NewWallets()
	add := wallets.CreateWallet()
	fmt.Printf("your address:%s\n",add)

	//钱包集重新写入文件
	wallets.SaveToFile2()
}

// 查看钱包集中所有的地址
func (cli *CLI) listAddress(){
	wallets,err:=NewWallets()
	if err!=nil{
		log.Panic(err)
	}
	alladdress := wallets.GetAllAddress()
	for _,add := range alladdress{
		fmt.Println(add)
	}
}

func (cli *CLI) getBestHeight() {
	height := cli.bc.GetBestHeight()

	fmt.Printf("last height= %d\n",height)
}

//启动节点
func (cli *CLI) startNode(nodeID string, minnerAddress string) {
	fmt.Printf("Starting node:  port=%s\n",nodeID)

	if len(minnerAddress)>0{
		if IsValidAdress([]byte(minnerAddress)){
			fmt.Println("minner address is ok ",minnerAddress)
		}else{
			log.Panic("Error: minner address")
		}
	}

	StartServer(nodeID,minnerAddress, cli.bc)
}




