package main

import (
	"bytes"
	"crypto/sha256"
	"math/big"
)


//计算工作量证明，只要提供区块和难度就能自动计算出来这个nonce
type ProofOfWork struct{
	block * Block
	target * big.Int  //这就是bits对应大整数，这里直接硬代码赋值
}

const targetBits = 16  //表示后面有256-16=240位为0，前面就是0x0001啦

func NewProofOfWork(b * Block) *ProofOfWork{
	target :=big.NewInt(1)
	target.Lsh(target,uint(256-targetBits))
	pow :=&ProofOfWork{b,target}
	return pow
}

func (pow *ProofOfWork) PrepareData(nonce uint32) []byte{
	data :=bytes.Join(
		[][]byte{
			IntToHex(pow.block.Version),
			pow.block.PrevBlockHash,
			pow.block.Merkleroot,
			IntToHex(pow.block.Time),
			IntToHex(pow.block.Bits),
			IntToHex(uint32(nonce)),   },
		[]byte{},
	)

	return data
}

//开始挖矿计算
func (pow * ProofOfWork) Run() (uint32,[]byte){
	var nonce uint32 = 0
	var secondHash [32]byte
	var currentHash big.Int

	for nonce < maxnonce {
		//序列化
		data :=pow.PrepareData(nonce)
		//double hash
		firstHash  := sha256.Sum256(data)
		secondHash  = sha256.Sum256(firstHash[:])
		currentHash.SetBytes(secondHash[:])
		//fmt.Printf("nonce=%d,  currenthash=%x\n",nonce, secondHash)

		//比较
		if currentHash.Cmp(pow.target) ==-1{
			break
		}else{
			nonce++
		}
	}

	return nonce,secondHash[:]
}

//验证nonce是否正确
func (pow * ProofOfWork) Validate() bool{
	var hashInt  big.Int

	data := pow.PrepareData(pow.block.Nonce)

	firstHash  := sha256.Sum256(data)
	secondHash := sha256.Sum256(firstHash[:])
	hashInt.SetBytes(secondHash[:])

	isValid := hashInt.Cmp(pow.target)==-1
	return isValid
}