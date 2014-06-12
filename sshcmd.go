package sshcmd

import (
	"code.google.com/p/go.crypto/ssh"
	"io"
	"bufio"
	"strings"
	"errors"
)

type SSHCommand struct {
	Config *ssh.ClientConfig
	Server string
	StdOut chan string
	StdErr chan string
	StdIn chan string
}

func (s *SSHCommand) Execute(cmd string) error {
	var outnotifier chan error
	var errnotifier chan error
	var inpipe io.WriteCloser 
	var err error
	var innotifier chan error
	var client *ssh.Client
	var session *ssh.Session

	if s.Server == "" || s.Config == nil {
		return errors.New("Cannot execute command unless server and config are specified.")
	} 

	if client, err = ssh.Dial("tcp", s.Server + ":22", s.Config); err != nil { return err }
	defer client.Close()
	if session, err = client.NewSession(); err != nil { return err }
	defer session.Close()
	if s.StdOut != nil {
		pipe, err := session.StdoutPipe()
		if err != nil {
			return err
		}
		outnotifier = s.handleOutput(s.StdOut, pipe)
	}
	if s.StdErr != nil {
		pipe, err := session.StderrPipe()
		if err != nil {
			return err
		}
		errnotifier = s.handleOutput(s.StdErr, pipe )
	}
	if s.StdIn != nil {
		if inpipe, err = session.StdinPipe(); err != nil { return err }
		innotifier := make(chan error)
		go s.processInput(innotifier, inpipe)
	}

	
	err = session.Run(cmd)	
	if err != nil {
		return err
	}

	// Gather stream errors.
	var ioerrs []string
	

	if s.StdOut != nil { ioerrs = append(ioerrs, gatherErrors(outnotifier)...) }
	if s.StdErr != nil { ioerrs = append(ioerrs, gatherErrors(errnotifier)...) }
	if s.StdIn != nil { ioerrs = append(ioerrs, gatherErrors(innotifier)...) }

	if ioerrs != nil && len(ioerrs) > 0 {
		errstr := "Errors found processing IO streams: \n"
		for i := 0; i < len(ioerrs); i++ {
			errstr = errstr + ioerrs[i]
		}
		return errors.New(errstr)
	}
	
	return nil
}

func gatherErrors(notifier chan error) []string {
	var errlist []string
	for {
		err, ok := <- notifier
		if ok {
			errlist = append(errlist, err.Error())
		} else {
			return errlist
		}
	}
}

func (s *SSHCommand) processInput(notifier chan error, pipe io.WriteCloser ) {
	defer pipe.Close()
	defer close(notifier)
	for {
		in, ok := <-s.StdIn
		if !ok { return }
		inio := strings.NewReader(in)
		_, err := io.Copy(pipe,inio)
		if err != nil {
			notifier <- err
			return
		}
	}
}

func (s *SSHCommand) handleOutput(outchan chan string, pipe io.Reader) (chan error) {
	notifier := make(chan error)
	go s.processOutput(notifier, pipe, outchan)
	return notifier
}

func (s *SSHCommand) processOutput(notifier chan error, r io.Reader, outchan chan string)  {
	defer close(notifier)
	defer close(outchan)
	bufr := bufio.NewReader(r)
	var str string
	var err error
	for {
		str, err = bufr.ReadString('\n')
		outchan <- str
		if err != nil {
			break
		}
	}
	if err != io.EOF {
		notifier <- err
	}
}
