package thing

import (
	"fmt"
	"os"
)

func Read(p string) error {
	_, err := os.ReadFile(p)
	if err != nil {
		return fmt.Errorf("read: %v", err)
	}
	return nil
}
