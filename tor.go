package ricochetbot

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func (bot *RicochetBot) ManageTor(datadir string) error {
	err := ioutil.WriteFile(datadir+"/empty-torrc", []byte("Log notice stdout"), 0644)
	if err != nil {
		return err
	}

	torCommand := exec.Command("/usr/bin/tor",
		"--defaults-torrc", datadir+"/empty-torrc",
		"-f", datadir+"/empty-torrc",
		"DataDirectory", datadir,
		"SocksPort", "auto",
		"ControlPort", "auto",
		"ControlPortWriteToFile", datadir+"/control-port",
		"__OwningControllerProcess", strconv.Itoa(os.Getpid()),
		"AvoidDiskWrites", "1",
		"Log", "notice stdout",
	)

	torCommand.Stdout = os.Stdout
	torCommand.Stderr = os.Stderr

	// unlink the control-port file
	err = os.Remove(datadir + "/control-port")
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// start tor
	err = torCommand.Start()
	if err != nil {
		return err
	}

	// wait for tor to tell us the control port
	// TODO: if tor exits we should notice instead of polling forever
	for {
		_, err := os.Stat(datadir + "/control-port")
		if err == nil {
			break
		}

		time.Sleep(1 * time.Second)
	}

	// read the control port file
	bytes, err := ioutil.ReadFile(datadir + "/control-port")
	if err != nil {
		return err
	}
	torControlPort := strings.TrimRight(string(bytes), "\n")

	if torControlPort[:5] != "PORT=" {
		return errors.New("can't understand tor control port: " + torControlPort)
	}
	bot.TorControlAddress = torControlPort[5:]
	fmt.Println(bot.TorControlAddress)
	bot.TorControlType = "tcp4"

	return nil
}
