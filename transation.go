package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"
)

const subsidy = 100   //初始挖矿奖励金额

//定义交易结构体
type Transation struct{
	ID   []byte        //交易的哈希值
	Vin  []TXInput     //交易输入
	Vout []TXOutput    //交易输出
}

//定义输入交易结构体
type TXInput struct{
	TXid  []byte        //交易的哈希值，指向被花费的UTXO所在交易的哈希
	Voutindex  int      //输出索引
	Signature  []byte   //解锁脚本
	Pubkey     []byte   //公钥
}

//定义输出交易结构体
type TXOutput struct{
	Value int           //总量，用聪表示的比特币值
	PubkeyHash  []byte  //公钥的hash
}

//定义输出交易结构体切片
type TXOutputs struct{
	Outputs  []TXOutput
}

//序列化
func (outs TXOutputs)Serialize() []byte{
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(outs)
	checkErr(err)
	return buf.Bytes()
}

//反序列化
func Deserialize(data []byte) TXOutputs{
	var outs TXOutputs
	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&outs)
	checkErr(err)
	return outs
}

//交易输出的上锁，这个公钥的hash值就对应着一个比特币地址，也就是钱包地址
func (out *TXOutput) Lock(address []byte){
	decodeAddress  := Base58Decode(address)
	pubkeyhash := decodeAddress[1:len(decodeAddress)-addressChecksumLen]
	out.PubkeyHash = pubkeyhash
}

//格式化打印交易完整信息
func (tx Transation) ToString()  string{
	var lines []string
	lines = append(lines,fmt.Sprintf("--- Transaction %x:",tx.ID))

	for i,input :=range tx.Vin{
		lines = append(lines, fmt.Sprintf("   Input: %d",i))
		lines = append(lines, fmt.Sprintf("       TXID:  %x",input.TXid))
		lines = append(lines, fmt.Sprintf("       Out:   %d",input.Voutindex))
		lines = append(lines, fmt.Sprintf("       Signature: %x",input.Signature))
	}

	for i,output :=range tx.Vout{
		lines = append(lines, fmt.Sprintf("   Output: %d",i))
		lines = append(lines, fmt.Sprintf("       Value:  %d",output.Value))
		lines = append(lines, fmt.Sprintf("       Sctrpt: %x",output.PubkeyHash))
	}

	return strings.Join(lines,"\n")
}


//序列化
func (tx *Transation) Serialize() []byte{
	var  encoded bytes.Buffer
	enc :=gob.NewEncoder(&encoded)

	err := enc.Encode(tx)
	if err != nil{
		log.Panic(err)
	}
	return encoded.Bytes()
}

//计算交易的hash值
func (tx *Transation) Hash() []byte {
	 txcopy := *tx
	 txcopy.ID = []byte{}

	 hash := sha256.Sum256(txcopy.Serialize())
	 return hash[:]
}

//根据金额与地址新建一个输出
func NewTXOutput(value int ,address string) *TXOutput{
	txo := TXOutput{value,nil}
	//txo.PubkeyHash = []byte(address)
	txo.Lock([]byte(address)) //设置公钥hash
	return &txo
}

//第一笔coinbase交易
func NewCoinbaseTX(to,data string) *Transation{
	txin  := TXInput{[]byte{}, -1, nil, []byte(data)}
	txout := NewTXOutput(subsidy, to)

	tx := Transation{nil,[]TXInput{txin},[]TXOutput{*txout}}
	tx.ID = tx.Hash()    // 交易的ID就是hash值

	return &tx
}

//交易输出中检验锁定脚本
func (out *TXOutput) CanBeUnlockedWith(pubkeyhash []byte) bool{
	return bytes.Compare(out.PubkeyHash, pubkeyhash)==0
}

//交易输入中检验锁定脚本
func (in *TXInput) CanBeUnlockedWith(pubkeyhash []byte) bool{

	lockinghash := HashPubKey(in.Pubkey)
	return bytes.Compare(lockinghash, pubkeyhash)==0
}

//检查交易是否是coninbase挖矿交易
func (tx Transation) isCoinBase() bool{
	if  len(tx.Vin) ==1 &&
		len(tx.Vin[0].TXid) ==0 &&
		tx.Vin[0].Voutindex == -1{
	   	return true
	}
	return false
}

//对交易进行签名
func (tx *Transation) Sign(privkey ecdsa.PrivateKey, prevTXs map[string]Transation) {
	//coninbase交易不用签名
	if tx.isCoinBase(){
		return
	}
	//合法性检查过程
	for _,vin := range tx.Vin{
		if prevTXs[hex.EncodeToString(vin.TXid)].ID ==nil{
			log.Panic("Error: Vin中引用的交易ID不存在！")
		}
	}

	// 创建副本,用于计算签名
	txcopy := tx.TrimmedCopy()
	for inID,vin := range txcopy.Vin{
		prevTX := prevTXs[hex.EncodeToString(vin.TXid)] //拿到前一笔交易的结构体

		txcopy.Vin[inID].Signature = nil
		txcopy.Vin[inID].Pubkey = prevTX.Vout[vin.Voutindex].PubkeyHash
		txcopy.ID = txcopy.Hash()

		r,s,err := ecdsa.Sign(rand.Reader,&privkey, txcopy.ID)
		if err != nil{
			log.Panic(err)
		}
		signature := append(r.Bytes(),s.Bytes()...)
		tx.Vin[inID].Signature = signature
	}

}

