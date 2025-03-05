package add

import (
	"strings"
	"unicode"
)

type Node struct {
	Valuse    []string
	Operators []string
}

func ParseExpression(input string) ([]string, []string, error) {
	input = strings.ReplaceAll(input, " ", "")
	var values []string
	var operators []string

	var num string
	iss := 0
	for _, char := range input {
		if char == '(' {
			iss += 1
		}

		if unicode.IsDigit(char) || iss > 0 {
			num += string(char)
		} else {
			if num != "" {
				values = append(values, num)
				num = ""
			}
			if char == '+' || char == '-' || char == '*' || char == '/' {
				operators = append(operators, string(char))
			}
		}
		if char == ')' {
			iss -= 1
		}
	}

	if num != "" {
		values = append(values, num)
	}

	return values, operators, nil
}
