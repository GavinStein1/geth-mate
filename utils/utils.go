package utils

import (
	"bufio"
	"log"
	"os"
	"strconv"

	"golang.org/x/crypto/sha3"
)

func ConvertHexToInt64(hex string) int64 {
	hex = hex[2:]
	dec, err := strconv.ParseInt(hex, 16, 64)
	if err != nil {
		log.Fatalf("Utils.go - %v", err)
	}
	return dec
}

func GetFunctionSelector(signature string) []byte {
	hash := sha3.NewLegacyKeccak256()
	hash.Write([]byte(signature))
	return hash.Sum(nil)[:4]
}

func ReadAddressesFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}
