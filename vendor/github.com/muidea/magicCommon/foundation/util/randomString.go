package util

import (
	"crypto/rand"
	"encoding/binary"
	"math"
	randv2 "math/rand/v2"
	"strconv"
	"sync"
	"time"
)

var randPool = sync.Pool{
	New: func() any {
		var b [16]byte
		if _, err := rand.Read(b[:]); err != nil {
			return randv2.New(randv2.NewPCG(uint64(time.Now().UnixNano()), 1))
		}
		return randv2.New(randv2.NewPCG(binary.LittleEndian.Uint64(b[:8]), binary.LittleEndian.Uint64(b[8:])))
	},
}

var codeRandPool = sync.Pool{
	New: func() any {
		var b [16]byte
		if _, err := rand.Read(b[:]); err != nil {
			// 兜底方案
			return randv2.New(randv2.NewPCG(uint64(time.Now().UnixNano()), 1))
		}
		seed1 := binary.LittleEndian.Uint64(b[0:8])
		seed2 := binary.LittleEndian.Uint64(b[8:16])
		return randv2.New(randv2.NewPCG(seed1, seed2))
	},
}

// RandomIdentifyCode 生成6位数字验证码，范围在100000-999999之间（不会以0开头）
func RandomIdentifyCode() string {
	r := codeRandPool.Get().(*randv2.Rand)
	defer codeRandPool.Put(r)

	// 直接生成 100,000 到 999,999 之间的随机数
	// r.IntN(900000) 返回 [0, 900000) 之间的整数
	// 加上 100000 后，范围变为 [100000, 1000000)，即标准的6位非0开头数字
	vcode := r.IntN(900000) + 100000

	// 使用 strconv 代替 fmt.Sprintf，性能提升约 5-10 倍
	return strconv.Itoa(vcode)
}

// RandomSpec0 根据各种选项创建随机字符串，使用提供的随机源。
// 如果 start 和 end 都为0，则 start 和 end 被设置为 ' ' 和 'z'，即 ASCII 可打印字符。
// 如果 letters 和 numbers 都为 false，则 start 和 end 被设置为 0 和 math.MaxInt32。
// 如果 set 不为 nil，则从 start 和 end 之间的字符中选择。
// 该方法接受一个用户提供的 randv2.Rand 实例作为随机源。
func RandomSpec0(count uint, start, end int, letters, numbers bool,
	chars []rune, r *randv2.Rand) string {
	if count == 0 {
		return ""
	}

	// 1. 初始化边界
	if start == 0 && end == 0 {
		end = 'z' + 1
		start = ' '
		if !letters && !numbers {
			start = 0
			end = math.MaxInt32
		}
	}

	buffer := make([]rune, count)
	var pool []rune

	// 2. 预建池逻辑
	if len(chars) > 0 {
		pool = chars
	} else if letters || numbers {
		for i := start; i < end; i++ {
			ch := rune(i)
			if (letters && ((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z'))) ||
				(numbers && (ch >= '0' && ch <= '9')) {
				pool = append(pool, ch)
			}
		}
	}

	// 3. 核心修复：检查 poolLen 是否真的大于 0
	poolLen := uint(len(pool))
	gap := uint(1)
	if end > start {
		gap = uint(end - start)
	}

	for i := 0; i < int(count); i++ {
		var ch rune
		// 只有 pool 真正有数据时才从 pool 取
		if poolLen > 0 {
			ch = pool[r.UintN(poolLen)]
		} else {
			// 兜底：如果要求的 letters/numbers 在范围内不存在，
			// 或者本身没要求过滤，则退回到 gap 随机
			ch = rune(r.UintN(gap) + uint(start))
		}

		// 4. Unicode 代理对逻辑优化
		if ch >= 0xD800 && ch <= 0xDBFF { // 高代理
			if i < int(count)-1 {
				buffer[i] = ch
				i++
				buffer[i] = rune(0xDC00 + r.UintN(128))
			} else {
				i-- // 重新生成当前位
			}
		} else if ch >= 0xDC00 && ch <= 0xDFFF { // 低代理
			if i > 0 {
				buffer[i] = ch
				i-- // 往回填一个高代理
				buffer[i] = rune(0xD800 + r.UintN(128))
			} else {
				i-- // 第一个字符不能是低代理，重新生成
			}
		} else {
			buffer[i] = ch
		}
	}

	return string(buffer)
}

// RandomSpec1 创建一个指定长度的随机字符串。
// 字符将从参数指示的字母数字字符集中选择。
func RandomSpec1(count uint, start, end int, letters, numbers bool) string {
	r := randPool.Get().(*randv2.Rand)
	defer randPool.Put(r)

	return RandomSpec0(count, start, end, letters, numbers, nil, r)
}

// RandomAlphaOrNumeric 创建一个指定长度的随机字符串。
// 字符将从参数指示的字母数字字符集中选择。
func RandomAlphaOrNumeric(count uint, letters, numbers bool) string {
	return RandomSpec1(count, 0, 0, letters, numbers)
}

// RandomString 创建一个指定长度的随机字符串。
func RandomString(count uint) string {
	return RandomAlphaOrNumeric(count, false, false)
}

// RandomStringSpec0 创建一个指定长度的随机字符串。
func RandomStringSpec0(count uint, set []rune) string {
	r := randPool.Get().(*randv2.Rand)
	defer randPool.Put(r)

	return RandomSpec0(count, 0, len(set), false, false, set, r)
}

// RandomStringSpec1 创建一个指定长度的随机字符串。
func RandomStringSpec1(count uint, set string) string {
	return RandomStringSpec0(count, []rune(set))
}

// RandomAscII 创建一个指定长度的随机字符串。
// 字符将从 ASCII 值在 32 到 126（包括）之间的字符集中选择。
func RandomAscII(count uint) string {
	return RandomSpec1(count, 32, 127, false, false)
}

// RandomAlphabetic 创建一个指定长度的随机字符串。
// 字符将从字母字符集中选择。
func RandomAlphabetic(count uint) string {
	return RandomAlphaOrNumeric(count, true, false)
}

// RandomAlphanumeric 创建一个指定长度的随机字符串。
// 字符将从字母数字字符集中选择。
func RandomAlphanumeric(count uint) string {
	return RandomAlphaOrNumeric(count, true, true)
}

// RandomNumeric 创建一个指定长度的随机字符串。
// 字符将从数字字符集中选择。
func RandomNumeric(count uint) string {
	return RandomAlphaOrNumeric(count, false, true)
}
