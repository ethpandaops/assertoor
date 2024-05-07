package wallet

import (
	"fmt"
	"math/big"
	"strings"
)

func GetReadableBalance(amount *big.Int, unitDigits, maxPreCommaDigitsBeforeTrim, digits int, addPositiveSign, trimAmount bool) string {
	// Initialize trimmedAmount and postComma variables to "0"
	fullAmount := ""
	trimmedAmount := "0"
	postComma := "0"
	proceed := ""

	if amount != nil {
		s := amount.String()

		if amount.Sign() > 0 && addPositiveSign {
			proceed = "+"
		} else if amount.Sign() < 0 {
			proceed = "-"
			s = strings.Replace(s, "-", "", 1)
		}

		l := len(s)

		// Check if there is a part of the amount before the decimal point
		switch {
		case l > unitDigits:
			// Calculate length of preComma part
			l -= unitDigits
			// Set preComma to part of the string before the decimal point
			trimmedAmount = s[:l]
			// Set postComma to part of the string after the decimal point, after removing trailing zeros
			postComma = strings.TrimRight(s[l:], "0")

			// Check if the preComma part exceeds the maximum number of digits before the decimal point
			if maxPreCommaDigitsBeforeTrim > 0 && l > maxPreCommaDigitsBeforeTrim {
				// Reduce the number of digits after the decimal point by the excess number of digits in the preComma part
				l -= maxPreCommaDigitsBeforeTrim
				if digits < l {
					digits = 0
				} else {
					digits -= l
				}
			}
			// Check if there is only a part of the amount after the decimal point, and no leading zeros need to be added
		case l == unitDigits:
			// Set postComma to part of the string after the decimal point, after removing trailing zeros
			postComma = strings.TrimRight(s, "0")
			// Check if there is only a part of the amount after the decimal point, and leading zeros need to be added
		case l != 0:
			// Use fmt package to add leading zeros to the string
			d := fmt.Sprintf("%%0%dd", unitDigits-l)
			// Set postComma to resulting string, after removing trailing zeros
			postComma = strings.TrimRight(fmt.Sprintf(d, 0)+s, "0")
		}

		fullAmount = trimmedAmount
		if postComma != "" {
			fullAmount += "." + postComma
		}

		// limit floating part
		if len(postComma) > digits {
			postComma = postComma[:digits]
		}

		// set floating point
		if postComma != "" {
			trimmedAmount += "." + postComma
		}
	}

	if trimAmount {
		return proceed + trimmedAmount
	}

	return proceed + fullAmount
}
