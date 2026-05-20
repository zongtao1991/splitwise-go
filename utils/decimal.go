package utils

import "math"

const epsilon = 1e-9
const precision = 100.0

func RoundToMoney(amount float64) float64 {
	return math.Round(amount*precision) / precision
}

func MoneyEqual(a, b float64) bool {
	return math.Abs(a-b) < epsilon
}

func MoneyLessThan(a, b float64) bool {
	return a < b-epsilon
}

func MoneyGreaterThan(a, b float64) bool {
	return a > b+epsilon
}

func MoneyLessOrEqual(a, b float64) bool {
	return a <= b+epsilon
}

func MoneyGreaterOrEqual(a, b float64) bool {
	return a >= b-epsilon
}

func SplitAmountEqually(total float64, count int) []float64 {
	if count <= 0 {
		return nil
	}
	if count == 1 {
		return []float64{RoundToMoney(total)}
	}
	
	perPerson := math.Floor(total*precision/float64(count)) / precision
	remainder := RoundToMoney(total - perPerson*float64(count))
	
	result := make([]float64, count)
	for i := 0; i < count; i++ {
		result[i] = perPerson
	}
	
	remainderCents := int(math.Round(remainder * precision))
	for i := 0; i < remainderCents; i++ {
		result[i] = RoundToMoney(result[i] + 1.0/precision)
	}
	
	return result
}

func SplitAmountByPercentages(total float64, percentages []float64) []float64 {
	count := len(percentages)
	if count == 0 {
		return nil
	}
	
	totalPercent := 0.0
	for _, p := range percentages {
		totalPercent += p
	}
	
	if MoneyEqual(totalPercent, 0) {
		return make([]float64, count)
	}
	
	result := make([]float64, count)
	allocated := 0.0
	
	for i, p := range percentages {
		if i < count-1 {
			result[i] = RoundToMoney(total * p / totalPercent)
			allocated = RoundToMoney(allocated + result[i])
		}
	}
	
	if count > 0 {
		result[count-1] = RoundToMoney(total - allocated)
	}
	
	return result
}
