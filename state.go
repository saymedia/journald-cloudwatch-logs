package main

import (
	"fmt"
	"os"
)

const stateFormat = "%s\n%s\n"
const mapSize = 64

type state struct {
	file *os.File
}

func openState(fn string) (state, error) {
	s := state{}
	f, err := os.OpenFile(fn, os.O_RDWR|os.O_CREATE, 0700)
	if err != nil {
		return s, err
	}
	s.file = f
	return s, nil
}

func (s state) close() error {
	return s.file.Close()
}

func (s state) sync() error {
	return s.file.Sync()
}

func (s state) lastState() (string, string) {
	var bootID string
	var seqToken string
	_, err := s.file.Seek(0, 0)
	if err != nil {
		return "", ""
	}
	n, err := fmt.Fscanf(s.file, stateFormat, &bootID, &seqToken)
	if err != nil || n < 2 {
		return "", ""
	}
	return bootID, seqToken
}

func (s state) setState(bootID, seqToken string) error {
	_, err := s.file.Seek(0, 0)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(s.file, stateFormat, bootID, seqToken)
	if err != nil {
		return err
	}
	return nil
}
