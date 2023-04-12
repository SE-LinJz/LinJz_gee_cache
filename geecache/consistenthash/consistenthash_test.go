package consistenthash

import (
	"strconv"
	"testing"
)

// 如果要进行测试，那么我们需要明确地知道每一个传入的key的哈希值，那使用默认的crc32.ChecksumIEEE算法显然达不到目的，所以在这里使用了自定义的Hash算法，自定义的Hash算法只处理数字，传入字符串表示的数字，返回对应的数字即可
// 一开始，有2/4/6三个真实节点，对应的虚拟节点的哈希值分别是02/12/22，04/14/24，06/16/26
// 那么用例2/11/23/27选择的虚拟节点分别是02/12/24/02，也就是真实节点2/2/4/2
// 添加一个真实节点8，对应虚拟节点的哈希值是08/18/28，此时，用例27对应的虚拟节点从02变成了28，真实节点为8
func TestHashing(t *testing.T) {
	hash := New(3, func(key []byte) uint32 { // 自定义的hash函数
		i, _ := strconv.Atoi(string(key))
		return uint32(i)
	})

	// Given the above hash function, this will give replicas with "hashes":
	// 2,4,6,12,14,16,22,24,26
	hash.Add("6", "4", "2") // 添加三个实体节点

	// 这里存放的是测试数据key以及其对应的真实节点
	testCases := map[string]string{
		"2":  "2",
		"11": "2",
		"23": "4",
		"27": "2",
	}

	for k, v := range testCases {
		if hash.Get(k) != v {
			t.Errorf("Asking for %s,should have yielded %s", k, v)
		}
	}

	// Adds 8,18,28
	hash.Add("8")

	// 27 should now map to 8
	testCases["27"] = "8"

	for k, v := range testCases {
		if hash.Get(k) != v {
			t.Errorf("Asking for %s,should have yielded %s", k, v)
		}
	}

}
