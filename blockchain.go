package main

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/boltdb/bolt-master"
	"log"
)

const dbFile = "blockchain.db"
const blockBucket ="blocks"
const genesisdata ="Tom blockChain"

// 注意，矿工的钱包地址一定要在钱包集中，不然以后转账时在钱包集中找不到矿工钱包地址，会出现map容器返回空指针。
const minneraddress = "14npxLBj8eGwCcGJPiuqoG4U6ssW7KA3hs"


//定义区块链的基本结构，hash+存储的数据库
type BlockChain struct{
	tip []byte    //链上最新的区块的hash值
	db * bolt.DB  //数据库
}


//定义区块链的迭代器
type BlockChainIterator struct{
	currenthash []byte
	db * bolt.DB
}

//获取区块链的迭代器
func (bc *BlockChain) iterator() * BlockChainIterator{
	bci := & BlockChainIterator{
		bc.tip,
		bc.db,
	}
	return bci
}

//获取迭代器所指的当前块, 读完后就指向前一个区块的hash值。 根据每个hash值在数据库中找到对应的区块序列化数据。
func (bci *BlockChainIterator) Next() *Block{
	var block *Block
	err := bci.db.View(func(tx *bolt.Tx) error{
		b:=tx.Bucket([]byte(blockBucket))
		data := b.Get(bci.currenthash)
		block = DeserializeBlock(data)
		return nil
	})
	if err!=nil{
		log.Panic(err)
	}
	bci.currenthash = block.PrevBlockHash
	return block
}

//遍历打印区块链
func (bc *BlockChain) PrintBlockChain(){
	bci := bc.iterator()

	for{
		block := bci.Next()
		block.ToString()
		fmt.Println()

		// PrevBlockHash长度==0表示这是创世区块，遍历结束
		if len(block.PrevBlockHash)==0{
			break
		}
	}
}

//创建一个区块链. 不存在就创建，存在就获取最新的区块信息, 参数是矿工地址:base58编码的字符串，不是字节数组
func NewBlockChain(address string) *BlockChain{
	var tip []byte

	db,err := bolt.Open(dbFile,0600,nil)
	if err !=nil{
		log.Panic(err)
	}
	err = db.Update(func(tx *bolt.Tx) error{

		b:=tx.Bucket([]byte(blockBucket))    //获得数据库的桶
		if b==nil{
			fmt.Println("区块链不存在，建立创世区块，建立新的区块链")
			b,err:=tx.CreateBucket([]byte(blockBucket))
			if err !=nil{
				log.Panic(err)
			}

			//建立CoinBase挖矿奖励交易
			transation := NewCoinbaseTX(address, genesisdata)
			genesis := NewGensisBlock([]*Transation{transation})   //建立创世区块

			err = b.Put(genesis.Hash, genesis.Serialize())  //区块数据写入数据库桶中
			if err !=nil{
				log.Panic(err)
			}
			err = b.Put([]byte("L"),genesis.Hash)  //新区块与“L”关联， L表示last，以后方便的寻找到区块链的最新头部
			if err !=nil{
				log.Panic(err)
			}
			tip = genesis.Hash
		}else{
			//区块链存在，获取“L”对应的区块数据
			tip = b.Get([]byte("L"))
		}

		return nil
	})

	if err !=nil{
		log.Panic(err)
	}

	//根据tip和db建立区块链对象
	bc :=BlockChain{tip,db}

	//重新创建UTXO数据库桶，从数据库文件中恢复UTXO
	set := UTXOSet{&bc}
	set.Reindex()

	return &bc
}

