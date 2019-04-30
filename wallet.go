package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"golang.org/x/crypto/ripemd160"
)

const version = byte(0x00) //定义版本号，一个字节
const addressChecksumLen = 4 //定义checksum长度为四个字节

//定义钱包， 包括一对私钥+公钥，以及转换成的钱包地址
type Wallet struct{
	PrivateKey  ecdsa.PrivateKey
	PublicKey  []byte
}

//创建钱包对象,返回指针
func NewWallet() *Wallet{
	private,public := newKeyPair()
	wallet := Wallet{private, public}
	return &wallet
}

//获取钱包地址，根据公钥计算出比特币地址。
func (w *Wallet) GetAddress() []byte  {

	//调用Ripemd160Hash返回160位的Pub Key hash
	ripemd160Hash := HashPubKey(w.PublicKey)

	//将version+Pub Key hash
	version_ripemd160Hash := append([]byte{version},ripemd160Hash...)

	//调用CheckSum方法返回前四个字节的checksum
	checkSumBytes := CheckSum(version_ripemd160Hash)
	//checkSumBytes := CheckSum(ripemd160Hash)  //这样写是错误的

	//将version+PubKeyhash+ checksum生成25个字节
	bytes := append(version_ripemd160Hash,checkSumBytes...)

	//将这25个字节进行base58编码并返回
	return Base58Encode(bytes)
}


//===============================================

//生成私钥和公钥
func newKeyPair() (ecdsa.PrivateKey,[]byte){

	//生成椭圆曲线,  secp256r1 曲线。 比特币当中的曲线是secp256k1
	curve :=elliptic.P256()

	private,err :=ecdsa.GenerateKey(curve,rand.Reader)

	if err !=nil{

		fmt.Println("error")
	}

	//拼接x和y坐标，就是公钥
	pubkey :=append(private.PublicKey.X.Bytes(),private.PublicKey.Y.Bytes()...)
	return *private,pubkey

}

//取前4个字节
func CheckSum(payload []byte) []byte {
	//这里传入的payload其实是version+Pub Key hash，对其进行两次256运算
	hash1 := sha256.Sum256(payload)

	hash2 := sha256.Sum256(hash1[:])

	return hash2[:addressChecksumLen] //返回前四个字节，为CheckSum值
}

//根据公钥计算出对应的公钥hash值，先SHA256后ripemd160，得到20字节的hash值
func HashPubKey(publicKey []byte) []byte {
	//将传入的公钥进行256运算，返回256位hash值
	hash256 := sha256.Sum256(publicKey)

	//将上面的256位hash值进行160运算，返回160位的hash值
	ripemd160 := ripemd160.New()
	_,err := ripemd160.Write(hash256[:])
	if err != nil{
		fmt.Print("error=%s\n",err)
	}

	publicRIPEMD160 := ripemd160.Sum(nil)
	return publicRIPEMD160
}


//判断地址是否有效
func IsValidAdress(adress []byte) bool {
	//将地址进行base58反编码，生成的其实是version+Pub Key hash+ checksum这25个字节
	version_public_checksumBytes := Base58Decode(adress)

	//[25-4:],就是21个字节往后的数（22,23,24,25一共4个字节）
	checkSumBytes := version_public_checksumBytes[len(version_public_checksumBytes) - addressChecksumLen:]
	//[:25-4],就是前21个字节（1～21,一共21个字节）
	version_ripemd160 := version_public_checksumBytes[:len(version_public_checksumBytes) - addressChecksumLen]
	//取version+public+checksum的字节数组的前21个字节进行两次256哈希运算，取结果值的前4个字节
	checkBytes := CheckSum(version_ripemd160)
	//将checksum比较，如果一致则说明地址有效，返回true
	if bytes.Compare(checkSumBytes,checkBytes) == 0 {
		return true
	}

	return false
}

//根据字符串形式的地址--->公钥hash
func GetPubKeyHash(address string) []byte{
	decodeAddress := Base58Decode([]byte(address))
	pubkeyhash := decodeAddress[1:len(decodeAddress)-addressChecksumLen]
	return pubkeyhash
}