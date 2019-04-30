package main

import (
	"bytes"
	"encoding/binary"
	"log"
)

//获取两数中的最小值
func min(a int, b int) int{
	if a<b{
		return a
	}
	return b
}

//将类型转化位字节数组，注意输出的字节数组是16进制数的小端顺序
func  IntToHex(num uint32) []byte{
	buff :=new(bytes.Buffer)
	//要求用小端模式
	err := binary.Write(buff, binary.LittleEndian, num)
	if err != nil {
		log.Panic(err)
	}
	return buff.Bytes()
}
//将类型转化位字节数组，注意输出的字节数组是16进制数的大端顺序
func  IntToHex2(num uint32) []byte{
	buff :=new(bytes.Buffer)
	//要求用小端模式
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		log.Panic(err)
	}
	return buff.Bytes()
}

//反转字节数组
func ReverseBytes(data []byte){
	for i,j :=0,len(data) - 1;i<j;i,j = i+1,j - 1{
		data[i],data[j] = data[j],data[i]
	}
}

//简化的错误检查方法
func checkErr(err error) {
	if err != nil {
		log.Panic(err)
	}
}
