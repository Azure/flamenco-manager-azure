package azssh

import (
	"io/ioutil"
	"net"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"gitlab.com/blender-institute/flamenco-deploy-azure/flamenco"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// Context provides everything necessary to connect via SSH.
type Context struct {
	sshConfig *ssh.ClientConfig
}

func keyfileAuther() ssh.AuthMethod {
	// If you have an encrypted private key, the crypto/x509 package
	// can be used to decrypt it.
	keyfile := os.ExpandEnv("$HOME/.ssh/id_rsa")
	logger := logrus.WithField("keyfile", keyfile)

	key, err := ioutil.ReadFile(keyfile)
	if err != nil {
		logger.WithError(err).Info("unable to load private SSH key")
		return nil
	}

	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		logger.WithField("reason", err).Info("unable to parse private key file")
		return nil
	}

	return ssh.PublicKeys(signer)
}

func sshAgent() ssh.AuthMethod {
	agentAddr := os.Getenv("SSH_AUTH_SOCK")
	if agentAddr == "" {
		logrus.Info("no SSH_AUTH_SOCK set, not using SSH agent")
		return nil
	}
	logger := logrus.WithField("SSH_AUTH_SOCK", agentAddr)
	sshAgent, err := net.Dial("unix", agentAddr)
	if err != nil {
		logger.WithError(err).Warning("unable to connect to SSH agent")
		return nil
	}
	agentClient := agent.NewClient(sshAgent)
	keys, err := agentClient.List()
	if err != nil {
		logger.WithError(err).Warning("unable to list keys in SSH agent")
		return nil
	}

	if len(keys) == 0 {
		logger.WithError(err).Warning("no keys loaded in SSH agent")
		return nil
	}

	logger.WithField("keysKnown", len(keys)).Info("using SSH agent")
	return ssh.PublicKeysCallback(agentClient.Signers)
}

// LoadSSHContext tries to find a private key to load.
func LoadSSHContext() Context {
	keyfileAuther := keyfileAuther()
	agentAuth := sshAgent()

	authMethods := []ssh.AuthMethod{}
	if keyfileAuther != ssh.AuthMethod(nil) {
		authMethods = append(authMethods, keyfileAuther)
	}
	if agentAuth != ssh.AuthMethod(nil) {
		authMethods = append(authMethods, agentAuth)
	}
	// This is also checked by the SSH library, but by checking here
	// we know in advance, instead of when we try to make the connection.
	if len(authMethods) == 0 {
		logrus.Fatal("no SSH key available")
	}

	config := &ssh.ClientConfig{
		User:            flamenco.AdminUsername,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // we don't know the hostname anyway
		Timeout:         10 * time.Second,
	}

	return Context{
		sshConfig: config,
	}
}
