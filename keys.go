package searchparty

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/denysvitali/searchparty-go/model"
)

func LoadKeys(dir string, key []byte) ([]model.MainKey, error) {
	keys := make([]model.MainKey, 0)
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, v := range files {
		if v.IsDir() {
			continue
		}
		var toAddKey model.MainKey
		switch {
		case strings.HasSuffix(v.Name(), ".keys"):
			f, err := os.Open(path.Join(dir, v.Name()))
			if err != nil {
				return nil, fmt.Errorf("failed to open file %s: %w", v.Name(), err)
			}
			toAddKey, err = LoadStaticKey(f)
			if err != nil {
				return nil, err
			}
		case strings.HasSuffix(v.Name(), ".record"):
			f, err := os.Open(path.Join(dir, v.Name()))
			if err != nil {
				return nil, fmt.Errorf("failed to open file %s: %w", v.Name(), err)
			}
			toAddKey, err = LoadDynamicKey(f, key)
			if err != nil {
				return nil, err
			}
		default:
			continue
		}
		keys = append(keys, toAddKey)
	}
	return keys, nil
}
