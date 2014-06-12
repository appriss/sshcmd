package sshcmd

import (
	"testing"
	"time"
	"code.google.com/p/go.crypto/ssh"
	"io/ioutil"
	"sync"
)

func loadPEM() (ssh.Signer, error) {
	privateKey, _ := ioutil.ReadFile("/Users/dpantke/.ssh/diftp_lab_new")
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	return signer, nil
}

func TestSSHCommand(t *testing.T) {
	signer, err := loadPEM()
	if err != nil {
		t.Errorf("%s", err)
	}

	config := &ssh.ClientConfig{
		User: "ec2-user",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
	}

	sshcmd := &SSHCommand{Config: config, Server: "54.204.196.218"}

	output := make(chan string)
	errout := make(chan string)

	sshcmd.StdOut = output
	sshcmd.StdErr = errout

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		var out string
		var err string
		for {
			select {
			case out = <- output:
				t.Logf("OUTPUT: %s", out)
			case err = <- errout:
				t.Logf("ERROR: %s", err)	
			case <- time.After(10 * time.Second):
				t.Logf("Breaking infinite read loop because of timeout")
				return
			}
			if out == "" && err == "" {
				t.Logf("Finished reading from stdout and stderr")
				break
			}
		}
	}(wg)

	err = sshcmd.Execute("ls -la")
	if err != nil {
		t.Errorf("%s", err)
	}

	wg.Wait()
}