package main

import (
	"flag"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"sort"
	"strconv"
	"strings"
)

type stacktrace struct {
	Num  int64
	Data string
	Sum  uint32
}

func (s *stacktrace) Length() int64 {
	return s.Num
}

func newStackTrace(data string) *stacktrace {
	pieces := strings.SplitN(data, " @ ", 2)
	num, err := strconv.ParseInt(pieces[0], 10, 64)
	h := fnv.New32a()
	h.Write([]byte(pieces[1]))
	if err != nil {
		panic(fmt.Sprintf("invalid integer '%s'", pieces[0]))
	}
	return &stacktrace{
		Num:  num,
		Data: pieces[1],
		Sum:  h.Sum32(),
	}
}

func (s *stacktrace) String() string {
	return fmt.Sprintf("%d @ %s", s.Num, s.Data)
}

type ByNum []*stacktrace

func (a ByNum) Len() int {
	return len(a)
}
func (a ByNum) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a ByNum) Less(i, j int) bool {
	return a[i].Length() > a[j].Length()
}

var _ sort.Interface = ByNum{}

type SameByNum []*sameStack

func (a SameByNum) Len() int {
	return len(a)
}
func (a SameByNum) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a SameByNum) Less(i, j int) bool {
	return a[i].Length() > a[j].Length()
}

var _ sort.Interface = SameByNum{}

type file struct {
	Stacks map[uint32]*stacktrace
}

func newFile(f *string, threshold *int64) *file {
	if *f == "" {
		panic("no file specified")
	}
	dat, err := ioutil.ReadFile(*f)
	if err != nil {
		panic("couldn't read file")
	}
	data := string(dat)
	pieces := strings.Split(data, "\n\n")
	fmt.Println(*f, len(pieces))
	stacks := make(map[uint32]*stacktrace, len(pieces)-2)
	for _, v := range pieces {
		if strings.HasPrefix(v, "goroutine") || v == "" {
			inner := strings.SplitN(v, "\n", 2)
			if len(v) < 1 {
				continue
			}
			v = inner[1]
		}
		trace := newStackTrace(v)
		if trace.Num > *threshold {
			stacks[trace.Sum] = trace
		}
	}
	return &file{
		Stacks: stacks,
	}
}

type sameStack struct {
	LeftNum  int64
	RightNum int64
	Data     string
}

func (s *sameStack) Length() int64 {
	if s.LeftNum > s.RightNum {
		return s.LeftNum
	}
	return s.RightNum
}

func (s *sameStack) String() string {
	return fmt.Sprintf("Left: %d Right: %d @ %s", s.LeftNum, s.RightNum, s.Data)
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func diffFiles(left *file, right *file, diff *int64) (same []*sameStack, leftnotright []*stacktrace, rightnotleft []*stacktrace) {
	newleftmap := make(map[uint32]*stacktrace)
	for k, v := range left.Stacks {
		newleftmap[k] = v
	}
	same = make([]*sameStack, 0)
	leftnotright = make([]*stacktrace, 0)
	rightnotleft = make([]*stacktrace, 0)

	for k, r := range right.Stacks {
		l, ok := newleftmap[k]
		if ok {
			if abs(l.Num-r.Num) > *diff {
				same = append(same, &sameStack{
					LeftNum:  l.Num,
					RightNum: r.Num,
					Data:     l.Data,
				})
			}
			delete(newleftmap, k)
		} else {
			rightnotleft = append(rightnotleft, r)
		}
	}

	for _, v := range newleftmap {
		leftnotright = append(leftnotright, v)
	}
	sort.Sort(ByNum(leftnotright))
	sort.Sort(ByNum(rightnotleft))
	sort.Sort(SameByNum(same))
	return
}

func main() {
	var file1 = flag.String("left", "", "left stacktrace file to parse")
	var file2 = flag.String("right", "", "right stacktrace file to parse")
	var omit = flag.Bool("omitidentical", true, "omit stacktraces that are identical between files")
	var over = flag.Int64("over", 10, "don't show in output if # of goroutines <= over")
	var diff = flag.Int64("diff", 5, "don't show in output if diff of # of goroutines <= diff")
	flag.Parse()

	left := newFile(file1, over)
	right := newFile(file2, over)
	same, leftnotright, rightnotleft := diffFiles(left, right, diff)
	for _, v := range same {
		if v.LeftNum == v.RightNum && *omit {
			continue
		}
		fmt.Println(v.String())
	}
	if len(leftnotright) > 0 {
		fmt.Println("Left not Right")
		for _, v := range leftnotright {
			fmt.Println(v.String())
		}
	}
	if len(rightnotleft) > 0 {
		fmt.Println("Right not Left")
		for _, v := range rightnotleft {
			fmt.Println(v.String())
		}
	}
}
