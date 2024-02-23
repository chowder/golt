package golt

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"path"
	"strings"
)

type Tuple[T1 any, T2 any] struct {
	First  T1
	Second T2
}

func makeRandomState() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	length := 12
	result := make([]byte, length)

	for i := range result {
		randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			panic(err)
		}
		result[i] = charset[randomIndex.Int64()]
	}

	return string(result)
}

func getConfigFile(filename string) (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return path.Join(configDir, "golt", filename), nil
}

func WriteToConfigFile(content string, filename string) error {
	configFile, err := getConfigFile(filename)
	if err != nil {
		return err
	}

	err = os.MkdirAll(path.Dir(configFile), 0755)
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, []byte(content), 0600)
}

func ReadFromConfigFile(filename string) (string, error) {
	configFile, err := getConfigFile(filename)
	if err != nil {
		return "", err
	}
	contents, err := os.ReadFile(configFile)
	if err != nil {
		return "", err
	}

	return string(contents), nil
}

func ParseIntentPayload(payload string) map[string]string {
	result := make(map[string]string)

	parts := strings.Split(payload, ":")
	if parts[0] != "jagex" || len(parts) != 2 {
		log.Fatal("invalid payload", payload)
	}

	pairs := strings.Split(parts[1], ",")

	for _, pair := range pairs {
		keyValue := strings.Split(pair, "=")
		if len(keyValue) == 2 {
			result[keyValue[0]] = keyValue[1]
		}
	}

	return result
}

func parseJson(buffer []byte) (map[string]interface{}, error) {
	var data map[string]interface{}
	err := json.Unmarshal(buffer, &data)
	return data, err
}

func Die(err error) {
	log.Println(err)
	log.Println("Press Enter to exit...")
	fmt.Scanln()
	os.Exit(1)
}
