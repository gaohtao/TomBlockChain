package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"
)

//测试函数
func main(){
	//TestCreateMerkleTreeRoot()
	//TestPow()
	//TestNewSerialize()
	//NewGensisBlock()
	//TestBoltDB()
	//wallet  := NewWallet()
	////打印私钥  曲线上的x点
	//fmt.Printf("私钥：%x\n",wallet.PrivateKey.D.Bytes())
	//
	////打印公钥， 曲线上的x点和y点
	//fmt.Printf("公钥：%x\n",wallet.PublicKey)
	//
	////打印钱包地址
	//fmt.Printf("地址：%x\n",wallet.GetAddress())
	//
	////验证地址
	//add,_ := hex.DecodeString("31413165624758754a45714b575231317150544234447a5773745a4a326652507279")
	//fmt.Printf("验证地址结果：%d\n",IsValidAdress(add))

	TestCliArgs()

}


func  main2(){

	//示例BTC区块:
	// https://www.blockchain.com/btc/block/00000000000090ff2791fe41d80509af6ffbd6c5b10294e29cdf1b603acab92c
	//00000000000090ff2791fe41d80509af6ffbd6c5b10294e29cdf1b603acab92c
	//00000000000090ff2791fe41d80509af6ffbd6c5b10294e29cdf1b603acab92c

	//版本号
	var version  uint32 = 1
	fmt.Printf("%x\n", IntToHex(version))

	//前一个区块的hash
	prev,_ :=hex.DecodeString("0000000000045b02ab29280b9df7e9513fa6fe274f0d7fd7ecf95c6d708ceb29")
	ReverseBytes(prev)
	fmt.Printf("%x\n",prev)

	//默克尔根hash
	merkleRoot,_ :=hex.DecodeString("c66ee6e01c2332b92e71e17b6c6c3d4e926f6330a06acbb0e203bf7d97d12249")
	ReverseBytes(merkleRoot)
	fmt.Printf("%x\n",merkleRoot)

	//时间戳转化成秒数，
	var timeStamp string = "2010-12-22 12:49:27"
	//从字符串转为时间戳，第一个参数是格式,必须是2006-01-02 15:04:05字符串，不然计算出来的秒数错误，第二个是要转换的时间字符串
	tm, _ := time.Parse("2006-01-02 15:04:05", timeStamp)
	var time uint32 = uint32(tm.Unix())
	fmt.Printf("time=%d\n",time)

	//网络的难度
	var bits uint32 = 453281356
	fmt.Printf("bits=%d\n",bits)

	//随机数  计算出来的是3806873897
	var nonce uint32 = 0
	fmt.Printf("nonce=%d\n",nonce)

	//初始化区块
	block :=&Block{
		[]byte{},
		1,
		prev,
		merkleRoot,
		time,
		bits,
		nonce,
		[]*Transation{},
		0,

	}

	//目标hash
	targetHash :=CaculateTargetValue(IntToHex2(block.Bits))
	fmt.Printf("targetHash=%x\n",targetHash)
	var target big.Int
	target.SetBytes(targetHash)  //目标hash转换为大整数

	var currenthash  big.Int    //当前区块计算出来的hash值
	block.Nonce = 3806873890

	//开始挖矿计算过程，反复计算当前hash值，直到小于目标hash为止
	for block.Nonce < maxnonce{

		//区块序列化
		data := block.Serialize()
		//两次hash256
		firsthash  := sha256.Sum256(data)
		secondhash := sha256.Sum256(firsthash[:])
		ReverseBytes(secondhash[:])
		fmt.Printf("nonce=%d,  currenthash=%x\n",block.Nonce, secondhash)

		//判断是否达到要求
		currenthash.SetBytes(secondhash[:])
		if currenthash.Cmp(&target) < 0 {
			break;
		}else{
			block.Nonce++
		}
	}

	fmt.Printf("区块hash计算结束，挖矿成功拉。。。\n")

}
