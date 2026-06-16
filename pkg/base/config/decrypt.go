package config

import (
	"github.com/liukunxin/go-infra/internal/option"
	"github.com/liukunxin/go-infra/pkg/base/config/crypto"
)

// Decryptor 配置解密接口。实现此接口即可接入自定义解密方案。
type Decryptor interface {
	DecryptBytes(data []byte) ([]byte, error)
}

// WithDecrypt 启用配置解密。传入 nil 时不做任何解密处理。
func WithDecrypt(d Decryptor) Option {
	return option.Func[optionConfig](func(c *optionConfig) error {
		c.decryptor = d
		return nil
	})
}

// AESKeyFromEnv 从指定环境变量读取 AES-256 密钥并构造解密器。
// 密钥格式支持 hex（64字符）或 base64 编码。
func AESKeyFromEnv(envKey string) Decryptor {
	return &aesEnvDecryptor{envKey: envKey}
}

type aesEnvDecryptor struct {
	envKey string
}

func (d *aesEnvDecryptor) DecryptBytes(data []byte) ([]byte, error) {
	key, err := crypto.ParseKeyFromEnv(d.envKey)
	if err != nil {
		return nil, err
	}
	return crypto.DecryptYAML(key, data)
}
