package taxid

import (
	"errors"
	"strconv"
	"strings"
)

var (
	ErrInvalidCPF  = errors.New("invalid CPF")
	ErrInvalidCNPJ = errors.New("invalid CNPJ")
	ErrInvalidBRTaxID = errors.New("invalid Brazilian tax ID: must be a CPF (11 digits) or CNPJ (14 digits)")
)

type brValidator struct{}

func (brValidator) Validate(taxID string) error {
	digits := onlyDigits(taxID)
	switch len(digits) {
	case 11:
		return validateCPF(digits)
	case 14:
		return validateCNPJ(digits)
	default:
		return ErrInvalidBRTaxID
	}
}

func validateCPF(cpf string) error {
	if allSameDigit(cpf) {
		return ErrInvalidCPF
	}

	sum := 0
	for i := 0; i < 9; i++ {
		sum += digitAt(cpf, i) * (10 - i)
	}
	first := (sum * 10) % 11
	if first == 10 {
		first = 0
	}
	if first != digitAt(cpf, 9) {
		return ErrInvalidCPF
	}

	sum = 0
	for i := 0; i < 10; i++ {
		sum += digitAt(cpf, i) * (11 - i)
	}
	second := (sum * 10) % 11
	if second == 10 {
		second = 0
	}
	if second != digitAt(cpf, 10) {
		return ErrInvalidCPF
	}

	return nil
}

func validateCNPJ(cnpj string) error {
	if allSameDigit(cnpj) {
		return ErrInvalidCNPJ
	}

	weights1 := []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	weights2 := []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}

	first := remainder(cnpj, weights1)
	if first != digitAt(cnpj, 12) {
		return ErrInvalidCNPJ
	}

	second := remainder(cnpj, weights2)
	if second != digitAt(cnpj, 13) {
		return ErrInvalidCNPJ
	}

	return nil
}

func remainder(s string, weights []int) int {
	sum := 0
	for i, w := range weights {
		sum += digitAt(s, i) * w
	}
	r := sum % 11
	if r < 2 {
		return 0
	}
	return 11 - r
}

func onlyDigits(s string) string {
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "/", "")
	s = strings.ReplaceAll(s, " ", "")
	return s
}

func allSameDigit(s string) bool {
	for _, c := range s[1:] {
		if c != rune(s[0]) {
			return false
		}
	}
	return true
}

func digitAt(s string, i int) int {
	n, _ := strconv.Atoi(string(s[i]))
	return n
}