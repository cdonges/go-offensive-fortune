package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type fortuneFlags uint32

const (
	STR_RANDOM fortuneFlags = 1 << iota
	STR_ORDERED
	STR_ROTATED
)

type fortuneHeader struct {
	Version  uint32
	NumStr   uint32
	LongLen  uint32
	ShortLen uint32
	Flags    uint32
	Delim    byte
}

type weightedFile struct {
	path string
	size int64
}

func GetRandomFortune(dir string) (string, error) {
	if dir == "" {
		return "", errors.New("fortune directory is required")
	}

	datFiles, err := findDatFiles(dir)
	if err != nil {
		return "", err
	}
	if len(datFiles) == 0 {
		return randomFortuneSimple(dir)
	}

	picked := weightedRandomFile(datFiles)
	fortune, err := parseFortuneDat(picked)
	if err == nil && strings.TrimSpace(fortune) != "" {
		return fortune, nil
	}

	return randomFortuneSimple(dir)
}

func findDatFiles(dir string) ([]weightedFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read fortune dir: %w", err)
	}

	files := make([]weightedFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".dat" {
			continue
		}
		fullPath := filepath.Join(dir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, weightedFile{path: fullPath, size: info.Size()})
	}

	return files, nil
}

func weightedRandomFile(files []weightedFile) string {
	if len(files) == 0 {
		return ""
	}
	if len(files) == 1 {
		return files[0].path
	}

	var total int64
	for _, file := range files {
		if file.size <= 0 {
			file.size = 1
		}
		total += file.size
	}
	pick := rand.Int63n(total) + 1

	for _, file := range files {
		if file.size <= 0 {
			continue
		}
		pick -= file.size
		if pick <= 0 {
			return file.path
		}
	}

	return files[len(files)-1].path
}

func parseFortuneDat(datPath string) (string, error) {
	f, err := os.Open(datPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	header, err := readFortuneHeader(reader)
	if err != nil {
		return "", err
	}
	if header.NumStr == 0 {
		return "", errors.New("fortune file contains no entries")
	}

	pick := rand.Intn(int(header.NumStr))
	if pick < 0 {
		pick = 0
	}
	if _, err := f.Seek(24+int64(pick)*4, io.SeekStart); err != nil {
		return "", err
	}

	var offsetBytes [4]byte
	if _, err := io.ReadFull(f, offsetBytes[:]); err != nil {
		return "", err
	}
	pos := binary.BigEndian.Uint32(offsetBytes[:])

	contentPath := filepath.Join(filepath.Dir(datPath), strings.TrimSuffix(filepath.Base(datPath), filepath.Ext(filepath.Base(datPath))))
	contentFile, err := os.Open(contentPath)
	if err != nil {
		return "", err
	}
	defer contentFile.Close()

	if _, err := contentFile.Seek(int64(pos), io.SeekStart); err != nil {
		return "", err
	}

	contentReader := bufio.NewReader(contentFile)
	var builder strings.Builder
	for {
		b, err := contentReader.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", err
		}
		if b == header.Delim {
			break
		}
		builder.WriteByte(b)
	}

	fortune := builder.String()
	if fortuneFlags(header.Flags)&STR_ROTATED != 0 {
		fortune = rot13(fortune)
	}
	return fortune, nil
}

func readFortuneHeader(reader *bufio.Reader) (fortuneHeader, error) {
	var header fortuneHeader
	var err error

	header.Version, err = readUint32BigEndian(reader)
	if err != nil {
		return header, err
	}
	header.NumStr, err = readUint32BigEndian(reader)
	if err != nil {
		return header, err
	}
	header.LongLen, err = readUint32BigEndian(reader)
	if err != nil {
		return header, err
	}
	header.ShortLen, err = readUint32BigEndian(reader)
	if err != nil {
		return header, err
	}
	header.Flags, err = readUint32BigEndian(reader)
	if err != nil {
		return header, err
	}

	header.Delim, err = reader.ReadByte()
	if err != nil {
		return header, err
	}
	_, _ = reader.Discard(3)

	return header, nil
}

func readUint32BigEndian(reader *bufio.Reader) (uint32, error) {
	var buf [4]byte
	if _, err := io.ReadFull(reader, buf[:]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(buf[:]), nil
}

func randomFortuneSimple(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	var sources []string
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if filepath.Ext(entry.Name()) == ".dat" {
			continue
		}
		sources = append(sources, filepath.Join(dir, entry.Name()))
	}
	if len(sources) == 0 {
		return "", errors.New("no fortune source files found")
	}

	var all []string
	for _, file := range sources {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		for _, chunk := range strings.Split(string(data), "%") {
			fortune := strings.TrimSpace(chunk)
			if fortune == "" {
				continue
			}
			all = append(all, rot13(fortune))
		}
	}
	if len(all) == 0 {
		return "", errors.New("no fortunes found in source files")
	}

	rand.New(rand.NewSource(time.Now().UnixNano()))
	return all[rand.Intn(len(all))], nil
}

func rot13(input string) string {
	chars := []rune(input)
	for i, ch := range chars {
		switch {
		case ch >= 'a' && ch <= 'z':
			if ch > 'm' {
				chars[i] = ch - 13
			} else {
				chars[i] = ch + 13
			}
		case ch >= 'A' && ch <= 'Z':
			if ch > 'M' {
				chars[i] = ch - 13
			} else {
				chars[i] = ch + 13
			}
		}
	}
	return string(chars)
}
