package main

import "crypto/sha256"

//定义默克尔树节点
type MerkleNode struct{
	Left  *MerkleNode
	Right *MerkleNode
	Data  []byte    //存储左右节点拼接后计算出来的hash值
}

//定义默克尔树， 只要有个根节点就可以追溯整棵树节点
type MerkleTree struct{
	RootNode *MerkleNode
}


//根据左右节点或hash值创建树节点
func NewMerkleNode(left, right *MerkleNode, data []byte) *MerkleNode{

	//建立一个空节点，分两种情况填充data数值：
	//1 叶节点没有左右节点，只要填写data就行了
	//2 树节点必然有左右节点，计算出hash值填写到data中
	node := MerkleNode{}
	if(left==nil && right ==nil){
		node.Data = data
	}else{
		totalData := append(left.Data,right.Data...)
		hash1 := sha256.Sum256(totalData)
		hash2 := sha256.Sum256(hash1[:])
		node.Data = hash2[:]
	}
	node.Left = left
	node.Right= right

	return &node
}

//创建完整的默克尔树，输入参数是hash值数组，每一个hash值都是一个切片字节数组，就表现成了二维数组形式。
func NewMerkleTree(data [][]byte) *MerkleTree  {

	// 定义节点列表
	var nodes  []MerkleNode

	// 创建全部初始叶节点
	for _,datum := range(data){
		node := NewMerkleNode(nil,nil,datum)
		nodes = append(nodes,*node)
	}

	//第一层循环代表树的层数，节点越多那么层级越多，第一层的循环次数就多。
	//例如5个节点就3层，循环3次。 每次循环时nSize就代表本层的节点个数
	j:=0   //nodes中节点序号标记，表示了在每一层中第一个节点序号
	for nSize:=len(nodes);nSize>1;nSize=(nSize+1)/2{
		//第二层循环，在每一层中两两计算出一个新节点，插入到节点列表尾巴，孤单节点就与自身计算出一个新节点。
		//在nodes中存储了所有原始的节点，依靠每层的节点个数划分层
		for i:=0;i<nSize;i+=2{
			i2:=min(i+1,nSize-1) //对于剩下的孤单节点i+1会越界，因此要限制在nSize-1.
			node := NewMerkleNode(&nodes[j+i],&nodes[j+i2],nil)
			nodes = append(nodes,*node)
		}
		j+=nSize
	}
	//循环结束，nodes中的最后一个节点就是树根
	treeRoot := MerkleTree{&nodes[len(nodes)-1]}
	return &treeRoot
}