package main

import (
	"fmt"
	"os"
)

const stateFormat = "%s\n"
const mapSize = 64

type State struct {
	file *os.File
}

func OpenState(fn string) (State, error) {
	s := State{}
	f, err := os.OpenFile(fn, os.O_RDWR|os.O_CREATE, 0700)
	if err != nil {
		return s, err
	}
	s.file = f
	return s, nil
}

func (s State) Close() error {
	return s.file.Close()
}

func (s State) Sync() error {
	return s.file.Sync()
}

func (s State) LastBootId() string {
	var bootId string
	n, err := fmt.Fscanf(s.file, stateFormat, &bootId)
	if err != nil || n < 1 {
		return ""
	}
	return bootId
}

func (s State) SetLastBootId(bootId string) error {
	_, err := s.file.Seek(0, 0)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(s.file, stateFormat, bootId)
	if err != nil {
		return err
	}
	return nil
}
