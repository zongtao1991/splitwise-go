package utils

import "testing"

func TestRoundToMoney(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{10.0 / 3.0, 3.33},
		{0.1 + 0.2, 0.30},
		{1.234, 1.23},
		{1.235, 1.24},
		{10.0, 10.00},
	}

	for _, test := range tests {
		result := RoundToMoney(test.input)
		if result != test.expected {
			t.Errorf("RoundToMoney(%v) = %v, expected %v", test.input, result, test.expected)
		}
	}
}

func TestMoneyEqual(t *testing.T) {
	if !MoneyEqual(0.1+0.2, 0.3) {
		t.Error("0.1+0.2 should equal 0.3 with epsilon")
	}
	if MoneyEqual(1.0, 2.0) {
		t.Error("1.0 should not equal 2.0")
	}
}

func TestSplitAmountEqually(t *testing.T) {
	tests := []struct {
		total    float64
		count    int
		expected []float64
	}{
		{10.0, 3, []float64{3.34, 3.33, 3.33}},
		{10.0, 2, []float64{5.0, 5.0}},
		{1.0, 3, []float64{0.34, 0.33, 0.33}},
	}

	for _, test := range tests {
		result := SplitAmountEqually(test.total, test.count)
		if len(result) != len(test.expected) {
			t.Errorf("SplitAmountEqually(%v, %v) length = %v, expected %v",
				test.total, test.count, len(result), len(test.expected))
			continue
		}
		for i := range result {
			if result[i] != test.expected[i] {
				t.Errorf("SplitAmountEqually(%v, %v)[%v] = %v, expected %v",
					test.total, test.count, i, result[i], test.expected[i])
			}
		}
		var sum float64
		for _, v := range result {
			sum = RoundToMoney(sum + v)
		}
		if !MoneyEqual(sum, test.total) {
			t.Errorf("SplitAmountEqually(%v, %v) sum = %v, expected %v",
				test.total, test.count, sum, test.total)
		}
	}
}

func TestSplitAmountByPercentages(t *testing.T) {
	total := 100.0
	percentages := []float64{33.33, 33.33, 33.34}
	result := SplitAmountByPercentages(total, percentages)

	var sum float64
	for _, v := range result {
		sum = RoundToMoney(sum + v)
	}
	if !MoneyEqual(sum, total) {
		t.Errorf("SplitAmountByPercentages sum = %v, expected %v", sum, total)
	}
}