//这个就是链上的挖矿动作，链上添加一个区块，记录到数据库中
func (bc *BlockChain) MineBlock(transations []*Transation) *Block{
	//先检查输入的交易签名是否正确
	for _,tx := range transations {
		if bc.VerifyTransation(tx) == false {
			log.Panic("BlockChain.MineBlock() : ERROR: Invalid transation!")
		} else {
			fmt.Println("BlockChain.MineBlock() :transation verify success!")
		}
	}

	//从数据库中找到最新区块
	var lasthash []byte
	var lastheight  int32
	err := bc.db.View(func(tx * bolt.Tx)error{
		b:= tx.Bucket([]byte(blockBucket))
		lasthash = b.Get([]byte("L"))
		blockdata := b.Get(lasthash)
		block := DeserializeBlock(blockdata)
		lastheight = block.Height
		return nil
	})
	if err!=nil{
		log.Panic(err)
	}
	newBlock := NewBlock(transations, lasthash,lastheight+1)

	//把新区块写入数据库中
	bc.db.Update(func(tx *bolt.Tx) error{
		b:=tx.Bucket([]byte(blockBucket))
		err:=b.Put(newBlock.Hash,newBlock.Serialize())
		if err !=nil{
			log.Panic(err)
		}
		//把“L”关联新区块hash值
		err = b.Put([]byte("L"),newBlock.Hash)
		if err !=nil{
			log.Panic(err)
		}

		bc.tip = newBlock.Hash
		return nil
	})
	return newBlock
}

//================这段代码3个函数是老师视频中定义的，我认为有bug，改进带代码在下一段 ==================================================================

//找出指定用户address的所有未花费输出，需要遍历整个区块链
func (bc *BlockChain) FindUnspentTransations(pubkeyhash []byte) []Transation{
	var unspentTXs []Transation  //所有未花费的交易记录

	/*定义映射关系:
	key:    string（交易的hash值）
	value:  []int（存储已经花费的交易的序号）
	表示这笔交易（hash）的输出序号，已经被花费了。 */
	spendTXOs := make(map[string][]int)   //已花费交易记录

	// 第一层循环：遍历区块链的区块
	bci :=bc.iterator()
	for{
		block := bci.Next()

		//第二层循环：遍历该区块中的每一笔交易
		for _,tx := range block.Transations{
			txID := hex.EncodeToString(tx.ID)  //交易的hash值转成字符串形式

			//第三层循环： 遍历这笔交易中的输出
		loop3: for outIdx,out := range tx.Vout{

			//如果这笔交易在已花费交易记录中存在，说明必然有一个输出被花费。
			//通过循环找到记录的输出序号。序号对上了表示这个输出已经被花费，跳出来检查下一个输出。
			if spendTXOs[txID] != nil {
				for _,spentOut := range spendTXOs[txID]{
					if spentOut == outIdx{
						continue loop3
					}
				}
			}

			// 程序跑到这里说明这笔输出未被花费，写入未花费交易记录。注意检查指定地址
			// by the way, 最后一个区块的输出都是未被使用的
			if out.CanBeUnlockedWith(pubkeyhash){
				unspentTXs = append(unspentTXs, *tx)
			}
		}

			//遍历这笔交易中的输入，只要是输入就表示被使用了，需要添加到已花费交易记录中
			//CoinBase交易没有输入，跳过
			//注意 spendTXOs[inTxID]可能存入了多个输出序号，因此是个序号数组
			if tx.isCoinBase() == false{
				for _,in := range tx.Vin{
					if in.CanBeUnlockedWith(pubkeyhash){
						inTxID := hex.EncodeToString(in.TXid)
						//参数说明                   交易的哈希值，       输出索引
						spendTXOs[inTxID] = append(spendTXOs[inTxID], in.Voutindex)
					}
				}
			}
		}
		if len(block.PrevBlockHash)==0{
			break
		}

	}
	//fmt.Println(unspentTXs)
	return unspentTXs
}

func (bc *BlockChain) FindUTXO(pubkeyhash []byte) []TXOutput{
	var UTXOs []TXOutput
	unspentTXs:= bc.FindUnspentTransations(pubkeyhash)

	for _,tx :=range unspentTXs{
		for _,out := range tx.Vout{
			if out.CanBeUnlockedWith(pubkeyhash){
				UTXOs = append(UTXOs,out)
			}
		}
	}

	return UTXOs
}

