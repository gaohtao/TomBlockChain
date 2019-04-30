package main

import (
	"encoding/hex"
	"github.com/boltdb/bolt-master"
	"log"
)
type UTXOSet struct{
	bchain * BlockChain
}

const utxoBucket = "chainset"

//重置数据库的桶, 创建区块链时会调用
func (u UTXOSet) Reindex(){
	db:=u.bchain.db
	bucketName :=[]byte(utxoBucket)

	err := db.Update(func(tx *bolt.Tx) error{
		err2 := tx.DeleteBucket(bucketName)
		//当数据库文件不存在时删除出错，允许这种情况
		if err2 != nil && err2 != bolt.ErrBucketNotFound{
			log.Panic(err2)
		}

		_,err3 := tx.CreateBucket(bucketName)
		checkErr(err3)
		return nil
	})

	checkErr(err)

	UTXO := u.bchain.FindAllUTXO()
	err4 := db.Update(func(tx *bolt.Tx) error{
		b:=tx.Bucket(bucketName)

		for txID,outs := range UTXO{
			key,err5 := hex.DecodeString(txID)
			checkErr(err5)
			err6 := b.Put(key,outs.Serialize())  //存储的是映射，数据要求是序列化后的字节
			checkErr(err6)
		}

		return nil
	})
	checkErr(err4)
}

//在数据桶中查找指定公钥hash的用户UTXO
func (u UTXOSet) FindUTXObyPubkeyHash(pubkeyhash []byte) []TXOutput{
	var UTXOs []TXOutput

	db  := u.bchain.db
	err := db.View(func(tx *bolt.Tx) error{
		b := tx.Bucket([]byte(utxoBucket))
		c := b.Cursor()   //理解为桶内部的迭代器

		for k,v :=c.First(); k!=nil;k,v=c.Next(){
			outs := Deserialize(v)
			for _,out := range outs.Outputs{
				if out.CanBeUnlockedWith(pubkeyhash){
					UTXOs = append(UTXOs,out)
				}
			}
		}
		return nil
	})
	checkErr(err)
	return UTXOs
}

/*当链上增加一个区块时更新数据库桶中的UTXO，更新策略:
把新区块引用的输出从桶中删除
把新区块的输出添加到桶中 */
func (u UTXOSet) update(block *Block){
	db :=u.bchain.db
	err:=db.Update(func(tx *bolt.Tx)error{
		b:= tx.Bucket([]byte(utxoBucket))

		for _,transation := range block.Transations{
			if transation.isCoinBase() == false{
				for _,vin := range transation.Vin{
					updateouts := TXOutputs{}   //这里是个新的集合，用来装载删除了某笔输出的剩下的其他输出
					outsbytes :=b.Get(vin.TXid) //在桶中找到引用的交易数据
					outs := Deserialize(outsbytes) //数据反序列化，恢复成对象

					for outIdx,out := range outs.Outputs{
						// 这笔交易的多个输出中，跳过Vin引用的输出序号，其他序号的输出都要添加到新集合中
						if outIdx != vin.Voutindex{
							updateouts.Outputs = append(updateouts.Outputs,out)
						}
					}

					//到了这里表示肯定有一笔交易新删除了一个输出，万一这笔交易的所有输出都删除了，
					// 那么这个交易就没有用了，要从桶中删除
					if len(updateouts.Outputs)==0{
						err := b.Delete(vin.TXid)
						checkErr(err)
					}else{
						//这笔交易还有未使用的输出，用新集合替换就集合
						err := b.Put(vin.TXid, updateouts.Serialize())
						checkErr(err)
					}
				}
			}

			newOutputs := TXOutputs{}
			for _,out := range transation.Vout{
				newOutputs.Outputs = append(newOutputs.Outputs, out)
			}
			err:= b.Put(transation.ID, newOutputs.Serialize())
			checkErr(err)
		}
		return nil
	})

	checkErr(err)
}











