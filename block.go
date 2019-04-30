package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"math"
	"time"
)

//确定nonce随机数的最大值，用无符号整数防止溢出
var maxnonce uint32 = math.MaxUint32

type Block struct{
	Hash          []byte   //本区块内容的hash值
	Version uint32         //版本号
	PrevBlockHash []byte   //前一区块的hash值
	Merkleroot    []byte   //默克尔根的hash值
	Time  uint32           //时间戳
	Bits  uint32           //网络计算难度
	Nonce uint32           //随机数
	Transations   []*Transation  // 切片中存储的是交易指针
	Height int32  //区块高度
}

////区块数据序列化
//func (block *Block) serialize() []byte{
//	result := bytes.Join(
//		[][]byte{
//			IntToHex(block.Version),
//			block.PrevBlockHash,
//			block.Merkleroot,
//			IntToHex(block.Time),
//			IntToHex(block.Bits),
//			IntToHex(block.Nonce)},
//		[]byte{},
//	)
//	return result
//}

//区块数据序列化
func (block *Block) Serialize() []byte{
	var encoded bytes.Buffer
	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(block)

	if err !=nil{
		log.Panic(err)
	}
	return encoded.Bytes()
}

//区块数据反序列化
func DeserializeBlock(d []byte) *Block{
	var block Block
	decode := gob.NewDecoder(bytes.NewReader(d))
	err := decode.Decode(&block)

	if err !=nil{
		log.Panic(err)
	}
	return &block
}

//格式化打印交易完整信息
func (block *Block) ToString() {
	fmt.Printf("block:#%d\n",block.Height)
	fmt.Printf("	Hash:%x\n",block.Hash)
	fmt.Printf("	Version:%d\n",block.Version)
	fmt.Printf("	PrevBlockHash:%x\n",block.PrevBlockHash)
	fmt.Printf("	Merkleroot:%x\n",block.Merkleroot)
	fmt.Printf("	Time:%d\n",block.Time)
	fmt.Printf("	Bites:%d\n",block.Bits)
	fmt.Printf("	Nonce:%x\n",block.Nonce)
	fmt.Printf("	number of Transations:%d\n",len(block.Transations))

}

////计算挖矿困难度difficulty
//func CaculateDifficulty(target []byte) string{
//	geniusHash :="00000000ffff0000000000000000000000000000000000000000000000000000"
//	var biGeniusHash  big.Int
//	var biTargetHash  big.Int
//	biGeniusHash.SetString(geniusHash,16)
//	biTargetHash.SetBytes(target)
//
//	difficulty :=big.NewFloat(0)
//	difficulty.Quo()
//	difficulty.Div(&biGeniusHash,&biTargetHash)
//	return fmt.Sprintf("%f",difficulty)
//}

//解析bits网络难度参数的含义，计算出目标hash值,以bits=453281356为例，16进制表达是0x1B04864C
//bits计算结果    000000000004864c000000000000000000000000000000000000000000000000
//block hash     00000000000090ff2791fe41d80509af6ffbd6c5b10294e29cdf1b603acab92c
//Previous Block 0000000000045b02ab29280b9df7e9513fa6fe274f0d7fd7ecf95c6d708ceb29
//上面的hash值表明，生成的hash值必须小于bits计算结果。 换句话说，只要小于bits计算结果就满足要求。
func  CaculateTargetValue(bits []byte) []byte{
	//第一个字节表示指数
	exp := bits[0]
	fmt.Printf("exp=%d\n",exp)
	fmt.Printf("type=%T\n",exp)

	//计算后面3个字节
	coefficient := bits[1:]

	//拼接出目标hash值格式
	result := append(bytes.Repeat([]byte{0x00}, 32-int(exp)),coefficient...)
	result = append(result,bytes.Repeat([]byte{0x00}, 32-len(result))...)

	return result
}

//为区块添加默克尔根hash值，输入参数是交易切片
func (block *Block) createMerkleTreeRoot(transations []*Transation){
	var tranHashs [][]byte

	for _,tx := range transations{
		tranHashs = append(tranHashs, tx.Hash())
	}
	mTree := NewMerkleTree(tranHashs)
	block.Merkleroot = mTree.RootNode.Data
}

//创建普通区块，要求输入前一区块的hash值
func NewBlock(transations []*Transation,prevBlockHash []byte,height int32) * Block{
	//初始化区块
	block :=&Block{
		[]byte{},
		1,
		prevBlockHash,
		[]byte{},
		uint32(time.Now().Unix()),
		453281356,
		0,
		transations,
		height,
	}

	pow := NewProofOfWork(block)
	nonce, hash := pow.Run()
	fmt.Printf("block.go, NewBlock(): hash=%x\n",hash)
	block.Nonce = nonce
	block.Hash  = hash

	return block
}

//建立创世区块，输入参数是交易
func NewGensisBlock(transations []*Transation) * Block{
	//初始化区块
	block :=&Block{
		[]byte{},
		1,
		[]byte{},
		[]byte{},
		uint32(time.Now().Unix()),
		453281356,    //453281356
		0,
		transations,
		0,
	}

	pow := NewProofOfWork(block)
	nonce, hash := pow.Run()
	//fmt.Printf("NewGensisBlock()： hash=%x\n",hash)
	block.Nonce = nonce
	block.Hash  = hash

	//block.ToString()
	return block
}





