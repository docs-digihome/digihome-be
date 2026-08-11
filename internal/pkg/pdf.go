package pkg

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

func InspectPDF(path string) ([]byte, error) {
	cmd := exec.Command(
		"pdf-inspector",
		path,
	)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return output, nil
}
func InspectPDFByPage(path string, page int) ([]byte, error) {
	cmd := exec.Command(
		"pdf-inspector",
		path,
		"--pages",
		strconv.Itoa(page),
	)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return output, nil
}

var pdfInfoRegex = regexp.MustCompile(
	`(?i)^([A-Za-z]+)\s+\((\d+)\s+pages?,\s+confidence:\s+([0-9.]+)\)$`,
)

func GetTotalPage(path string) (int, error) {
	cmd := exec.Command(
		"pdf-inspector",
		"detect",
		path,
	)

	output, err := cmd.Output()
	if err != nil {
		return -1, err
	}

	result := strings.TrimSpace(string(output))
	matches := pdfInfoRegex.FindStringSubmatch(result)
	if len(matches) != 4 {
		return -1, err
	}

	pages, err := strconv.Atoi(matches[2])
	if err != nil {
		return -1, err
	}

	return pages, nil
}