//复制本交易，返回一个新的副本，这是深拷贝
func (tx *Transation) TrimmedCopy() Transation {
	var inputs  []TXInput
	var outputs []TXOutput

	for _,vin := range tx.Vin{
		newIn :=TXInput{vin.TXid,vin.Voutindex,nil,nil}
		inputs = append(inputs,newIn)
	}
	for _,vout := range tx.Vout{
		newOut :=TXOutput{vout.Value,vout.PubkeyHash}
		outputs = append(outputs,newOut)
	}
	txCopy := Transation{tx.ID,inputs,outputs}
	return txCopy
}

//检验输入参数中的交易签名
func (tx Transation) Verify(prevTXs map[string]Transation) bool {
	//coinbase交易不用验证，直接通过
	if tx.isCoinBase(){
		return true
	}

	//再次检查tx.Vin中的引用交易ID是否包含在输入数据中，其实没有必要
	for _,vin :=range tx.Vin{
		if prevTXs[hex.EncodeToString(vin.TXid)].ID==nil{
			log.Panic("Error: Transation.Verify() failed!")
		}
	}

	//校验交易会破坏这个交易内容，用克隆的交易进行验证。
	txcopy := tx.TrimmedCopy()
	curve := elliptic.P256()  //椭圆曲线，用于加解密计算

	//计算交易整体的hash，必须把Vin.Signature清空，不能受之干扰
	for inID, vin := range tx.Vin{
		prevTX := prevTXs[hex.EncodeToString(vin.TXid)]
		txcopy.Vin[inID].Signature = nil
		txcopy.Vin[inID].Pubkey = prevTX.Vout[vin.Voutindex].PubkeyHash
		//上面的Vin.Pubkey明明是公钥，为什么用公钥Hash赋值？ 想不明白
		txcopy.ID = txcopy.Hash()

		//这个Sinature是由ecdsa的Sign函数生成的r、s拼接得到的，这里再把Signature分解成r/s
		r:=big.Int{}
		s:=big.Int{}
		siglen:=len(vin.Signature)
		r.SetBytes(vin.Signature[:(siglen/2)])
		s.SetBytes(vin.Signature[(siglen/2):])

		//把vin中的公钥Pubkey拆成两半得到x、y坐标，再恢复公钥数据结构体rawPubkey;
		//验证签名，实际上就是使用公钥数据结构体对信息的hash串重新计算一遍签名，再和原有签名(r+s)进行比较
		x:=big.Int{}
		y:=big.Int{}
		keylen := len(vin.Pubkey)

		x.SetBytes(vin.Pubkey[:(keylen/2)])
		y.SetBytes(vin.Pubkey[(keylen/2):])

		rawPubkey := ecdsa.PublicKey{curve,&x,&y}
		if ecdsa.Verify(&rawPubkey,txcopy.ID, &r,&s) ==false{
			return false
		}
		txcopy.Vin[inID].Pubkey = nil
	}
	return true
}

//根据发送方、接收方、转账金额创建出对应的交易
func NewUTXOTransation(from,to string,amount int, bc *BlockChain) *Transation{
	var inputs   []TXInput
	var outputs  []TXOutput

	wallets,err := NewWallets()
	if err !=nil{
		log.Panic(err)
	}
	//根据发送方地址找到对应的钱包，里面包含了公钥和私钥，可用于签名
	wallet := wallets.GetWallet(from)

	acc,validoutputs :=bc.FindSpendableOutputs2(HashPubKey(wallet.PublicKey),amount)
	if acc < amount{
		log.Panic("Error: Not enough funds")
	}

	//遍历输出，得到每笔交易的hash和输出序号
	for txid,outs := range validoutputs{
		txID,err :=hex.DecodeString(txid)  //把交易hash值从字符串形式转成字节切片形式
		if err !=nil{
			log.Panic(err)
		}

		//遍历每一个序号, 把这笔输出作为新交易的Vin项。
		//交易输入需要用户的公钥，只能从钱包集中找到指定的钱包，再得到公钥。
		for _,out :=range outs{
			input := TXInput{txID,out,nil,wallet.PublicKey}
			inputs = append(inputs,input)
		}
	}

	//开始填写Vout项，注意这些Vin总金额可能>转账金额，要把剩下的余额还给发送方
	outputs = append(outputs,*NewTXOutput(amount,to))
	if acc > amount{
		outputs = append(outputs,*NewTXOutput(acc-amount,from))
	}

	//根据Vin和Vout填写交易结构体，注意要调用hash()方法计算这笔交易的hash值
	tx := Transation{nil,inputs,outputs}
	tx.ID = tx.Hash()

	//用私钥对交易进行签名
	bc.SignTransation(&tx, wallet.PrivateKey)
	return &tx
}














