package password

import "golang.org/x/crypto/bcrypt"

// Hash 使用 bcrypt 对明文密码生成哈希，cost 使用 bcrypt.DefaultCost（10）。
// 返回的哈希字符串可安全持久化至数据库，每次调用结果不同（内含随机 salt）。
func Hash(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Verify 校验明文密码与存储的 bcrypt 哈希是否匹配。
// 匹配返回 true，不匹配或出错均返回 false。
func Verify(plain, hashed string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)) == nil
}
