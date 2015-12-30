package main
import (
	"bufio"
	"os"
	"fmt"
	"strings"
)

func promptForString(prompt string) (string, error) {
	fmt.Printf("Enter %s: ", prompt)
	text, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err == nil {
		text = strings.TrimSpace(text)
	}
	return text, err
}