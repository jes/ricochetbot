package ricochetbot

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
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

	password, err := RandomPassword()
	if err != nil {
		return err
	}

	hashedPassword, err := HashedPassword(password)
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
		"HashedControlPassword", hashedPassword,
		"__OwningControllerProcess", strconv.Itoa(os.Getpid()),
		"AvoidDiskWrites", "1",
		"Log", "notice stdout",
	)

	torCommand.Stdout = os.Stdout
	torCommand.Stderr = os.Stderr

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
	bot.TorControlAuthentication = password

	return nil
}

func RandomPassword() (string, error) {
	return "jesdabest", nil
}

func RandomSalt(length int) (string, error) {
	salt := make([]byte, length)
	for i := 0; i < length; i++ {
		b, err := rand.Int(rand.Reader, big.NewInt(255))
		if err != nil {
			return "", err
		}
		salt[i] = byte(b.Int64())
	}
	return string(salt), nil
}

// reimplementation of torControlHashedPassword() from https://github.com/ricochet-im/ricochet/blob/master/src/utils/CryptoKey.cpp
//
// in plain English, this repeats the password and the salt over and over again, until 65536 bytes has been reached,
// and then takes SHA1 of that
func HashedPassword(password string) (string, error) {
	salt, err := RandomSalt(8)
	if err != nil {
		return "", err
	}

	// original was:
	//   int count = ((quint32)16 + (96 & 15)) << ((96 >> 4) + 6);
	// wtf?
	count := 65536

	tmp := salt + password
	data := ""

	// wtf?
	for count > 0 {
		c := len(tmp)
		if count < c {
			c = count
		}
		data += tmp[:c]
		count -= len(tmp)
	}

	md := sha1.Sum([]byte(data))

	// 60 is the hex-encoded value of 96, which is a constant used by Tor's algorithm
	return "16:" + strings.ToUpper(hex.EncodeToString([]byte(salt))) + "60" + strings.ToUpper(hex.EncodeToString(md[:])), nil
}
