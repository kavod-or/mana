package menu

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Price stores euro cents exactly, without floating-point rounding.
type Price int64

var pricePattern = regexp.MustCompile(`^[0-9]+(?:\.[0-9]{1,2})?$`)

func (price *Price) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode || !pricePattern.MatchString(node.Value) {
		return fmt.Errorf("line %d: price must be a non-negative euro amount with at most two decimal places", node.Line)
	}
	parts := strings.SplitN(node.Value, ".", 2)
	fraction := "00"
	if len(parts) == 2 {
		fraction = (parts[1] + "0")[:2]
	}
	cents, err := strconv.ParseInt(parts[0]+fraction, 10, 64)
	if err != nil {
		return fmt.Errorf("line %d: price is too large", node.Line)
	}
	*price = Price(cents)
	return nil
}

func (price Price) German() string {
	return price.format(".", ",") + "\u00a0€"
}

func (price Price) English() string {
	return "€" + price.format(",", ".")
}

func (price Price) format(group, decimal string) string {
	whole := strconv.FormatInt(int64(price)/100, 10)
	for index := len(whole) - 3; index > 0; index -= 3 {
		whole = whole[:index] + group + whole[index:]
	}
	return fmt.Sprintf("%s%s%02d", whole, decimal, price%100)
}
