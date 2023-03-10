package nfa

import (
	"fmt"
)

const (
	DFA_MAX   = 254 //DFA 最多节点数
	F         = -1  //用于初始化跳转表
	MAX_CHARS = 128 //128个ascii字符
)

type ACCEPT struct {
	acceptString string //接收节点对应的执行代码字符串
	anchor       Anchor
}

type DFA struct {
	group        int  //后面执行最小化算法时有用
	mark         bool //当前节点是否已经设置好接收字符对应的边
	anchor       Anchor
	set          []*NFA //dfa节点对应的nfa节点集合
	state        int    //dfa 节点号码
	acceptString string
	isAccepted   bool
}

type NfaDfaConverter struct {
	nstates    int     //当前dfa 节点计数
	lastMarked int     //下一个需要处理的dfa节点
	dtrans     [][]int //dfa状态机的跳转表
	accepts    []*ACCEPT
	dstates    []DFA   //所有dfa节点的集合
	groups     [][]int //用于dfa节点分区
	inGroups   []int   //根据节点值给出其所在分区
	numGroups  int     //当前分区数
}

func NewNfaDfaConverter() *NfaDfaConverter {
	n := &NfaDfaConverter{
		nstates:    0,
		lastMarked: 0,
		dtrans:     make([][]int, DFA_MAX),
		dstates:    make([]DFA, DFA_MAX),
		groups:     make([][]int, DFA_MAX),
		inGroups:   make([]int, DFA_MAX),
		numGroups:  0,
	}

	for i := range n.dtrans {
		n.dtrans[i] = make([]int, MAX_CHARS)
	}

	for i := range n.groups {
		n.groups[i] = make([]int, 0)
	}

	return n
}

func (n *NfaDfaConverter) getUnMarked() *DFA {
	for ; n.lastMarked < n.nstates; n.lastMarked++ {
		debug := 0
		if n.dstates[n.lastMarked].state == 5 {
			debug = 1
			fmt.Printf("debug: %d", debug)
		}
		if n.dstates[n.lastMarked].mark == false {
			return &n.dstates[n.lastMarked]
		}
	}

	return nil
}

func (n *NfaDfaConverter) compareNfaSlice(setOne []*NFA, setTwo []*NFA) bool {
	//比较两个集合的元素是否相同
	if len(setOne) != len(setTwo) {
		return false
	}

	equal := false
	for _, nfaOne := range setOne {
		for _, nfaTwo := range setTwo {
			if nfaTwo == nfaOne {
				equal = true
				break
			}
		}

		if equal != true {
			return false
		}
	}

	return true
}

func (n *NfaDfaConverter) hasDfaContainsNfa(nfaSet []*NFA) (bool, int) {
	//查看是否存在dfa节点它对应的nfa节点集合与输入的集合相同
	for _, dfa := range n.dstates {
		if n.compareNfaSlice(dfa.set, nfaSet) == true {
			return true, dfa.state
		}
	}

	return false, -1
}

func (n *NfaDfaConverter) addDfaState(epsilonResult *EpsilonResult) int {
	//根据当前nfa节点集合构造一个新的dfa节点
	nextState := F
	if n.nstates >= DFA_MAX {
		panic("Too many DFA states")
	}

	nextState = n.nstates
	n.nstates += 1
	n.dstates[nextState].set = epsilonResult.results
	n.dstates[nextState].mark = false
	n.dstates[nextState].acceptString = epsilonResult.acceptStr
	//该节点是否为终结节点
	n.dstates[nextState].isAccepted = epsilonResult.hasAccepted

	n.dstates[nextState].anchor = epsilonResult.anchor
	n.dstates[nextState].state = nextState //记录当前dfa节点的编号s

	n.printDFAState(&n.dstates[nextState])
	fmt.Print("\n")

	return nextState
}

func (n *NfaDfaConverter) printDFAState(dfa *DFA) {
	fmt.Printf("DFA state : %d, it is nfa are: {", dfa.state)
	for _, nfa := range dfa.set {
		fmt.Printf("%d,", nfa.state)
	}

	fmt.Printf("}")
}

func (n *NfaDfaConverter) MakeDTran(start *NFA) {
	//根据输入的nfa状态机起始节点构造dfa状态机的跳转表
	startStates := make([]*NFA, 0)
	startStates = append(startStates, start)
	statesCopied := make([]*NFA, len(startStates))
	copy(statesCopied, startStates)

	//先根据起始状态的求Epsilon闭包操作的结果，由此获得第一个dfa节点
	epsilonResult := EpsilonClosure(statesCopied)
	n.dstates[0].set = epsilonResult.results
	n.dstates[0].anchor = epsilonResult.anchor
	n.dstates[0].acceptString = epsilonResult.acceptStr
	n.dstates[0].mark = false

	//debug purpose
	n.printDFAState(&n.dstates[0])
	fmt.Print("\n")
	nextState := 0
	n.nstates = 1 //当前已经有一个dfa节点
	//先获得第一个没有设置其跳转边的dfa节点
	current := n.getUnMarked()
	for current != nil {
		current.mark = true
		for c := 0; c < MAX_CHARS; c++ {
			nfaSet := move(current.set, c)
			if len(nfaSet) > 0 {
				statesCopied = make([]*NFA, len(nfaSet))
				copy(statesCopied, nfaSet)
				epsilonResult = EpsilonClosure(statesCopied)
				nfaSet = epsilonResult.results
			}

			if len(nfaSet) == 0 {
				nextState = F
			} else {
				//如果当前没有那个dfa节点对应的nfa节点集合和当前nfaSet相同，那么就增加一个新的dfa节点
				isExist, state := n.hasDfaContainsNfa(nfaSet)
				if isExist == false {
					nextState = n.addDfaState(epsilonResult)
				} else {
					nextState = state
				}
			}

			//设置dfa跳转表
			n.dtrans[current.state][c] = nextState
		}

		current = n.getUnMarked()
	}
}

