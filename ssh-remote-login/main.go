package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

func sshConfig() *ssh.ServerConfig {
	byteKey, _ := os.ReadFile("test_private_key.pem")
	privateKey, err := ssh.ParsePrivateKey(byteKey)
	fmt.Println(privateKey, err)

	if err != nil {
		log.Fatal("Error ", err)
	}

	config := &ssh.ServerConfig{
		// PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
		// 	if conn.User() == "user" && string(password) == "123" {
		// 		fmt.Println(string(password))
		// 		return nil, nil
		// 	}
		// 	return nil, fmt.Errorf("authentication failed")
		// },
		NoClientAuth: true,
	}
	config.AddHostKey(privateKey)
	return config
}

func handleSSHConnection(conn net.Conn, config *ssh.ServerConfig) {
	fmt.Println(conn.RemoteAddr(), "incoming con")
	defer conn.Close()
	sshConn, chans, _, err := ssh.NewServerConn(conn, config)

	if err != nil {
		fmt.Println("Error cannot established connection", err)
	}

	fmt.Println("SSH connect established from :", sshConn.RemoteAddr())
	// go ssh.DiscardRequests(reqs)
	fmt.Printf("channel length %d", len(chans))
	handleChans(chans)
}
func handleChans(chans <-chan ssh.NewChannel) {
	fmt.Println("handle channel 66", chans)
	for newChannel := range chans {
		// Handling the new channel
		fmt.Println("from handle channel", newChannel.ChannelType())
		if newChannel.ChannelType() == "session" {
			channel, requests, err := newChannel.Accept()
			if err != nil {
				fmt.Println("Failed to accept channel:", err)
				continue
			}

			go func(in <-chan *ssh.Request) {
				for req := range in {
					if req.Type == "shell" {
						req.Reply(true, nil)
					}
				}
			}(requests)
			terminal := term.NewTerminal(channel, " > ")
			//terminal := term.NewTerminal(channel, "$")
			terminal.Write([]byte("PTY allocation sucessfull\n"))
			go func() {
				defer channel.Close()
				for {
					line, err := terminal.ReadLine()
					if err != nil {
						break
					}
					fmt.Println(line)
					handleCommand(channel, line)

					/*** use bash instead of handleCommmand(c)***/
					// cmd := exec.Command("bash", "-c", line)
					// output, err2 := cmd.CombinedOutput()

					// if err2 != nil {
					// 	fmt.Fprintln(channel, "Error", err)
					// } else {
					// 	fmt.Fprint(channel, string(output))
					// }
				}
			}()
		}
	}
}

func handleCommand(channel ssh.Channel, cmd string) {
	cmdParts := strings.Fields(cmd)
	if len(cmdParts) == 0 {
		return
	}

	switch cmdParts[0] {
	case "cd":
		if len(cmdParts) < 2 {
			fmt.Fprintln(channel, "Usage: cd <directory>")
			return
		}
		err := os.Chdir(cmdParts[1])
		if err != nil {
			fmt.Fprintln(channel, "Error", err)
			return
		}
	case "exit":
		fmt.Fprintln(channel, "exiting")
		channel.Close()
		return
	default:

		// Prepare the command
		command := exec.Command(cmdParts[0], cmdParts[1:]...)
		command.Stderr = channel
		command.Stdout = channel
		command.Stdin = channel

		// Start the command
		err := command.Start()
		if err != nil {
			fmt.Fprintln(channel, "Error:", err)
			return
		}

		// Wait for the command to finish
		err = command.Wait()
		if err != nil {
			fmt.Fprintln(channel, "Error:", err)
		}
	}
}

func main() {
	fmt.Println("----1-----")
	config := sshConfig()
	li, err := net.Listen("tcp", "0.0.0.0:2022")

	if err != nil {
		log.Fatalln(err.Error())
	}
	defer li.Close()
	for {
		fmt.Println("-------2-----")
		conn, err := li.Accept()

		if err != nil {
			log.Fatalln(err.Error())
		}
		fmt.Println(conn.RemoteAddr())
		go handleSSHConnection(conn, config)
	}
}
