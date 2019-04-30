package main

import (
	"bytes"
	"math/big"
)

//切片存储base58字母
var b58Alphabet = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")


//把普通的字节数组转换成base58编码的字节数组
func Base58Encode(input []byte) []byte{
	//定义一个字节切片，返回值
	var result []byte

	//把字节数组input转化为了大整数big.Int
	x:= big.NewInt(0).SetBytes(input)

	//长度58的大整数
	base := big.NewInt(int64(len(b58Alphabet)))
	//0的大整数
	zero := big.NewInt(0)
	//大整数的指针
	mod := &big.Int{}

	//循环，不停地对x取余数,大小为58
	for x.Cmp(zero) != 0 {
		x.DivMod(x,base,mod)  // 对x取余数

		//将余数添加到数组当中
		result =  append(result, b58Alphabet[mod.Int64()])
	}


	//反转字节数组
	ReverseBytes(result)

	//如果这个字节数组的前面为字节0，会把它替换为1.
	for _,b:=range input{

		if b ==0x00{
			result =  append([]byte{b58Alphabet[0]},result...)
		}else{
			break
		}
	}


	return result

}



func Base58Decode(input []byte) []byte{
	result :=  big.NewInt(0)
	zeroBytes :=0
	for _,b :=range input{
		if b=='1'{
			zeroBytes++
		}else{
			break
		}
	}
	payload:= input[zeroBytes:]

	//这个乘58+余数的方法太巧妙了
	for _,b := range payload{
		charIndex := bytes.IndexByte(b58Alphabet,b)  //反推出余数

		result.Mul(result,big.NewInt(58))   //之前的结果乘以58

		result.Add(result,big.NewInt(int64(charIndex)))  //加上这个余数

	}

	decoded :=result.Bytes()
	decoded =  append(bytes.Repeat([]byte{0x00},zeroBytes),decoded...)
	return decoded
}
