package main

import (
	"bytes"
	"crypto/elliptic"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

const walletFile = "wallet.dat"

//定义钱包集，里面通过map存储了多个钱包
type Wallets struct{
	Store map[string]*Wallet  //这里的Store变量必须首字母大写，这样才能在序列化时被输出。
}


//读取文件建立钱包集
func NewWallets() (*Wallets,error){
	wallets := Wallets{}
	wallets.Store = make(map[string]*Wallet)

	//改造: 如果发现钱包文件存在就读取文件内容，恢复钱包地址；不存在就创建文件，并新建地址。
	_,err := os.Stat(walletFile)
	if os.IsNotExist(err){  //检查文件是否存在
		fmt.Printf("钱包文件（%s）不存在，创建钱包文件...\n",walletFile)
		wallets.CreateWallet()
		wallets.SaveToFile2()  //钱包集重新写入文件
		err=nil   //必须清空错误信息
	}else{
		err = wallets.LoadFromFile()
	}


	return &wallets,err
}

//创建钱包，返回字符串形式的钱包地址
func (ws *Wallets) CreateWallet() string{
	wallet := NewWallet()
	address := fmt.Sprintf("%s",wallet.GetAddress())
	ws.Store[address] = wallet
	return address
}

//根据地址获取钱包
func (ws *Wallets) GetWallet(address string) Wallet{
	//注意这里可能会犯错误。 如果在钱包集ws中找不到指定的钱包地址，就会出现空指针ws.Store[address]是nil，接着*ws.Store[address]就会出错了。
	//容易出现矿工的钱包地址没在钱包集中情况。
	//fmt.Printf("ws.Store[address]=",ws.Store[address])
	//fmt.Println("")
	return *ws.Store[address]
}

//获取钱包集中的所有地址,返回字符串数组
func (ws *Wallets) GetAllAddress() []string{
	var alladdress []string
	for address,_ := range ws.Store{
		alladdress = append(alladdress,address)
	}
	return alladdress
}

// Encode via Gob to file
func (ws *Wallets) SaveToFile() {
	file, err := os.Create(walletFile)
	if err != nil {
		log.Panic(err)
	}

	gob.Register(elliptic.P256())  //注册钱包地址中用的椭圆曲线
	encoder := gob.NewEncoder(file)
	err = encoder.Encode(ws)
	if err != nil {
		log.Panic(err)
	}
	file.Close()
}

//把ws序列化后写入文件保存
func (ws *Wallets) SaveToFile2() {
	var content bytes.Buffer

	gob.Register(elliptic.P256())  //注册钱包地址中用的椭圆曲线
	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(ws)
	if err != nil {
		log.Panic(err)
	}
	err = ioutil.WriteFile(walletFile,content.Bytes(),0777)
	if err != nil {
		log.Panic(err)
	}
}

//读取文件内容，反序列化成钱包集, 要求这个文件必须存在
func (ws *Wallets) LoadFromFile() error{
	////检查文件是否存在
	//_,err := os.Stat(walletFile)
	//if os.IsNotExist(err){
	//	log.Panic(err)
	//	return err
	//}

	fileContent,err := ioutil.ReadFile(walletFile)
	if err !=nil{
		log.Panic(err)
		return err
	}

	var wallets Wallets  //接受反序列化的临时变量
	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)
	if err !=nil{
		log.Panic(err)
		return err
	}

	ws.Store = wallets.Store  //把当前对象的store替换掉
	return nil
}


