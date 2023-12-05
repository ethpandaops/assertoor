package utils

import (
	"fmt"
	"html/template"
	"math"
	"math/big"
	"os"
	"strings"
	"time"

	logger "github.com/sirupsen/logrus"
)

// GetTemplateFuncs will get the template functions
func GetTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"includeHTML": IncludeHTML,
		"html": func(x string) template.HTML {
			//nolint:gosec // ignore
			return template.HTML(x)
		},
		"bigIntCmp":  func(i *big.Int, j int) int { return i.Cmp(big.NewInt(int64(j))) },
		"mod":        func(i, j int) bool { return i%j == 0 },
		"sub":        func(i, j int) int { return i - j },
		"subUI64":    func(i, j uint64) uint64 { return i - j },
		"add":        func(i, j int) int { return i + j },
		"addI64":     func(i, j int64) int64 { return i + j },
		"addUI64":    func(i, j uint64) uint64 { return i + j },
		"addFloat64": func(i, j float64) float64 { return i + j },
		"mul":        func(i, j float64) float64 { return i * j },
		"div":        func(i, j float64) float64 { return i / j },
		"divInt":     func(i, j int) float64 { return float64(i) / float64(j) },
		"nef":        func(i, j float64) bool { return i != j },
		"gtf":        func(i, j float64) bool { return i > j },
		"ltf":        func(i, j float64) bool { return i < j },
		"inlist":     checkInList,
		"round": func(i float64, n int) float64 {
			return math.Round(i*math.Pow10(n)) / math.Pow10(n)
		},
		"percent":        func(i float64) float64 { return i * 100 },
		"contains":       strings.Contains,
		"formatTimeDiff": FormatTimeDiff,
		"formatDateTime": FormatDateTime,
	}
}

func checkInList(item, list string) bool {
	items := strings.Split(list, ",")
	for _, i := range items {
		if i == item {
			return true
		}
	}

	return false
}

func IncludeHTML(path string) template.HTML {
	b, err := os.ReadFile(path)
	if err != nil {
		logger.Printf("includeHTML - error reading file: %v", err)
		return ""
	}

	//nolint:gosec // ignore
	return template.HTML(string(b))
}

func FormatTimeDiff(ts time.Time) template.HTML {
	var timeStr string

	duration := time.Until(ts)
	absDuraction := duration.Abs()

	switch {
	case absDuraction < 1*time.Second:
		return template.HTML("now")
	case absDuraction < 60*time.Second:
		timeStr = fmt.Sprintf("%v sec.", uint(absDuraction.Seconds()))
	case absDuraction < 60*time.Minute:
		timeStr = fmt.Sprintf("%v min.", uint(absDuraction.Minutes()))
	case absDuraction < 24*time.Hour:
		timeStr = fmt.Sprintf("%v hr.", uint(absDuraction.Hours()))
	default:
		timeStr = fmt.Sprintf("%v day.", uint(absDuraction.Hours()/24))
	}

	if duration < 0 {
		//nolint:gosec // ignore
		return template.HTML(fmt.Sprintf("%v ago", timeStr))
	}

	//nolint:gosec // ignore
	return template.HTML(fmt.Sprintf("in %v", timeStr))
}

func FormatDateTime(ts time.Time) template.HTML {
	//nolint:gosec // ignore
	return template.HTML(ts.Format("02-01-2006 15:04:05"))
}