func (n *NfaDfaConverter) PrintDfaTransition() {
	for i := 0; i < DFA_MAX; i++ {
		if n.dstates[i].mark == false {
			break
		}

		for j := 0; j < MAX_CHARS; j++ {
			if n.dtrans[i][j] != F {
				n.printDFAState(&n.dstates[i])
				fmt.Print(" jump to : ")
				n.printDFAState(&n.dstates[n.dtrans[i][j]])
				fmt.Printf("by character %s\n", string(j))
			}
		}
	}
}

func (n *NfaDfaConverter) initGroups() {
	//先把节点根据接收状态分为两个分区
	for i := 0; i < n.nstates; i++ {
		if n.dstates[i].isAccepted {
			n.groups[1] = append(n.groups[1], n.dstates[i].state)
			//记录状态点对应的分区
			n.inGroups[n.dstates[i].state] = 1
		} else {
			n.groups[0] = append(n.groups[0], n.dstates[i].state)
			n.inGroups[n.dstates[i].state] = 0
		}
	}

	n.numGroups = 2
}

func (n *NfaDfaConverter) printGroups() {
	//打印当前分区的信息
	for i := 0; i < n.numGroups; i++ {
		group := n.groups[i]
		fmt.Printf("分区号: %d", i)
		fmt.Println("分区节点如下:")
		for j := 0; j < len(group); j++ {
			fmt.Printf("%d ", group[j])
		}
		fmt.Printf("\n")
	}
}

func (n *NfaDfaConverter) minimizeGroups() {
	for {
		oldNumGroups := n.numGroups
		for current := 0; current < n.numGroups; current++ {
			//遍历每个分区，将不属于同一个分区的节点拿出来形成新的分区
			if len(n.groups[current]) <= 1 {
				//对于只有1个元素的分区不做处理
				continue
			}

			idx := 0
			//获取分区第一个元素
			first := n.groups[current][idx]
			newPartition := false
			for idx+1 < len(n.groups[current]) {
				next := n.groups[current][idx+1]
				//如果分区还有未处理的元素，那么看其是否跟first对应元素属于同一分区
				for c := MAX_CHARS - 1; c >= 0; c-- {
					gotoFirst := n.dtrans[first][c]
					gotoNext := n.dtrans[next][c]
					if gotoFirst != gotoNext && (gotoFirst == F || gotoNext == F || n.inGroups[gotoFirst] != n.inGroups[gotoNext]) {
						//如果first和next对应的两个节点在接收相同输入后跳转的节点不在同一分区，那么需要将next对应节点加入新分区
						//先将next对应节点从当前分区拿走
						n.groups[current] = append(n.groups[current][:idx+1], n.groups[current][idx+2:]...)
						n.groups[n.numGroups] = append(n.groups[n.numGroups], next)
						n.inGroups[next] = n.numGroups
						newPartition = true
						break
					}
				}

				if !newPartition {
					//如果next没有被拿出当前分区，那么idx要增加去指向下一个元素
					idx += 1
				} else {
					//如果next被挪出当前分区，那么idx不用变就能指向下一个元素♀️
					newPartition = false
				}
			}

			if len(n.groups[n.numGroups]) > 0 {
				//有新的分区生成，因此分区计数要加1
				n.numGroups += 1
			}
		}

		if oldNumGroups == n.numGroups {
			//如果没有新分区生成，算法结束
			break
		}
	}

	n.printGroups()
}

func (n *NfaDfaConverter) fixTran() {
	newDTran := make([][]int, DFA_MAX)
	//新建一个跳转表
	for i := 0; i < n.numGroups; i++ {
		newDTran[i] = make([]int, MAX_CHARS)
	}

	/*
		我们把当前分区号对应一个新的DFA节点，当前分区(用A表示)中取出一个节点，根据输入字符c获得其跳转的节点。
		然后根据跳转节点获得其所在分区(用B表示)，那么我们就得到新节点A在接收字符c后跳转到B节点	。
		这里我们从当前分区取出一个节点就行，因为在minimizeGroups中我们已经确保最终的分区中，里面每个节点在接收
		同样的字符后，所跳转的节点所在的分区肯定是一样的。
	*/
	for i := 0; i < n.numGroups; i++ {
		//从当前分区取出一个节点即可
		state := n.groups[i][0]
		for c := MAX_CHARS - 1; c >= 0; c-- {
			if n.dtrans[state][c] == F {
				newDTran[state][c] = F
			} else {
				destState := n.dtrans[state][c]
				destPartition := n.inGroups[destState]
				newDTran[state][c] = destPartition
			}
		}
	}

	n.dtrans = newDTran
}

func (n *NfaDfaConverter) MinimizeDFA() {
	n.initGroups()
	n.minimizeGroups()
	n.fixTran()
}

func (n *NfaDfaConverter) PrintMinimizeDFATran() {
	for i := 0; i < n.numGroups; i++ {
		for j := 0; j < MAX_CHARS; j++ {
			if n.dtrans[i][j] != F {
				fmt.Printf("from state %d jump to state %d with input: %s\n", i, n.dtrans[i][j], string(j))
			}
		}
	}
}