//找出能满足指定（地址+金额）的 未花费交易输出，用这些交易作为输入够转账了
func (bc *BlockChain) FindSpendableOutputs(pubkeyhash []byte, amount int) (int,map[string][]int){
	unspentOutputs := make(map[string][]int)
	unspentTXs := bc.FindUnspentTransations(pubkeyhash)
	accumulated :=0   //检查累计金额

breakPoint: for _,tx := range unspentTXs{
	txID := hex.EncodeToString(tx.ID)

	for outIndex,out := range tx.Vout{
		if out.CanBeUnlockedWith(pubkeyhash) && accumulated <amount{
			accumulated += out.Value
			unspentOutputs[txID] = append(unspentOutputs[txID],outIndex)
			if accumulated >= amount{
				break breakPoint
			}
		}
	}
}

	return accumulated,unspentOutputs
}

//==================自己改进的代码=======================================================
//找出指定用户address的所有未花费输出，需要遍历整个区块链
//改进了bug：当发现交易中的一笔未花费输出时没有记录下输出的序号Voutindex，而是添加这条交易。应该要保存这条交易和输出序号。
//返回：多了一个每条交易的输出序号数组
func (bc *BlockChain) FindUnspentTransations2(pubkeyhash []byte) ([]Transation, map[string][]int){
	var unspentTXs []Transation  //所有未花费的交易记录
	var unspendTXOs = make(map[string][]int)   //未花费交易输出序号， key是交易的ID字符串，value是输出序号数组

	/*定义映射关系:
	key:    string（交易的hash值）
	value:  []int（存储已经花费的交易的序号）
	表示这笔交易（hash）的输出序号，已经被花费了。 */
	spendTXOs := make(map[string][]int)   //已花费交易记录

	// 第一层循环：遍历区块链的区块
	bci :=bc.iterator()
	for{
		block := bci.Next()

		//第二层循环：遍历该区块中的每一笔交易
		for _,tx := range block.Transations{
			txID := hex.EncodeToString(tx.ID)  //交易的hash值转成字符串形式

			//第三层循环： 遍历这笔交易中的输出
			loop3: for outIdx,out := range tx.Vout{

				//如果这笔交易在已花费交易记录中存在，说明必然有一个输出被花费。
				//通过循环找到记录的输出序号。序号对上了表示这个输出已经被花费，跳出来检查下一个输出。
				if spendTXOs[txID] != nil {   //找到这笔交易
					for _,spentOut := range spendTXOs[txID]{
						if spentOut == outIdx{  //找到了这笔输出序号
							continue loop3
						}
					}
				}

				// 程序跑到这里说明这笔输出未被花费，写入未花费交易记录。注意检查指定地址
				// by the way, 最后一个区块的输出都是未被使用的
				if out.CanBeUnlockedWith(pubkeyhash){
					unspentTXs = append(unspentTXs, *tx)
					unspendTXOs[txID] = append(spendTXOs[txID], outIdx)
				}
			}

			//遍历这笔交易中的输入，只要是输入就表示被使用了，需要添加到已花费交易记录中
			//CoinBase交易没有输入，跳过
			//注意 spendTXOs[inTxID]可能存入了多个输出序号，因此是个序号数组
			if tx.isCoinBase() == false{
				for _,in := range tx.Vin{
					if in.CanBeUnlockedWith(pubkeyhash){
						inTxID := hex.EncodeToString(in.TXid)
						//参数说明                   交易的哈希值，       输出索引
						spendTXOs[inTxID] = append(spendTXOs[inTxID], in.Voutindex)
					}
				}
			}
		}
		if len(block.PrevBlockHash)==0{
			break
		}

	}
	//fmt.Println(unspentTXs)
	return unspentTXs,unspendTXOs
}

