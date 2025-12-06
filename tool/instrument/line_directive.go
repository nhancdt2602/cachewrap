package instrument

import (
	"os"
	"regexp"
)

// EnableLineDirective enables line directive in the file
// Line directive must be placed at the beginning of the line, otherwise
// it will be ignored by the compiler
func EnableLineDirective(filePath string) error {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	text := string(bytes)
	re := regexp.MustCompile(".*//line ")
	text = re.ReplaceAllString(text, "//line ")
	err = os.WriteFile(filePath, []byte(text), 0644)
	if err != nil {
		return err
	}
	return nil
}
