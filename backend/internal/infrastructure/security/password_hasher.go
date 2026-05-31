package security

import "golang.org/x/crypto/bcrypt"

// BcryptPasswordHasher хэширует и сравнивает пароли через bcrypt.
type BcryptPasswordHasher struct{}

// NewBcryptPasswordHasher создаёт bcrypt adapter для application password service.
func NewBcryptPasswordHasher() *BcryptPasswordHasher {
	return &BcryptPasswordHasher{}
}

// Hash возвращает bcrypt-хэш пароля.
func (h *BcryptPasswordHasher) Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// Compare проверяет пароль против bcrypt-хэша.
func (h *BcryptPasswordHasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
