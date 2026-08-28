// Package hash membungkus bcrypt untuk hashing password.
package hash

import "golang.org/x/crypto/bcrypt"

// Password menghasilkan hash bcrypt dari password plaintext.
// bcrypt hanya membaca 72 byte pertama, karena itu DTO membatasi panjangnya.
func Password(plain string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// Compare mengecek apakah password plaintext cocok dengan hash tersimpan.
func Compare(hashed, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)) == nil
}