func (bc *BlockChain) FindUTXO2(pubkeyhash []byte) []TXOutput{
	var UTXOs []TXOutput
	unspentTXs,unspendTXOs := bc.FindUnspentTransations2(pubkeyhash)

	for _,tx :=range unspentTXs{
		txID := hex.EncodeToString(tx.ID)  //交易的hash值转成字符串形式
		for _,outIdx:= range unspendTXOs[txID]{
			out := tx.Vout[outIdx]
			if out.CanBeUnlockedWith(pubkeyhash){
				UTXOs = append(UTXOs,out)
			}
		}
	}
	return UTXOs
}

//找出能满足指定（地址+金额）的 未花费交易输出，用这些交易作为输入够转账了
func (bc *BlockChain) FindSpendableOutputs2(pubkeyhash []byte, amount int) (int,map[string][]int){
	unspentOutputs := make(map[string][]int)
	unspentTXs,unspendTXOs := bc.FindUnspentTransations2(pubkeyhash)
	accumulated :=0   //检查累计金额

	breakPoint: for _,tx := range unspentTXs{
		txID := hex.EncodeToString(tx.ID)  //交易的hash值转成字符串形式
		for _,outIdx:= range unspendTXOs[txID]{
			out := tx.Vout[outIdx]
			if out.CanBeUnlockedWith(pubkeyhash) && accumulated <amount {
				accumulated += out.Value
				unspentOutputs[txID] = append(unspentOutputs[txID],outIdx)
				if accumulated >= amount{
					break breakPoint
				}
			}
		}
	}

	return accumulated,unspentOutputs
}

//查找链上所有未花费交易输出，用于计算各个钱包的余额
func (bc *BlockChain) FindAllUTXO() map[string]TXOutputs{
	var utxo = make(map[string]TXOutputs)   //未花费交易输出集合， key是交易的ID字符串，value是TXOutput切片

	/*定义映射关系:
	key:    string（交易的hash值）
	value:  []int（存储已经花费的交易的序号）
	表示这笔交易（hash）的输出序号，已经被花费了。 */
	spendTXOs := make(map[string][]int)   //已花费交易记录

	// 第一层循环：遍历区块链的区块
	bci :=bc.iterator()
	for{
		block := bci.Next()

		//第二层循环：遍历该区块中的每一笔交易
		for _,tx := range block.Transations{
			txID := hex.EncodeToString(tx.ID)  //交易的hash值转成字符串形式

			//第三层循环： 遍历这笔交易中的输出
			loop3: for outIdx,out := range tx.Vout{
				//如果这笔交易在已花费交易记录中存在，说明必然有一个输出被花费。
				//通过循环找到记录的输出序号。序号对上了表示这个输出已经被花费，跳出来检查下一个输出。
				if spendTXOs[txID] != nil {   //找到这笔交易
					for _,spentOut := range spendTXOs[txID]{
						if spentOut == outIdx{  //找到了这笔输出序号
							continue loop3
						}
					}
				}

				// 程序跑到这里说明这笔输出未被花费，写入未花费交易记录。注意检查指定地址
				outs :=utxo[txID]
				outs.Outputs = append(outs.Outputs, out)
				utxo[txID] = outs
			}

			//遍历这笔交易中的输入，只要是输入就表示被使用了，需要添加到已花费交易记录中
			//CoinBase交易没有输入，跳过
			//注意 spendTXOs[inTxID]可能存入了多个输出序号，因此是个序号数组
			if tx.isCoinBase() == false{
				for _,in := range tx.Vin{
					inTxID := hex.EncodeToString(in.TXid)
					//参数说明                   交易的哈希值，       输出索引
					spendTXOs[inTxID] = append(spendTXOs[inTxID], in.Voutindex)
				}
			}
		}
		if len(block.PrevBlockHash)==0{
			break
		}

	}

	return utxo
}

//================================================================================
func (bc * BlockChain) SignTransation(tx *Transation,prikey ecdsa.PrivateKey) {

	//定义映射，ID-->Transation， 保存所有的vin
	prevTXs := make(map[string]Transation)
	for _,vin :=range tx.Vin{
		//根据ID在之前的区块中找到这笔交易
		prevTX, err := bc.FindTransationById(vin.TXid)
		if err!=nil{
			log.Panic(err)
		}
		prevTXs[hex.EncodeToString(vin.TXid)]=prevTX
	}

	//再次封装，真实的签名动作。
	tx.Sign(prikey,prevTXs)
}


