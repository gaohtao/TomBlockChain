package main

import "fmt"

//测试创建区块的默克尔根
func TestCreateMerkleTreeRoot() {
	tx1 := NewCoinbaseTX("Tom", "")

	txin2  := TXInput{[]byte{}, -1, nil, nil}
	txout2  := NewTXOutput(10,"Jerry")
	tx2 := Transation{nil,[]TXInput{txin2},[]TXOutput{*txout2}}
	tx2.ID = tx2.Hash()    // 交易的ID就是hash值

	var Transations []*Transation
	Transations = append(Transations, tx1,&tx2)

	//初始化区块
	block :=&Block{
		[]byte{},
		1,
		[]byte{},
		[]byte{},

		1293022167,
		453281356,
		0,
		[]*Transation{},
		0,
	}
	block.createMerkleTreeRoot(Transations)
	fmt.Printf("%x\n",block.Merkleroot)
}

func TestNewSerialize(){
	//初始化区块
	block :=&Block{
		[]byte{},
		1,
		[]byte{},
		[]byte{},

		1293022167,
		453281356,
		0,
		[]*Transation{},
		0,
	}

	deBlock := DeserializeBlock(block.Serialize())
	//deBlock.ToString()

	fmt.Printf("%d\n",deBlock.Bits)

	deBlock.ToString()
}

//测试pow计算
func TestPow(){
	//初始化区块
	block :=&Block{
		[]byte{},
		1,
		[]byte{},
		[]byte{},
		1293022167,
		453281356,
		0,
		[]*Transation{},
		0,
	}

	pow := NewProofOfWork(block)
	nonce,_:=pow.Run()
	block.Nonce = nonce
	fmt.Print("Pow:",pow.Validate())
}

//测试区块链创建、新增、遍历打印区块数据
func TestBoltDB(){
	bc := NewBlockChain("")
	fmt.Printf("bc=",bc)
	bc.MineBlock([]*Transation{})
	bc.MineBlock([]*Transation{})

	bc.PrintBlockChain()
}

//测试命令行参数
func TestCliArgs(){
	bc := NewBlockChain(minneraddress)

	cli := CLI{bc}
	cli.Run()

}

