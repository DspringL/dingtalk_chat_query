// Package crypto 的单元测试 - 演示密钥生成与解密功能
package crypto

import (
	"fmt"
	"testing"
)

// TestGenerateKey 测试根据用户 UID 生成 AES 密钥
func TestGenerateKey(t *testing.T) {
	cases := []struct {
		userID      string
		expectLen   int
		description string
	}{
		{"123456789", 16, "普通数字 UID"},
		{"505256109", 16, "真实钉钉 UID"},
		{"", 16, "空字符串 UID"},
		{"user@example.com", 16, "邮箱格式 UID"},
	}

	fmt.Println("=== 测试 GenerateKey ===")
	for _, c := range cases {
		key := GenerateKey(c.userID)
		fmt.Printf("  UID=%-20q => 密钥=%-16s (长度=%d)\n", c.userID, key, len(key))

		if len(key) != c.expectLen {
			t.Errorf("[%s] 密钥长度期望 %d，实际 %d", c.description, c.expectLen, len(key))
		}
	}
}

// TestGenerateKeyDeterministic 验证相同 UID 每次生成相同密钥
func TestGenerateKeyDeterministic(t *testing.T) {
	fmt.Println("\n=== 测试 GenerateKey 幂等性 ===")
	uid := "505256109"
	key1 := GenerateKey(uid)
	key2 := GenerateKey(uid)

	fmt.Printf("  第一次生成: %s\n", key1)
	fmt.Printf("  第二次生成: %s\n", key2)

	if string(key1) != string(key2) {
		t.Errorf("相同 UID 生成的密钥不一致: %s != %s", key1, key2)
	} else {
		fmt.Println("  ✓ 两次结果一致，幂等性验证通过")
	}
}

// TestGenerateKeyUniqueness 验证不同 UID 生成不同密钥
func TestGenerateKeyUniqueness(t *testing.T) {
	fmt.Println("\n=== 测试 GenerateKey 唯一性 ===")
	uid1, uid2 := "111111111", "222222222"
	key1 := GenerateKey(uid1)
	key2 := GenerateKey(uid2)

	fmt.Printf("  UID=%s => 密钥=%s\n", uid1, key1)
	fmt.Printf("  UID=%s => 密钥=%s\n", uid2, key2)

	if string(key1) == string(key2) {
		t.Errorf("不同 UID 生成了相同密钥，存在碰撞风险")
	} else {
		fmt.Println("  ✓ 不同 UID 生成不同密钥，唯一性验证通过")
	}
}

// TestDecryptDatabaseFileNotFound 测试输入文件不存在时的错误处理
func TestDecryptDatabaseFileNotFound(t *testing.T) {
	fmt.Println("\n=== 测试 DecryptDatabase 文件不存在 ===")
	key := GenerateKey("123456789")
	err := DecryptDatabase("/tmp/nonexistent_dingtalk.db", "/tmp/out.db", key)

	fmt.Printf("  错误信息: %v\n", err)
	if err == nil {
		t.Error("期望返回错误，但实际返回 nil")
	} else {
		fmt.Println("  ✓ 正确返回了文件不存在错误")
	}
}