//在链中查找指定ID的交易，不存在就返回错误
func (bc *BlockChain) FindTransationById(ID []byte)(Transation,error){
	bci := bc.iterator()
	for{
		block :=bci.Next()
		for _,tx := range block.Transations{
			if bytes.Compare(tx.ID,ID)==0{
				return *tx, nil
			}
		}

		if len(block.PrevBlockHash)==0{
			break
		}
	}

	return Transation{},errors.New("error: transation is not find!")
}

//校验交易的数据签名是否正确
func (bc *BlockChain) VerifyTransation(tx *Transation) bool{
	//得到Vin中所有的引用交易，对这些引用的交易每一笔都进行检验
	prevTXs := make(map[string]Transation)

	for _,vin := range tx.Vin{
		prevTX,err := bc.FindTransationById(vin.TXid)
		checkErr(err)
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}
	return tx.Verify(prevTXs)
}

//获取最高高度
func (bc *BlockChain) GetBestHeight() int32{
	var lastBlock Block
	err := bc.db.View(func(tx *bolt.Tx) error{
		b:=tx.Bucket([]byte(blockBucket))
		lastHash:=b.Get([]byte("L"))
		blockdata := b.Get(lastHash)
		lastBlock = *DeserializeBlock(blockdata)

		return nil
	})
	checkErr(err)
	return lastBlock.Height
}

//获取全部区块的Hash，返回的是hash数组
func (bc *BlockChain) GetBlockHash() [][]byte {
	var blocks [][]byte
	bci := bc.iterator()
	for{
		block:=bci.Next()
		blocks = append(blocks,block.Hash)

		if len(block.PrevBlockHash)==0{
			break
		}
	}
	return blocks
}

//获取指定区块范围的Hash，返回的是hash数组
func (bc *BlockChain) GetBlockHashScope(low int32, high int32) [][]byte {
	var blocks [][]byte
	bci := bc.iterator()
	fmt.Printf("GetBlockHashScope: low=%d, high=%d\n", low,high)
	for{
		block:=bci.Next()

		if (block.Height >= high) && (block.Height <= high) {
			blocks = append(blocks,block.Hash)
		}

		if block.Height <= high{
			break;
		}

		if len(block.PrevBlockHash)==0{
			break
		}
	}
	return blocks
}

//从数据库中找出指定区块数据
func (bc *BlockChain) GetBlock(blockHash []byte) (Block, error) {
	var block Block
	err := bc.db.View(func(tx *bolt.Tx) error{
		b := tx.Bucket([]byte(blockBucket))
		blockData := b.Get(blockHash)
		if blockData == nil{
			return errors.New("GetBlock(): Block is not Found in Bucket!")
		}
		block = *DeserializeBlock(blockData)
		return nil
	})

	return block,err
}

//把区块写入数据库中
func (bc *BlockChain) AddBlock(block *Block){
	err := bc.db.Update(func(tx *bolt.Tx) error{
		b := tx.Bucket([]byte(blockBucket))

		//添加前先在桶中查找下这个区块Hash， 检查是否已经存在，不存在才添加
		blockIndb := b.Get(block.Hash)
		if blockIndb != nil{
			fmt.Printf("AddBlock(): Block is already exist in Bucket!\n")
			return nil
		}
		blockdata := block.Serialize()
		err  := b.Put(block.Hash,blockdata)
		checkErr(err)

		//检查区块链上的最新区块高度与新添加的区块高度， 更新最新区块映射
		lastHash := b.Get([]byte("L"))
		lastBlockData := b.Get(lastHash)
		lastBlock := DeserializeBlock(lastBlockData)
		if block.Height > lastBlock.Height {
			err:= b.Put([]byte("L"), block.Hash)
			checkErr(err)
			bc.tip = block.Hash
		}

		return nil
	})

	checkErr(err)
}
